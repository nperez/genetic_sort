package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"nickandperla.net/genetic_sort"

	"github.com/BurntSushi/toml"
)

var (
	configPath    = flag.String("config", "./config.toml", "Tool config path")
	popConfigPath = flag.String("popconfig", "./pop.toml", "Population config path (starting params)")
	trials        = flag.Int("trials", 1, "Number of trials (1 = single run, >1 = LLM optimization)")
	genCap        = flag.Uint("gen-cap", 500, "Max generations per trial")
	check         = flag.Uint("check", 10, "Generations between metric checks")
	ollamaURL     = flag.String("ollama", "http://localhost:11434/api/generate", "Ollama API endpoint")
	model         = flag.String("model", "gemma3:4b-it-qat", "Ollama model name")
	stagnation    = flag.Uint("stagnation", 5, "Consecutive check intervals with no improvement before aborting (0 = disabled)")
)

type TrialResult struct {
	Params         genetic_sort.PopulationConfig
	Outcome        string // "success", "extinct", "timeout", "stagnant", "synthesis_failed"
	GenerationsRun uint
	BestSortedness byte
	BestFidelity   byte
	AliveAtEnd     uint
	Program        string // BF program if success
	GenCap         uint
	AtCapChecks    uint   // how many metric checks alive was >= 90% of carrying capacity
	TotalChecks    uint   // total metric checks performed
}

func main() {
	flag.Parse()
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Ltime)

	// Load tool config
	conffile, err := os.Open(*configPath)
	if err != nil {
		log.Fatalf("Unable to load tool config: %v", err)
	}
	var toolConfig genetic_sort.ToolConfig
	if _, err = toml.NewDecoder(conffile).Decode(&toolConfig); err != nil {
		log.Fatalf("Failed to unmarshal tool config: %v", err)
	}
	conffile.Close()

	// Load starting population config
	popConfig := loadPopConfig(*popConfigPath)

	currentGenCap := *genCap
	var history []TrialResult

	for trial := 0; trial < *trials; trial++ {
		log.Printf("========== TRIAL %d/%d ==========", trial+1, *trials)
		logPopConfig(&popConfig)
		log.Printf("  gen_cap=%d check_interval=%d stagnation=%d", currentGenCap, *check, *stagnation)

		result := runTrial(&toolConfig, &popConfig, currentGenCap, *check, *stagnation)
		result.GenCap = currentGenCap
		history = append(history, result)

		switch result.Outcome {
		case "success":
			log.Printf("SUCCESS at generation %d!", result.GenerationsRun)
			log.Printf("Best sortedness=%d fidelity=%d alive=%d",
				result.BestSortedness, result.BestFidelity, result.AliveAtEnd)
			// Print winning program to stdout
			if result.Program != "" {
				fmt.Println(result.Program)
			}
		case "synthesis_failed":
			log.Printf("SYNTHESIS FAILED — could not generate viable units with current params")
		case "extinct":
			log.Printf("EXTINCT at generation %d", result.GenerationsRun)
		case "stagnant":
			log.Printf("STAGNANT at generation %d", result.GenerationsRun)
			log.Printf("Best sortedness=%d fidelity=%d alive=%d",
				result.BestSortedness, result.BestFidelity, result.AliveAtEnd)
		case "timeout":
			log.Printf("TIMEOUT after %d generations (cap=%d)", result.GenerationsRun, currentGenCap)
			log.Printf("Best sortedness=%d fidelity=%d alive=%d",
				result.BestSortedness, result.BestFidelity, result.AliveAtEnd)
		}

		// If success on single-trial mode or last trial, stop
		if result.Outcome == "success" || trial == *trials-1 {
			break
		}

		// Consult LLM for next parameters
		log.Println("Consulting LLM for next parameters...")
		llmStart := time.Now()
		response, err := askOllama(*ollamaURL, *model, buildPrompt(history))
		if err != nil {
			log.Printf("LLM error: %v — keeping current params", err)
			continue
		}
		log.Printf("LLM responded in %v", time.Since(llmStart))

		jsonStr := extractJSON(response)
		var adj ParamAdjustments
		if err := json.Unmarshal([]byte(jsonStr), &adj); err != nil {
			log.Printf("Failed to parse LLM JSON: %v", err)
			log.Printf("Raw response: %s", response)
			log.Println("Keeping current params")
			continue
		}

		log.Printf("LLM reasoning: %s", adj.Reasoning)
		applyAdjustments(&popConfig, adj, &currentGenCap)
	}

	// Summary to stderr
	log.Println("========== OPTIMIZATION SUMMARY ==========")
	var bestIdx int
	for i, r := range history {
		marker := "  "
		if r.Outcome == "success" {
			marker = "* "
			bestIdx = i
		}
		log.Printf("%sTrial %2d: outcome=%s gens=%d sortedness=%d fidelity=%d alive=%d gen_cap=%d at_cap=%d/%d",
			marker, i+1, r.Outcome, r.GenerationsRun, r.BestSortedness, r.BestFidelity, r.AliveAtEnd, r.GenCap, r.AtCapChecks, r.TotalChecks)
	}

	// Find best trial — prefer success, then highest sortedness+fidelity
	if history[bestIdx].Outcome != "success" {
		bestScore := 0
		for i, r := range history {
			score := int(r.BestSortedness) + int(r.BestFidelity)
			if score > bestScore {
				bestScore = score
				bestIdx = i
			}
		}
	}
	log.Printf("Best trial: #%d", bestIdx+1)
}

