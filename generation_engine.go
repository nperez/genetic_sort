package genetic_sort

import (
	"context"
	"sync"
)

type GenerationEngine struct {
	Processors []*Processor
}

func NewGenerationEngine(loaders []UnitLoader, persistor UnitPersistor, evaluator *Evaluator, selector *Selector) *GenerationEngine {
	processors := make([]*Processor, len(loaders))
	for i, loader := range loaders {
		processors[i] = NewProcessor(loader, persistor, evaluator, selector)
	}
	return &GenerationEngine{
		Processors: processors,
	}
}

func (ge *GenerationEngine) Run(ctx context.Context) {
	var wg sync.WaitGroup
	count := uint(len(ge.Processors))
	for i, processor := range ge.Processors {
		wg.Add(1)
		go func(id, total uint) {
			defer wg.Done()
			processor.Run(ctx, id, total)
		}(uint(i), count)
	}

	wg.Wait()
}
