package genetic_sort

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
)

// "math/rand"

type Population struct {
	ID               uint
	Units            []*Unit
	PopulationConfig *PopulationConfig `gorm:"embedded"`
	persist          *Persistence      `gorm:"-:all"`
}

type PopulationConfig struct {
	UnitCount       uint             `toml:"unit_count"`
	UnitConfig      *UnitConfig      `gorm:"embedded" toml:"unit"`
	EvaluatorConfig *EvaluatorConfig `gorm:"embedded" toml:"eval"`
	SelectorConfig  *SelectorConfig  `gorm:"embedded" toml:"select"`
}

func NewPopulationFromConfig(config *PopulationConfig) *Population {
	return &Population{
		PopulationConfig: config,
	}
}

func (p *Population) SynthesizeUnits() {
	count := p.PopulationConfig.UnitCount
	cpus := uint(runtime.NumCPU())
	odds := count % cpus
	split := uint(count / cpus)

	var wg sync.WaitGroup

	for i := uint(0); i < cpus; i++ {
		i := i
		if i == cpus-1 {
			split = split + odds
		}
		wg.Add(1)
		go func(id, max, batchSize uint) {
			defer wg.Done()
			if DEBUG {
				log.Printf("Synthesizer %d generating %d with batch size %d\n", id, max, batchSize)
			}
			batch := make([]*Unit, batchSize)
			bid := uint(0)
			for q := uint(0); q < max; q++ {
				unit := NewUnitFromConfig(p.PopulationConfig.UnitConfig)
				unit.PopulationID = p.ID
				batch[bid] = unit
				bid++
				if bid == batchSize {
					if err := p.persist.SaveUnits(&batch); err != nil {
						panic(fmt.Errorf("Saving new Units failed! %w", err))
					}
					for i := range batch {
						batch[i] = nil
					}
					bid = 0
				}
			}
			if bid != 0 {
				sub := batch[:bid]
				if err := p.persist.SaveUnits(&sub); err != nil {
					panic(fmt.Errorf("Saving new Units failed! %w", err))
				}
			}
		}(i, split, p.persist.Config.BatchSize)
	}

	wg.Wait()
}

func (p *Population) GetAliveCount() (count uint) {
	var dbcnt int64
	p.persist.DB.Model(&Unit{}).Where("alive = 1 and population_id = ?", p.ID).Count(&dbcnt)
	count = uint(dbcnt)
	return
}

func (p *Population) ProcessGeneration() {

	loaders := p.persist.GetUnitLoaders(p, p.persist.Config.BatchSize)
	persistor := p.persist.GetUnitPersistor()
	evaluator := NewEvaluator(p.PopulationConfig.EvaluatorConfig)
	selector := NewSelector(p.PopulationConfig.SelectorConfig)
	engine := NewGenerationEngine(loaders, persistor, evaluator, selector)

	ctx, _ := context.WithCancel(context.Background())
	engine.Run(ctx)
}
