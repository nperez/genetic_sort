package genetic_sort

import (
	"log"
	"math"

	bf "nickandperla.net/brainfuck"
)

// An evaluation of fitness. There is a group of related metrics for a Unit
// that are used to determine survival. First, is if there is any kind of
// Machine error, then the Unit didn't survive. Second, is set fidelity where
// the intersection of input values and output values is calculated. Third is
// sortedness of the output values by measuring inversions Fourth is
// instruction execution count. An evaluation just represent a snapshot of a
// Unit's survivability. Determining if a Unit has survived is the
// responsibility of the Selector type.

type Evaluation struct {
	ID                   uint
	UnitID               uint
	MachineRun           bool
	SetFidelity          byte
	Sortedness           byte
	InstructionCount     uint
	InstructionsExecuted uint
	MachineError         *string
}

type EvaluatorConfig struct {
	MachineConfig           *bf.MachineConfig `toml:"machine"`
	InputCellCount          uint              `toml:"input_cell_count"`
	OutputCellCount         uint              `toml:"output_cell_count"`
	SynthesisInputCellCount uint              `toml:"synthesis_input_cell_count"`
	InputCellStart          uint              `toml:"input_cell_start"`
	InputCellStep           uint              `toml:"input_cell_step"`
	EvalRounds              uint              `toml:"eval_rounds"`
}

// ComputeEffectiveInputCellCount returns the input cell count to use for a
// given generation, implementing curriculum learning. If InputCellStart is 0,
// it returns InputCellCount unchanged. Otherwise it grows from InputCellStart
// by 1 every InputCellStep generations, capped at InputCellCount.
func (ec *EvaluatorConfig) ComputeEffectiveInputCellCount(generation uint) uint {
	if ec.InputCellStart == 0 || ec.InputCellStep == 0 {
		return ec.InputCellCount
	}
	effective := ec.InputCellStart + generation/ec.InputCellStep
	if effective > ec.InputCellCount {
		return ec.InputCellCount
	}
	return effective
}

type Evaluator struct {
	Machine *bf.Machine
	Config  *EvaluatorConfig
}

func NewEvaluator(ec *EvaluatorConfig) *Evaluator {
	return &Evaluator{
		Machine: bf.NewMachine(ec.MachineConfig),
		Config:  ec,
	}
}

func (e *Evaluator) Evaluate(u *Unit) *Evaluation {
	eval := &Evaluation{
		UnitID: u.ID,
	}

	input := makeRandomInput(e.Config.InputCellCount)
	e.Machine.LoadProgram(Instructions(u.Instructions).ToProgram())
	if ok, err := e.Machine.LoadMemory(input); !ok {
		log.Fatalf("Failed to load memory into machine. %v", err)
	}

	if ok, err := e.Machine.Run(); !ok {
		if err != nil {
			var msg string = err.Error()
			eval.MachineError = &msg
		}
	} else {
		eval.MachineRun = true
	}

	ok, output, err := e.Machine.ReadMemory(e.Config.OutputCellCount)

	if !ok {
		log.Fatalf("Failed to read memory. Check MachineConfig.MemoryConfig.CellCount and EvaluatorConfig.OutputCellCount. %v", err)
	}

	copyOutput := make([]uint8, len(output))
	copy(copyOutput, output)

	inMap := make(map[uint8]bool)
	outMap := make(map[uint8]bool)

	for g := 0; g < len(input); g++ {
		inMap[input[g]] = true
		outMap[output[g]] = true
	}

	total, count := 0, 0
	for k, _ := range inMap {
		total++
		if _, ok := outMap[k]; ok {
			count++
		}
	}
	eval.SetFidelity = byte(uint(float32(count) / float32(total) * 100))

	inversions := merge_sort(copyOutput)
	maxInversions := uint(len(copyOutput) * (len(copyOutput) - 1) / 2)

	if DEBUG {
		log.Printf("Inversions: %v\nMax Inversions: %v", inversions, maxInversions)
	}

	eval.Sortedness = byte(-(int((float32(inversions)/float32(maxInversions))*100) - 100))
	eval.InstructionsExecuted = e.Machine.InstructionCount
	eval.InstructionCount = uint(len(Instructions(u.Instructions).ToProgram()))

	u.Evaluations = append(u.Evaluations, eval)

	return eval
}

// Fitness returns a composite fitness score for ranking during synthesis.
// MachineRun is not gated — a program that timed out may still have
// done useful work in memory.
func (e *Evaluation) Fitness() uint {
	return uint(e.SetFidelity) + uint(e.Sortedness)
}

