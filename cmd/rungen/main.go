package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"time"

	"nickandperla.net/genetic_sort"

	"github.com/BurntSushi/toml"
)

var toolConfigPath *string = flag.String("config", "./config.toml", "The config file for genetic_sort tools to use. Defaults to './config.toml'")

var popId *uint = flag.Uint("popid", 1, "The id of the population you'd like to progress one generation")
var generations *uint = flag.Uint("generations", 1, "Number of generations to process")
var cpuprofile *string = flag.String("cpuprofile", "", "Write CPU profile to file")
var evalRounds *uint = flag.Uint("eval-rounds", 0, "Override eval rounds per unit (0 = use DB config)")
var streaming *bool = flag.Bool("streaming", true, "Use streaming mode (low memory, DB I/O per batch)")

func main() {
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatalf("Could not create CPU profile: %v", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("Could not start CPU profile: %v", err)
		}
		defer pprof.StopCPUProfile()
	}

	conffile, err := os.Open(*toolConfigPath)

	if err != nil {
		log.Fatalf("Unable to load genetic_sort config: %v", err)
	}

	confDecoder := toml.NewDecoder(conffile)
	var toolConfig genetic_sort.ToolConfig
	if _, err = confDecoder.Decode(&toolConfig); err != nil {
		log.Fatalf("Failed to unmarshal tool config: %v", err)
	}
	conffile.Close()

	genetic_sort.InitRNG(toolConfig.Persistence.Seed)

	persist, err := genetic_sort.NewPersistence(toolConfig.Persistence)
	if err != nil {
		log.Fatalf("Failed to create or initialize Persistence: %v", err)
	}
	defer persist.Shutdown()

	pop, err := persist.LoadShallow(*popId)
	if err != nil {
		log.Fatalf("Unable to load population from DB: %v", err)
	}

	// Override eval rounds if specified on command line
	if *evalRounds > 0 {
		pop.PopulationConfig.EvaluatorConfig.EvalRounds = *evalRounds
		log.Printf("Overriding eval_rounds to %d", *evalRounds)
	}

	if *streaming {
		// Streaming mode: process in batches, persist per-batch
		log.Printf("Using streaming mode (eval_batch_size: %d)", toolConfig.Persistence.EvalBatchSize)
		for gen := uint(1); gen <= *generations; gen++ {
			if err := pop.ProcessGenerationStreaming(); err != nil {
				log.Fatalf("Generation %d failed: %v", gen, err)
			}
			alive := pop.GetAliveCount()
			if alive == 0 {
				log.Fatalf("Population went extinct at generation %d", gen)
			}
			fmt.Printf("Population [%d] generation %d complete: %d alive\n", pop.ID, pop.CurrentGeneration, alive)
		}
	} else {
		// In-memory mode: load everything, process, persist at end
		log.Printf("Loading population %d from database...", pop.ID)
		loadStart := time.Now()
		units, err := pop.LoadUnitsIntoMemory()
		if err != nil {
			log.Fatalf("Failed to load units: %v", err)
		}
		log.Printf("Loaded %d units in %v", len(units), time.Since(loadStart))

		if len(units) == 0 {
			log.Fatalf("Population [%d] has no living units", pop.ID)
		}

		for gen := uint(1); gen <= *generations; gen++ {
			units = pop.ProcessGenerationInMemory(units)
			if len(units) == 0 {
				log.Fatalf("Population went extinct at generation %d", gen)
			}
		}

		// Persist final state
		log.Printf("Persisting %d units to database...", len(units))
		persistStart := time.Now()
		if err := pop.PersistPopulationState(units); err != nil {
			log.Fatalf("Failed to persist: %v", err)
		}
		log.Printf("Persisted in %v", time.Since(persistStart))

		fmt.Printf("Population [%d] unit count: %d\n", pop.ID, len(units))
	}
}
