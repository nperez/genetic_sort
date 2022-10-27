package genetic_sort

import (
	"log"
	"math"
	"math/rand"

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
	Input                []uint8 `gorm:"type:blob"`
	Output               []uint8 `gorm:"type:blob"`
}

type EvaluatorConfig struct {
	MachineConfig   *bf.MachineConfig `gorm:"embedded" toml:"machine"`
	InputCellCount  uint              `toml:"input_cell_count"`
	OutputCellCount uint              `toml:"output_cell_count"`
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
	if u.ID == 0 {
		log.Fatalf("Unit must be persisted prior to evaluation")
	}

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
	eval.Input = input
	eval.Output = output
	eval.InstructionsExecuted = e.Machine.InstructionCount
	eval.InstructionCount = uint(len(Instructions(u.Instructions).ToProgram()))

	u.Evaluations = append(u.Evaluations, eval)

	return eval
}

func makeRandomInput(count uint) []uint8 {
	ret := make([]uint8, count)
	for i := uint(0); i < count; i++ {
		ret[i] = uint8(rand.Intn(int(math.MaxUint8)))
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