// EvaluateWithCellCounts runs evaluation using the given input/output cell
// counts instead of the ones from Config. Used during synthesis with
// smaller inputs.
func (e *Evaluator) EvaluateWithCellCounts(u *Unit, inputCells, outputCells uint) *Evaluation {
	eval := &Evaluation{
		UnitID: u.ID,
	}

	input := makeRandomInput(inputCells)
	e.Machine.LoadProgram(Instructions(u.Instructions).ToProgram())
	if ok, err := e.Machine.LoadMemory(input); !ok {
		log.Fatalf("Failed to load memory into machine. %v", err)
	}

	if ok, err := e.Machine.Run(); !ok {
		if err != nil {
			var msg string = err.Error()
			eval.MachineError = &msg
		}
	} else {
		eval.MachineRun = true
	}

	ok, output, err := e.Machine.ReadMemory(outputCells)

	if !ok {
		log.Fatalf("Failed to read memory. Check MachineConfig.MemoryConfig.CellCount and EvaluatorConfig.OutputCellCount. %v", err)
	}

	copyOutput := make([]uint8, len(output))
	copy(copyOutput, output)

	inMap := make(map[uint8]bool)
	outMap := make(map[uint8]bool)

	for g := 0; g < len(input); g++ {
		inMap[input[g]] = true
		outMap[output[g]] = true
	}

	total, count := 0, 0
	for k, _ := range inMap {
		total++
		if _, ok := outMap[k]; ok {
			count++
		}
	}
	rawFidelity := float32(count) / float32(total) * 100

	inversions := merge_sort(copyOutput)
	maxInversions := uint(len(copyOutput) * (len(copyOutput) - 1) / 2)

	if DEBUG {
		log.Printf("Inversions: %v\nMax Inversions: %v", inversions, maxInversions)
	}

	rawSortedness := float32(-(int((float32(inversions)/float32(maxInversions))*100) - 100))

	// Scale scores by inputCells/maxInputCells so difficulty reflects input size
	scale := float32(inputCells) / float32(e.Config.InputCellCount)
	eval.SetFidelity = byte(uint(rawFidelity * scale))
	eval.Sortedness = byte(uint(rawSortedness * scale))
	eval.InstructionsExecuted = e.Machine.InstructionCount
	eval.InstructionCount = uint(len(Instructions(u.Instructions).ToProgram()))

	u.Evaluations = append(u.Evaluations, eval)

	return eval
}

// EvaluateMultiRound runs EvalRounds evaluations with different random inputs
// and keeps only the worst result (lowest Fitness). Forces programs to be
// robust across inputs rather than getting lucky on one.
func (e *Evaluator) EvaluateMultiRound(u *Unit, rounds uint) *Evaluation {
	return e.evaluateMultiRound(u, rounds, e.Config.InputCellCount, e.Config.OutputCellCount)
}

// EvaluateMultiRoundWithCellCounts is like EvaluateMultiRound but with custom cell counts.
func (e *Evaluator) EvaluateMultiRoundWithCellCounts(u *Unit, rounds, inputCells, outputCells uint) *Evaluation {
	return e.evaluateMultiRound(u, rounds, inputCells, outputCells)
}

func (e *Evaluator) evaluateMultiRound(u *Unit, rounds, inputCells, outputCells uint) *Evaluation {
	var worst *Evaluation
	var worstFitness uint = math.MaxUint64

	program := Instructions(u.Instructions).ToProgram()
	instrCount := uint(len(program))

	for r := uint(0); r < rounds; r++ {
		eval := &Evaluation{UnitID: u.ID}

		input := makeRandomInput(inputCells)
		e.Machine.LoadProgram(program)
		if ok, err := e.Machine.LoadMemory(input); !ok {
			log.Fatalf("Failed to load memory into machine. %v", err)
		}

		if ok, err := e.Machine.Run(); !ok {
			if err != nil {
				var msg string = err.Error()
				eval.MachineError = &msg
			}
		} else {
			eval.MachineRun = true
		}

		ok, output, err := e.Machine.ReadMemory(outputCells)
		if !ok {
			log.Fatalf("Failed to read memory. %v", err)
		}

		copyOutput := make([]uint8, len(output))
		copy(copyOutput, output)

		inMap := make(map[uint8]bool)
		outMap := make(map[uint8]bool)
		for g := 0; g < len(input); g++ {
			inMap[input[g]] = true
			outMap[output[g]] = true
		}

		total, count := 0, 0
		for k := range inMap {
			total++
			if _, ok := outMap[k]; ok {
				count++
			}
		}
		rawFidelity := float32(count) / float32(total) * 100

		inversions := merge_sort(copyOutput)
		maxInversions := uint(len(copyOutput) * (len(copyOutput) - 1) / 2)
		rawSortedness := float32(-(int((float32(inversions)/float32(maxInversions))*100) - 100))

		// Scale scores by inputCells/maxInputCells so difficulty reflects input size
		scale := float32(inputCells) / float32(e.Config.InputCellCount)
		eval.SetFidelity = byte(uint(rawFidelity * scale))
		eval.Sortedness = byte(uint(rawSortedness * scale))
		eval.InstructionsExecuted = e.Machine.InstructionCount
		eval.InstructionCount = instrCount

		fitness := eval.Fitness()
		if fitness < worstFitness {
			worstFitness = fitness
			worst = eval
		}
	}

	u.Evaluations = append(u.Evaluations, worst)
	return worst
}

func makeRandomInput(count uint) []uint8 {
	ret := make([]uint8, count)
	for i := uint(0); i < count; i++ {
		ret[i] = uint8(rng.Intn(int(math.MaxUint8)))
	}
	return ret
}

func merge(a []uint8, inversion0 uint) uint {

	inversion1 := uint(0)

	copyLeft := uint(0)
	copyRight := uint(len(a) / 2)

	for copyLeft < copyRight && copyRight < uint(len(a)) {
		if a[copyLeft] <= a[copyRight] {
			copyLeft++
		} else {
			copyRight++
			inversion1 += uint(len(a)/2) - copyLeft
		}
	}

	return inversion0 + inversion1
}

func merge_sort(a []uint8) uint {
	inversions := uint(0)
	if len(a) > 1 {
		mid := len(a) / 2
		reply := make(chan uint)
		go func() {
			reply <- merge_sort(a[mid:])
		}()
		inv1 := merge_sort(a[:mid])
		inv2 := <-reply
		return merge(a, inv1+inv2)
	}
	return inversions
}
