package genetic_sort

import (
	test "testing"
)

func makeSelectorConfig() *SelectorConfig {
	return &SelectorConfig{
		MachineRun:           true,
		SetFidelity:          100,
		Sortedness:           100,
		InstructionCount:     200,
		InstructionsExecuted: 10000,
	}
}

func makeEvaluation() *Evaluation {
	return &Evaluation{
		MachineRun:           true,
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
	result := s.Select(&Unit{}, makeEvaluation(), 0)

	if result != 0 {
		t.Errorf("Selector.Select() unexpected returned false: %v", result)
	}
}

func TestSelectFailures(t *test.T) {
	s := NewSelector(makeSelectorConfig())
	ev := makeEvaluation()
	ev.MachineRun = false

	if result := s.Select(&Unit{}, ev, 0); result != FailedMachineRun {
		t.Errorf("Selector.Select() unexpectedly succeeded at FailedMachineRun")
	}

	ev = makeEvaluation()
	ev.SetFidelity = 0

	if result := s.Select(&Unit{}, ev, 0); result != FailedSetFidelity {
		t.Errorf("Selector.Select() unexpectedly succeeded at FailedSetFidelity")
	}

	ev = makeEvaluation()
	ev.Sortedness = 0

	if result := s.Select(&Unit{}, ev, 0); result != FailedSortedness {
		t.Errorf("Selector.Select() unexpectedly succeeded at FailedSortedness")
	}

	ev = makeEvaluation()
	ev.InstructionCount = 10000

	if result := s.Select(&Unit{}, ev, 0); result != FailedInstructionCount {
		t.Errorf("Selector.Select() unexpectedly succeeded at FailedInstructionCount")
	}

	ev = makeEvaluation()
	ev.InstructionsExecuted = 100000

	if result := s.Select(&Unit{}, ev, 0); result != FailedInstructionsExecuted {
		t.Errorf("Selector.Select() unexpectedly succeeded at FailedInstructionsExecuted")
	}
}
