package genetic_sort

import (
	"log"
	rnd "math/rand"
	test "testing"

	bf "nickandperla.net/brainfuck"
)

func makeEvaluatorAndUnit() (*Evaluator, *Unit) {
	evaluator := NewEvaluator(
		&EvaluatorConfig{
			MachineConfig: &bf.MachineConfig{
				MaxInstructionExecutionCount: 10000,
				MemoryConfig: &bf.MemoryConfig{
					CellCount:  20,
					LowerBound: 0,
					UpperBound: 255,
				},
			},
			InputCellCount:      5,
			InputCellUpperBound: 255,
			OutputCellCount:     10,
		},
	)

	unit := NewUnitFromConfig(&UnitConfig{
		MutationChance:    0.25,
		InstructionCount:  1,
		InstructionConfig: &InstructionConfig{OpSetCount: 2},
	})
	return evaluator, unit
}

func TestNewEvaluator(t *test.T) {
	rnd.Seed(42)
	evaluator, unit := makeEvaluatorAndUnit()

	result := evaluator.Evaluate(unit)
	log.Printf("Evaluation: %v", result)
}

func TestSortedness(t *test.T) {
	unsorted := []uint{5, 4, 3, 2, 1}

	inversions := merge_sort(unsorted)

	if inversions != 10 {
		t.Errorf("merge_sort implementation returns unexpected number of inversions on reversed list. Expected 10, got: %v", inversions)
	}

	unsorted = []uint{1, 2, 3, 5, 4}

	inversions = merge_sort(unsorted)

	if inversions != 1 {
		t.Errorf("merge_sort implementation returns unexpected number of inversions on mostly sorted list. Expected 1, got: %v", inversions)
	}

}
