package genetic_sort

import "sort"

type FitnessConfig struct {
	SortednessPriority  uint `toml:"sortedness_priority"`
	SetFidelityPriority uint `toml:"set_fidelity_priority"`
	EfficiencyPriority  uint `toml:"efficiency_priority"`
}

type FitnessRanker struct {
	Config *FitnessConfig
}

func NewFitnessRanker(config *FitnessConfig) *FitnessRanker {
	return &FitnessRanker{Config: config}
}

// priorityKeys returns the comparison order based on priority values.
// Lower priority number = compared first. Priority 0 means skip.
// Returns a slice of metric identifiers in comparison order.
// Metric IDs: 1=sortedness, 2=set_fidelity, 3=efficiency
func (fr *FitnessRanker) priorityKeys() []uint {
	type pv struct {
		metric   uint
		priority uint
	}

	config := fr.Config
	if config == nil {
		config = &FitnessConfig{}
	}

	entries := []pv{
		{1, config.SortednessPriority},
		{2, config.SetFidelityPriority},
		{3, config.EfficiencyPriority},
	}

	// If all zeros, use defaults: sortedness=1, set_fidelity=2, efficiency=3
	allZero := true
	for _, e := range entries {
		if e.priority != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		return []uint{1, 2, 3}
	}

	// Filter out zeros (skipped metrics) and sort by priority ascending
	var active []pv
	for _, e := range entries {
		if e.priority != 0 {
			active = append(active, e)
		}
	}
	sort.Slice(active, func(i, j int) bool {
		return active[i].priority < active[j].priority
	})

	keys := make([]uint, len(active))
	for i, a := range active {
		keys[i] = a.metric
	}
	return keys
}

// CompareEvaluations returns -1 if a is better, +1 if b is better, 0 if tied.
// Sortedness/SetFidelity: higher is better. Efficiency: lower is better.
func (fr *FitnessRanker) CompareEvaluations(a, b *Evaluation) int {
	keys := fr.priorityKeys()

	for _, metric := range keys {
		switch metric {
		case 1: // sortedness — higher is better
			if a.Sortedness > b.Sortedness {
				return -1
			}
			if a.Sortedness < b.Sortedness {
				return 1
			}
		case 2: // set_fidelity — higher is better
			if a.SetFidelity > b.SetFidelity {
				return -1
			}
			if a.SetFidelity < b.SetFidelity {
				return 1
			}
		case 3: // efficiency — lower is better
			if a.InstructionsExecuted < b.InstructionsExecuted {
				return -1
			}
			if a.InstructionsExecuted > b.InstructionsExecuted {
				return 1
			}
		}
	}

	return 0
}
