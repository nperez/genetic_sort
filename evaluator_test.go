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
	evaluator, _ := makeEvaluatorAndUnit()

	if evaluator == nil {
		t.Errorf("NewEvaluator() returned nil")
	}
}

func TestEvaluate(t *test.T) {
	rnd.Seed(42)
	evaluator, unit := makeEvaluatorAndUnit()

	if evaluator == nil {
		t.Errorf("NewEvaluator() returned nil")
	}

	result := evaluator.Evaluate(unit)

	if result == nil {
		t.Errorf("Evaluator.Evaluate() returned nil")
	}

	if result.ID != 0 {
		t.Errorf("Unexpected Evaluation.ID: %v", result.ID)
	}

	if result.UnitID != unit.ID {
		t.Errorf("Evaluation.UnitID [%v] doesn't match unit.ID [%v]",
			result.UnitID, unit.ID)
	}

	if result.MachineRun != 1 {
		t.Errorf("Unexpected MachineRun value [%v], expected 1", result.MachineRun)
	}

	if result.MachineError != nil {
		t.Errorf("Unexpected MachineError: %v\nExpected: nil", result.MachineError)
	}

	if result.SetFidelity != 80 {
		t.Errorf("Unexpected SetFidelity: [%v], expected: 80", result.SetFidelity)
	}

	if result.Sortedness != 38 {
		t.Errorf("Unexpected Sortedness: [%v], expected: 38", result.Sortedness)
	}

	if result.InstructionsExecuted != 156 {
		t.Errorf("Unexpected InstructionsExecuted: [%v], expected: 156", result.InstructionsExecuted)
	}

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
