package genetic_sort

import (
	test "testing"
)

func makeSelectorConfig() *SelectorConfig {
	return &SelectorConfig{
		MachineRun:           1,
		SetFidelity:          100,
		Sortedness:           100,
		InstructionCount:     200,
		InstructionsExecuted: 10000,
	}
}

func makeEvaluation() *Evaluation {
	return &Evaluation{
		MachineRun:           1,
		SetFidelity:          100,
		Sortedness:           100,
		InstructionCount:     200,
		InstructionsExecuted: 10000,
	}
}

func TestNewSelector(t *test.T) {
	s := NewSelector(makeSelectorConfig())
	if s == nil {
		t.Errorf("NewSelector returned nil")
	}
}

func TestSimpleSelect(t *test.T) {
	s := NewSelector(makeSelectorConfig())
	result := s.Select(&Unit{}, makeEvaluation())

	if !result {
		t.Errorf("Selector.Select() unexpected returned false")
	}
}