func runTrial(toolConfig *genetic_sort.ToolConfig, popConfig *genetic_sort.PopulationConfig, genCap uint, checkInterval uint, stagnationLimit uint) TrialResult {
	result := TrialResult{
		Params: *popConfig,
	}

	genetic_sort.InitRNG(toolConfig.Persistence.Seed)

	persist, err := genetic_sort.NewPersistence(toolConfig.Persistence)
	if err != nil {
		log.Printf("Failed to create persistence: %v", err)
		result.Outcome = "extinct"
		return result
	}
	defer persist.Shutdown()

	// Create population and synthesize
	pop, err := persist.Create(popConfig)
	if err != nil {
		log.Printf("Failed to create population: %v", err)
		result.Outcome = "extinct"
		return result
	}
	log.Printf("Population %d created, synthesizing %d units...", pop.ID, popConfig.UnitCount)

	synthStart := time.Now()
	if err := pop.SynthesizeUnits(); err != nil {
		log.Printf("Synthesis failed: %v", err)
		result.Outcome = "synthesis_failed"
		return result
	}
	log.Printf("Synthesis complete in %v", time.Since(synthStart))

	// Stagnation tracking
	var bestComposite int
	var stagnationCounter uint

	// Population collapse threshold: 1% of carrying capacity or 100, whichever is smaller
	collapseThreshold := popConfig.CarryingCapacity / 100
	if collapseThreshold > 100 {
		collapseThreshold = 100
	}
	if collapseThreshold < 1 {
		collapseThreshold = 1
	}

	// Generation loop
	for gen := uint(1); gen <= genCap; gen++ {
		if err := pop.ProcessGenerationStreaming(); err != nil {
			log.Printf("Generation %d failed: %v", gen, err)
			result.Outcome = "extinct"
			result.GenerationsRun = gen
			return result
		}

		alive := pop.GetAliveCount()
		if alive == 0 {
			log.Printf("Population went extinct at generation %d", gen)
			result.Outcome = "extinct"
			result.GenerationsRun = gen
			return result
		}

		// Periodic metric check
		if gen%checkInterval == 0 || gen == genCap {
			metrics, err := pop.QueryMetrics()
			if err != nil {
				log.Printf("Warning: failed to query metrics at gen %d: %v", gen, err)
				continue
			}

			effectiveInput := popConfig.EvaluatorConfig.ComputeEffectiveInputCellCount(pop.CurrentGeneration)
			log.Printf("  Gen %d: alive=%d effective_input=%d best_sort=%d best_fid=%d avg_sort=%.1f avg_fid=%.1f",
				gen, metrics.AliveCount, effectiveInput,
				metrics.BestSortedness, metrics.BestSetFidelity,
				metrics.AvgSortedness, metrics.AvgSetFidelity)

			result.BestSortedness = metrics.BestSortedness
			result.BestFidelity = metrics.BestSetFidelity
			result.AliveAtEnd = metrics.AliveCount
			result.TotalChecks++

			// Track how often we're at carrying capacity (>= 90%)
			if metrics.AliveCount >= popConfig.CarryingCapacity*9/10 {
				result.AtCapChecks++
			}

			// Stagnation detection
			if stagnationLimit > 0 {
				composite := int(metrics.BestSortedness) + int(metrics.BestSetFidelity)
				if composite > bestComposite {
					bestComposite = composite
					stagnationCounter = 0
				} else {
					stagnationCounter++
				}

				// Population collapse check
				if metrics.AliveCount < collapseThreshold {
					log.Printf("Population collapsing (alive=%d < threshold=%d), aborting trial", metrics.AliveCount, collapseThreshold)
					result.Outcome = "stagnant"
					result.GenerationsRun = gen
					return result
				}

				if stagnationCounter >= stagnationLimit {
					log.Printf("Stagnation detected: no improvement in %d consecutive checks, aborting trial", stagnationCounter)
					result.Outcome = "stagnant"
					result.GenerationsRun = gen
					return result
				}
			}

			// Check victory condition: effective_input >= 10 AND best scores are 100/100
			if effectiveInput >= 10 && metrics.BestSortedness == 100 && metrics.BestSetFidelity == 100 {
				log.Printf("Potential winner found at gen %d! Verifying...", gen)

				bestUnit, _, err := pop.QueryBestUnit()
				if err != nil {
					log.Printf("Warning: failed to query best unit: %v", err)
					continue
				}
				if bestUnit == nil {
					continue
				}

				program := genetic_sort.Instructions(bestUnit.Instructions).ToProgram()
				if verifyPerfectSorter(popConfig, bestUnit, 20) {
					log.Printf("VERIFIED: Perfect 10-item sorter found!")
					result.Outcome = "success"
					result.GenerationsRun = gen
					result.Program = program
					return result
				}
				log.Printf("Verification failed — not a consistent sorter")
			}
		}

		result.GenerationsRun = gen
	}

	result.Outcome = "timeout"
	return result
}

