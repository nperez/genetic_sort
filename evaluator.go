package genetic_sort

import (
	"fmt"
	"log"
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
	MachineRun           byte
	SetFidelity          byte
	Sortedness           byte
	InstructionCount     uint
	InstructionsExecuted uint
	MachineError         *string
	Input                []byte `gorm:"type:blob"`
	Output               []byte `gorm:"type:blob"`
}

type EvaluatorConfig struct {
	MachineConfig       *bf.MachineConfig
	InputCellCount      uint
	InputCellUpperBound uint
	OutputCellCount     uint
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

	eval := &Evaluation{}

	input := makeRandomInput(e.Config.InputCellCount, e.Config.InputCellUpperBound)
	e.Machine.LoadProgram(u.Instructions.ToProgram())
	if ok, err := e.Machine.LoadMemory(input); !ok {
		panic(fmt.Errorf("Failed to load memory into machine. Check MachineConfig.MemoryConfig.UpperBound and EvaluatorConfig.InputCellUpperBound. %v", err))
	}

	if ok, err := e.Machine.Run(); !ok {
		if err != nil {
			var msg string = err.Error()
			eval.MachineError = &msg
		}
	} else {
		eval.MachineRun = 1
	}

	ok, output, err := e.Machine.ReadMemory(e.Config.OutputCellCount)

	if !ok {
		panic(fmt.Errorf("Failed to read memory. Check MachineConfig.MemoryConfig.CellCount and EvaluatorConfig.OutputCellCount. %v", err))
	}

	copyOutput := make([]uint, len(output))
	copy(copyOutput, output)

	inMap := make(map[uint]bool)
	outMap := make(map[uint]bool)

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
	eval.Input = makeTruncated(input)
	eval.Output = makeTruncated(output)
	eval.InstructionsExecuted = e.Machine.InstructionCount
	eval.InstructionCount = uint(len(u.Instructions.ToProgram()))

	return eval
}

func makeRandomInput(count, upperbound uint) []uint {
	ret := make([]uint, count)
	for i := uint(0); i < count; i++ {
		ret[i] = uint(rand.Intn(int(upperbound)))
	}
	return ret
}

func makeTruncated(stuff []uint) []byte {
	ret := make([]byte, len(stuff))
	for i := 0; i < len(stuff); i++ {
		ret[i] = byte(stuff[i])
	}
	return ret
}

func merge(a []uint, inversion0 uint) uint {

	inversion1 := uint(0)

	c := make([]uint, len(a))
	copy(c, a)

	copyLeft := uint(0)
	copyRight := uint(len(c) / 2)
	current := uint(0)

	for copyLeft <= copyRight-1 && copyRight <= uint(len(c)-1) {
		if c[copyLeft] <= c[copyRight] {
			a[current] = c[copyLeft]
			copyLeft++
		} else {
			a[current] = c[copyRight]
			copyRight++
			inversion1 += uint(len(c)/2) - copyLeft
		}
		current++
	}

	for copyLeft <= copyRight-1 && current <= uint(len(c)-1) {
		a[current] = c[copyLeft]
		current++
		copyLeft++
	}

	return inversion0 + inversion1
}

func merge_sort(a []uint) uint {
	inversions := uint(0)
	if len(a) > 1 {
		mid := len(a) / 2
		reply := make(chan uint, 0)
		go func() {
			reply <- merge_sort(a[mid:])
		}()
		inv1 := merge_sort(a[:mid])
		inv2 := <-reply
		return merge(a, inv1+inv2)
	}
	return inversions
}
