package genetic_sort

import (
	test "testing"
)

func TestCompareEvaluationsDefaultPriority(t *test.T) {
	ranker := NewFitnessRanker(nil)

	a := &Evaluation{Sortedness: 80, SetFidelity: 50, InstructionsExecuted: 100}
	b := &Evaluation{Sortedness: 60, SetFidelity: 90, InstructionsExecuted: 50}

	// Default: sortedness is primary, so a (80) beats b (60)
	result := ranker.CompareEvaluations(a, b)
	if result != -1 {
		t.Errorf("Expected a to be better (sortedness 80 > 60), got %d", result)
	}
}

func TestCompareEvaluationsTiedPrimary(t *test.T) {
	ranker := NewFitnessRanker(nil)

	a := &Evaluation{Sortedness: 70, SetFidelity: 80, InstructionsExecuted: 100}
	b := &Evaluation{Sortedness: 70, SetFidelity: 60, InstructionsExecuted: 50}

	// Tied on sortedness, falls to set_fidelity: a (80) beats b (60)
	result := ranker.CompareEvaluations(a, b)
	if result != -1 {
		t.Errorf("Expected a to be better (set_fidelity 80 > 60), got %d", result)
	}
}

func TestCompareEvaluationsEfficiencyTiebreaker(t *test.T) {
	ranker := NewFitnessRanker(nil)

	a := &Evaluation{Sortedness: 70, SetFidelity: 80, InstructionsExecuted: 50}
	b := &Evaluation{Sortedness: 70, SetFidelity: 80, InstructionsExecuted: 100}

	// Tied on sortedness and set_fidelity, efficiency: a (50) beats b (100) — lower is better
	result := ranker.CompareEvaluations(a, b)
	if result != -1 {
		t.Errorf("Expected a to be better (efficiency 50 < 100), got %d", result)
	}
}

func TestCompareEvaluationsFullTie(t *test.T) {
	ranker := NewFitnessRanker(nil)

	a := &Evaluation{Sortedness: 70, SetFidelity: 80, InstructionsExecuted: 100}
	b := &Evaluation{Sortedness: 70, SetFidelity: 80, InstructionsExecuted: 100}

	result := ranker.CompareEvaluations(a, b)
	if result != 0 {
		t.Errorf("Expected tie (0), got %d", result)
	}
}

func TestCompareEvaluationsCustomPriority(t *test.T) {
	// Make efficiency primary, sortedness secondary, skip set_fidelity
	ranker := NewFitnessRanker(&FitnessConfig{
		SortednessPriority:  2,
		SetFidelityPriority: 0,
		EfficiencyPriority:  1,
	})

	a := &Evaluation{Sortedness: 60, SetFidelity: 90, InstructionsExecuted: 50}
	b := &Evaluation{Sortedness: 80, SetFidelity: 10, InstructionsExecuted: 100}

	// Efficiency is primary: a (50) beats b (100)
	result := ranker.CompareEvaluations(a, b)
	if result != -1 {
		t.Errorf("Expected a to be better (efficiency 50 < 100), got %d", result)
	}
}

func TestCompareEvaluationsCustomPriorityFallthrough(t *test.T) {
	// Efficiency primary, sortedness secondary
	ranker := NewFitnessRanker(&FitnessConfig{
		SortednessPriority:  2,
		SetFidelityPriority: 0,
		EfficiencyPriority:  1,
	})

	a := &Evaluation{Sortedness: 60, SetFidelity: 90, InstructionsExecuted: 100}
	b := &Evaluation{Sortedness: 80, SetFidelity: 10, InstructionsExecuted: 100}

	// Tied on efficiency, falls to sortedness: b (80) beats a (60)
	result := ranker.CompareEvaluations(a, b)
	if result != 1 {
		t.Errorf("Expected b to be better (sortedness 80 > 60), got %d", result)
	}
}