// verifyPerfectSorter re-evaluates the unit with numTrials fresh random
// 10-cell inputs. All must score sortedness=100 AND set_fidelity=100.
func verifyPerfectSorter(config *genetic_sort.PopulationConfig, unit *genetic_sort.Unit, numTrials int) bool {
	evaluator := genetic_sort.NewEvaluator(config.EvaluatorConfig)
	for i := 0; i < numTrials; i++ {
		// Clone the unit so evaluations don't accumulate
		clone := unit.Clone()
		eval := evaluator.EvaluateWithCellCounts(clone, 10, 10)
		if eval.Sortedness != 100 || eval.SetFidelity != 100 {
			log.Printf("  Verification trial %d: sortedness=%d fidelity=%d — FAIL",
				i+1, eval.Sortedness, eval.SetFidelity)
			return false
		}
	}
	return true
}

func loadPopConfig(path string) genetic_sort.PopulationConfig {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("Unable to load population config: %v", err)
	}
	defer f.Close()

	var config genetic_sort.PopulationConfig
	if _, err := toml.NewDecoder(f).Decode(&config); err != nil {
		log.Fatalf("Failed to unmarshal population config: %v", err)
	}
	return config
}

func logPopConfig(c *genetic_sort.PopulationConfig) {
	log.Printf("  unit_count=%d carrying_capacity=%d elitism=%d max_offspring=%d",
		c.UnitCount, c.CarryingCapacity, c.Elitism, c.MaxOffspring)
	if c.UnitConfig != nil {
		log.Printf("  lifespan=%d mutation_chance=%.4f instruction_count=%d",
			c.UnitConfig.Lifespan, c.UnitConfig.MutationChance, c.UnitConfig.InstructionCount)
	}
	if c.EvaluatorConfig != nil {
		log.Printf("  eval_rounds=%d input_cells=%d synthesis_input_cells=%d",
			c.EvaluatorConfig.EvalRounds, c.EvaluatorConfig.InputCellCount, c.EvaluatorConfig.SynthesisInputCellCount)
	}
	if c.SelectorConfig != nil {
		log.Printf("  sel: fidelity=%d sortedness=%d ins_count=%d ins_exec=%d",
			c.SelectorConfig.SetFidelity, c.SelectorConfig.Sortedness,
			c.SelectorConfig.InstructionCount, c.SelectorConfig.InstructionsExecuted)
	}
}

// --- LLM Integration ---

