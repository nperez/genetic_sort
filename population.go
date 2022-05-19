package genetic_sort

import (
//"math/rand"
)

type Population struct {
	ID             uint
	Units          []*Unit
	MutationChance float32
}

type PopulationConfig struct {
	UnitCount      uint
	UnitConfig     *UnitConfig
	MutationChance float32
}

func NewPopulationFromConfig(config *PopulationConfig) *Population {
	units := make([]*Unit, config.UnitCount)

	for i := uint(0); i < config.UnitCount; i++ {
		units[i] = NewUnitFromConfig(config.UnitConfig)
	}

	return &Population{
		Units:          units,
		MutationChance: config.MutationChance,
	}
}

// ProcessGeneration - Iterate the current units and check their fitness
func (p *Population) ProcessGeneration() {
}
