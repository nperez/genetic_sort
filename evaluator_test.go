package genetic_sort

import (
	"log"
	test "testing"

	bf "nickandperla.net/brainfuck"
)

func makeEvaluatorAndUnit() (*Evaluator, *Unit) {
	evaluator := NewEvaluator(
		&EvaluatorConfig{
			MachineConfig: &bf.MachineConfig{
				MaxInstructionExecutionCount: 10000,
				MemoryCellCount:              20,
			},
			InputCellCount:  5,
			OutputCellCount: 10,
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
	rng = newPooledRand(42)
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

	if !result.MachineRun {
		t.Errorf("Unexpected MachineRun value [%v], expected 1", result.MachineRun)
	}

	if result.MachineError != nil {
		t.Errorf("Unexpected MachineError: %v\nExpected: nil", result.MachineError)
	}

	if result.SetFidelity != 80 {
		t.Errorf("Unexpected SetFidelity: [%v], expected: 80", result.SetFidelity)
	}

	if result.Sortedness != 32 {
		t.Errorf("Unexpected Sortedness: [%v], expected: 32", result.Sortedness)
	}

	if result.InstructionsExecuted != 157 {
		t.Errorf("Unexpected InstructionsExecuted: [%v], expected: 156", result.InstructionsExecuted)
	}

	log.Printf("Evaluation: %v", result)

}

func TestComputeEffectiveInputCellCount(t *test.T) {
	ec := &EvaluatorConfig{InputCellCount: 10}

	// No curriculum (InputCellStart == 0): always returns InputCellCount
	if got := ec.ComputeEffectiveInputCellCount(0); got != 10 {
		t.Errorf("Expected 10, got %d", got)
	}
	if got := ec.ComputeEffectiveInputCellCount(100); got != 10 {
		t.Errorf("Expected 10, got %d", got)
	}

	// InputCellStep == 0 but InputCellStart != 0: no growth, returns InputCellCount
	ec.InputCellStart = 2
	ec.InputCellStep = 0
	if got := ec.ComputeEffectiveInputCellCount(50); got != 10 {
		t.Errorf("Expected 10 (step=0 disables curriculum), got %d", got)
	}

	// Curriculum enabled: start=2, step=5, max=10
	ec.InputCellStart = 2
	ec.InputCellStep = 5
	// gen 0: 2 + 0/5 = 2
	if got := ec.ComputeEffectiveInputCellCount(0); got != 2 {
		t.Errorf("Gen 0: expected 2, got %d", got)
	}
	// gen 4: 2 + 4/5 = 2
	if got := ec.ComputeEffectiveInputCellCount(4); got != 2 {
		t.Errorf("Gen 4: expected 2, got %d", got)
	}
	// gen 5: 2 + 5/5 = 3
	if got := ec.ComputeEffectiveInputCellCount(5); got != 3 {
		t.Errorf("Gen 5: expected 3, got %d", got)
	}
	// gen 40: 2 + 40/5 = 10
	if got := ec.ComputeEffectiveInputCellCount(40); got != 10 {
		t.Errorf("Gen 40: expected 10, got %d", got)
	}
	// gen 100: capped at 10
	if got := ec.ComputeEffectiveInputCellCount(100); got != 10 {
		t.Errorf("Gen 100: expected 10 (capped), got %d", got)
	}
}

func TestSortedness(t *test.T) {
	unsorted := []uint8{5, 4, 3, 2, 1}

	inversions := merge_sort(unsorted)

	if inversions != 10 {
		t.Errorf("merge_sort implementation returns unexpected number of inversions on reversed list. Expected 10, got: %v", inversions)
	}

	unsorted = []uint8{1, 2, 3, 5, 4}

	inversions = merge_sort(unsorted)

	if inversions != 1 {
		t.Errorf("merge_sort implementation returns unexpected number of inversions on mostly sorted list. Expected 1, got: %v", inversions)
	}

}
