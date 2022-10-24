package genetic_sort

import (
	"fmt"
)

type Selector struct {
	Config *SelectorConfig
}

type SelectorConfig struct {
	MachineRun           bool `toml:"machine_run"`
	SetFidelity          byte `toml:"set_fidelity"`
	Sortedness           byte `toml:"sortedness"`
	InstructionCount     uint `toml:"instruction_count"`
	InstructionsExecuted uint `toml:"instructions_executed"`
}

func NewSelector(config *SelectorConfig) *Selector {
	return &Selector{Config: config}
}

var FailedMachineRun error = fmt.Errorf("FailedMachineRun")
var FailedSetFidelity error = fmt.Errorf("FailedSetFidelity")
var FailedSortedness error = fmt.Errorf("FailedSortedness")
var FailedInstructionCount error = fmt.Errorf("FailedInstructionCount")
var FailedInstructionsExecuted error = fmt.Errorf("FailedInstructionsExecuted")

func (s *Selector) Select(u *Unit, e *Evaluation) (bool, error) {

	if e.MachineRun != s.Config.MachineRun {
		return false, FailedMachineRun
	}
	if e.SetFidelity < s.Config.SetFidelity {
		return false, FailedSetFidelity
	}
	if e.Sortedness < s.Config.Sortedness {
		return false, FailedSortedness
	}
	if e.InstructionCount > s.Config.InstructionCount {
		return false, FailedInstructionCount
	}
	if e.InstructionsExecuted > s.Config.InstructionsExecuted {
		return false, FailedInstructionsExecuted
	}
	return true, nil

}
