package genetic_sort

type VeldtConfig struct {
	PopulationCount  int
	PopulationConfig *PopulationConfig
	//MemoryConfig     *MemoryConfig
}

type Veldt struct {
	Populations []*Population
	//Memory      *Memory
}

func (v *Veldt) Process(input []int) {

}

func (v *Veldt) Execute() {
}