type ollamaReq struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}
type ollamaResp struct {
	Response string `json:"response"`
}

func askOllama(url, modelName, prompt string) (string, error) {
	body, _ := json.Marshal(ollamaReq{Model: modelName, Prompt: prompt, Stream: false})
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var r ollamaResp
	if err := json.Unmarshal(data, &r); err != nil {
		return "", fmt.Errorf("parse error: %w\nraw: %s", err, string(data))
	}
	return r.Response, nil
}

func extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end <= start {
		return s
	}
	return s[start : end+1]
}

type ParamAdjustments struct {
	MaxOffspring               *uint    `json:"max_offspring"`
	MutationChance             *float64 `json:"mutation_chance"`
	Elitism                    *uint    `json:"elitism"`
	EvalRounds                 *uint    `json:"eval_rounds"`
	SelectSetFidelity          *uint    `json:"select_set_fidelity"`
	SelectSortedness           *uint    `json:"select_sortedness"`
	SelectInstructionCount     *uint    `json:"select_instruction_count"`
	SelectInstructionsExecuted *uint    `json:"select_instructions_executed"`
	InstructionCount           *uint    `json:"instruction_count"`
	OpSetCount                 *uint    `json:"op_set_count"`
	Lifespan                   *uint    `json:"lifespan"`
	UnitCount                  *uint    `json:"unit_count"`
	GenCap                     *uint    `json:"gen_cap"`
	CarryingCapacity           *uint    `json:"carrying_capacity"`
	Reasoning                  string   `json:"reasoning"`
}

