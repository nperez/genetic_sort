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

	// Curriculum fields: if set, thresholds ramp from *Start to SetFidelity/Sortedness
	// over generations, increasing by 1 every *Step generations.
	SetFidelityStart byte `toml:"set_fidelity_start"`
	SetFidelityStep  uint `toml:"set_fidelity_step"`
	SortednessStart  byte `toml:"sortedness_start"`
	SortednessStep   uint `toml:"sortedness_step"`
}

func NewSelector(config *SelectorConfig) *Selector {
	return &Selector{Config: config}
}

// EffectiveSetFidelity returns the set fidelity threshold for a given generation.
// If SetFidelityStart is 0, returns SetFidelity unchanged.
func (c *SelectorConfig) EffectiveSetFidelity(generation uint) byte {
	if c.SetFidelityStart == 0 || c.SetFidelityStep == 0 {
		return c.SetFidelity
	}
	effective := uint(c.SetFidelityStart) + generation/c.SetFidelityStep
	if effective > uint(c.SetFidelity) {
		return c.SetFidelity
	}
	return byte(effective)
}

// EffectiveSortedness returns the sortedness threshold for a given generation.
// If SortednessStart is 0, returns Sortedness unchanged.
func (c *SelectorConfig) EffectiveSortedness(generation uint) byte {
	if c.SortednessStart == 0 || c.SortednessStep == 0 {
		return c.Sortedness
	}
	effective := uint(c.SortednessStart) + generation/c.SortednessStep
	if effective > uint(c.Sortedness) {
		return c.Sortedness
	}
	return byte(effective)
}

type SelectFailReason uint

func (s *Selector) Select(u *Unit, e *Evaluation, generation uint) SelectFailReason {

	if s.Config.MachineRun && !e.MachineRun {
		return FailedMachineRun
	}
	if e.SetFidelity < s.Config.EffectiveSetFidelity(generation) {
		return FailedSetFidelity
	}
	if e.Sortedness < s.Config.EffectiveSortedness(generation) {
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
