package genetic_sort

// "math/rand"

type Population struct {
	ID               uint
	Units            []*Unit
	PopulationConfig *PopulationConfig `gorm:"embedded"`
}

type PopulationConfig struct {
	UnitCount       uint             `toml:"unit_count"`
	UnitConfig      *UnitConfig      `gorm:"embedded" toml:"unit"`
	EvaluatorConfig *EvaluatorConfig `gorm:"embedded" toml:"eval"`
	SelectorConfig  *SelectorConfig  `gorm:"embedded" toml:"select"`
}

func NewPopulationFromConfig(config *PopulationConfig) *Population {
	units := make([]*Unit, config.UnitCount)

	for i := uint(0); i < config.UnitCount; i++ {
		units[i] = NewUnitFromConfig(config.UnitConfig)
	}

	return &Population{
		Units:            units,
		PopulationConfig: config,
	}
}

// ProcessGeneration - Iterate the current units and check their fitness
func (p *Population) ProcessGeneration() {
}
