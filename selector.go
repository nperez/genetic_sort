package genetic_sort

type Selector struct {
	Config *SelectorConfig
}

type SelectorConfig struct {
	MachineRun           byte
	SetFidelity          byte
	Sortedness           byte
	InstructionCount     uint
	InstructionsExecuted uint
}

func NewSelector(config *SelectorConfig) *Selector {
	return &Selector{Config: config}
}

func (s *Selector) Select(u *Unit, e *Evaluation) bool {
	var selected bool = true
	if e.MachineRun != s.Config.MachineRun {
		selected = false
	}
	return selected

}
