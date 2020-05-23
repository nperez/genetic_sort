package genetic_sort

type GenerationReport struct {
	Input        []int
	Output       []int
	Exception    error
	OpsExecuted  int64
	CellsVisited int64
	FitnessScore *FitnessScore
}
