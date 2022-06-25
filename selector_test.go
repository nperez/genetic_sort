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
	result, err := s.Select(&Unit{}, makeEvaluation())

	if !result {
		t.Errorf("Selector.Select() unexpected returned false: %v", err)
	}
}

func TestSelectFailures(t *test.T) {
	s := NewSelector(makeSelectorConfig())
	ev := makeEvaluation()
	ev.MachineRun = 0

	if result, err := s.Select(&Unit{}, ev); result != false || err != FailedMachineRun {
		t.Errorf("Selector.Select() unexpectedly succeeded at FailedMachineRun")
	}

	ev = makeEvaluation()
	ev.SetFidelity = 0

	if result, err := s.Select(&Unit{}, ev); result != false || err != FailedSetFidelity {
		t.Errorf("Selector.Select() unexpectedly succeeded at FailedSetFidelity")
	}

	ev = makeEvaluation()
	ev.Sortedness = 0

	if result, err := s.Select(&Unit{}, ev); result != false || err != FailedSortedness {
		t.Errorf("Selector.Select() unexpectedly succeeded at FailedSortedness")
	}

	ev = makeEvaluation()
	ev.InstructionCount = 10000

	if result, err := s.Select(&Unit{}, ev); result != false || err != FailedInstructionCount {
		t.Errorf("Selector.Select() unexpectedly succeeded at FailedInstructionCount")
	}

	ev = makeEvaluation()
	ev.InstructionsExecuted = 100000

	if result, err := s.Select(&Unit{}, ev); result != false || err != FailedInstructionsExecuted {
		t.Errorf("Selector.Select() unexpectedly succeeded at FailedInstructionsExecuted")
	}
}