func clamp(v, lo, hi uint) uint {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func applyAdjustments(p *genetic_sort.PopulationConfig, adj ParamAdjustments, genCap *uint) {
	if adj.MaxOffspring != nil {
		p.MaxOffspring = clamp(*adj.MaxOffspring, 1, 20)
	}
	if adj.MutationChance != nil {
		p.UnitConfig.MutationChance = float32(clampFloat(*adj.MutationChance, 0.01, 0.50))
	}
	if adj.Elitism != nil {
		p.Elitism = clamp(*adj.Elitism, 10, 50000)
	}
	if adj.EvalRounds != nil {
		p.EvaluatorConfig.EvalRounds = clamp(*adj.EvalRounds, 1, 5)
	}
	if adj.SelectSetFidelity != nil {
		p.SelectorConfig.SetFidelity = byte(clamp(*adj.SelectSetFidelity, 0, 100))
	}
	if adj.SelectSortedness != nil {
		p.SelectorConfig.Sortedness = byte(clamp(*adj.SelectSortedness, 0, 100))
	}
	if adj.SelectInstructionCount != nil {
		p.SelectorConfig.InstructionCount = clamp(*adj.SelectInstructionCount, 100, 50000)
	}
	if adj.SelectInstructionsExecuted != nil {
		p.SelectorConfig.InstructionsExecuted = clamp(*adj.SelectInstructionsExecuted, 1000, 100000)
	}
	if adj.InstructionCount != nil {
		p.UnitConfig.InstructionCount = clamp(*adj.InstructionCount, 5, 50)
	}
	if adj.OpSetCount != nil {
		p.UnitConfig.InstructionConfig.OpSetCount = int(clamp(*adj.OpSetCount, 5, 50))
	}
	if adj.Lifespan != nil {
		p.UnitConfig.Lifespan = clamp(*adj.Lifespan, 10, 1000)
	}
	if adj.UnitCount != nil {
		p.UnitCount = clamp(*adj.UnitCount, 500, 100000)
	}
	if adj.CarryingCapacity != nil {
		p.CarryingCapacity = clamp(*adj.CarryingCapacity, 1000, 10000000)
	}
	if adj.GenCap != nil {
		*genCap = clamp(*adj.GenCap, 50, 5000)
	}
}

func buildPrompt(history []TrialResult) string {
	var sb strings.Builder

	sb.WriteString(`You are tuning a genetic algorithm that evolves Brainfuck programs to sort 10-item arrays of uint8.

GOAL: Find starting parameters that produce a PERFECT 10-item sorter in the fewest generations.
A perfect sorter scores sortedness=100 AND set_fidelity=100 on 20 random 10-cell inputs.

The algorithm uses curriculum learning: input size starts small and grows over generations.
Programs must work reliably, not just get lucky on one input.

TUNABLE PARAMETERS (with safe ranges):
- max_offspring (1-20): Children per survivor. Higher = faster recovery from culling.
- mutation_chance (0.01-0.50): Mutation probability per instruction per generation.
- elitism (10-50000): Top N immune from competitive culling.
- eval_rounds (1-5): Random inputs tested per unit (score = worst case). Higher = harder.
- select_set_fidelity (0-100): Min output/input overlap threshold.
- select_sortedness (0-100): Min sortedness threshold.
- select_instruction_count (100-50000): Max program length.
- select_instructions_executed (1000-100000): Max runtime instructions.
- instruction_count (5-50): Instruction segments per unit.
- op_set_count (5-50): Ops per segment.
- lifespan (10-1000): Max generations a unit lives.
- unit_count (500-100000): Starting population size.
- carrying_capacity (1000-10000000): Max alive units after culling.
- gen_cap (50-5000): Max generations before timeout.

KEY DYNAMICS:
- If survival_rate * (1 + avg_offspring) < 1, population shrinks every generation.
- eval_rounds=1 is much easier to survive than eval_rounds=3.
- Larger populations explore more of the search space.
- Lower selection thresholds let weaker units survive and evolve.
- Curriculum learning grows input size gradually: start small, increase over generations.
- at_capacity=X/Y means X out of Y metric checks the population was at >=90% carrying capacity. High values mean selection pressure is too low — units survive too easily and you're just capping growth mechanically.

`)

	if len(history) > 0 {
		sb.WriteString("PREVIOUS TRIALS:\n")
		for i, trial := range history {
			fmt.Fprintf(&sb, "  Trial %d: outcome=%s gens=%d gen_cap=%d best_sort=%d best_fid=%d alive=%d at_capacity=%d/%d\n",
				i+1, trial.Outcome, trial.GenerationsRun, trial.GenCap,
				trial.BestSortedness, trial.BestFidelity, trial.AliveAtEnd,
				trial.AtCapChecks, trial.TotalChecks)
			fmt.Fprintf(&sb, "    params: unit_count=%d max_offspring=%d mutation=%.2f elitism=%d eval_rounds=%d carrying_cap=%d\n",
				trial.Params.UnitCount, trial.Params.MaxOffspring,
				trial.Params.UnitConfig.MutationChance, trial.Params.Elitism,
				trial.Params.EvaluatorConfig.EvalRounds, trial.Params.CarryingCapacity)
			fmt.Fprintf(&sb, "    sel: fidelity=%d sortedness=%d ins_count=%d ins_exec=%d\n",
				trial.Params.SelectorConfig.SetFidelity, trial.Params.SelectorConfig.Sortedness,
				trial.Params.SelectorConfig.InstructionCount, trial.Params.SelectorConfig.InstructionsExecuted)
		}
		sb.WriteString("\n")
	}

	sb.WriteString(`Based on the trial results, suggest NEW parameter values for the next trial.
Rules:
1. Change 2-5 parameters at a time, not everything.
2. If population collapsed (extinct), prioritize increasing max_offspring, reducing eval_rounds, or relaxing selection thresholds.
3. If the trial was stagnant (aborted for lack of progress or population collapse), the parameters need more significant changes — consider relaxing thresholds, increasing diversity, or changing mutation rates.
4. If synthesis_failed, the selection thresholds or eval_rounds are too harsh for random programs to pass. Drastically reduce select_set_fidelity, select_sortedness, reduce eval_rounds to 1, or increase instruction_count/op_set_count to give programs more capacity.
5. If population survived but didn't find a sorter, try increasing gen_cap or tightening selection gradually.
6. If at_capacity is high (e.g. 8/10 checks at capacity), selection is too lenient — the population is coasting. Tighten select_set_fidelity, select_sortedness, or increase eval_rounds to apply real evolutionary pressure.
7. null means keep current value.

Respond with ONLY valid JSON, no markdown fences, no commentary outside the object:
{"reasoning":"...","max_offspring":null,"mutation_chance":null,"elitism":null,"eval_rounds":null,"select_set_fidelity":null,"select_sortedness":null,"select_instruction_count":null,"select_instructions_executed":null,"instruction_count":null,"op_set_count":null,"lifespan":null,"unit_count":null,"carrying_capacity":null,"gen_cap":null}
`)

	return sb.String()
}
