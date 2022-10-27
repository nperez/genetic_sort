package genetic_sort

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

type SelectFailReason uint

func (s *Selector) Select(u *Unit, e *Evaluation) SelectFailReason {

	if e.MachineRun != s.Config.MachineRun {
		return FailedMachineRun
	}
	if e.SetFidelity < s.Config.SetFidelity {
		return FailedSetFidelity
	}
	if e.Sortedness < s.Config.Sortedness {
		return FailedSortedness
	}
	if e.InstructionCount > s.Config.InstructionCount {
		return FailedInstructionCount
	}
	if e.InstructionsExecuted > s.Config.InstructionsExecuted {
		return FailedInstructionsExecuted
	}
	return 0

}
