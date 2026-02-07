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
		p := processor
		go func(id, total uint) {
			defer wg.Done()
			p.Run(ctx, id, total)
		}(uint(i), count)
	}

	wg.Wait()
}

// EvalGenerationEngine runs EvalProcessors in parallel — evaluate and
// threshold-select only, no reproduction or persistence. Results are
// collected in memory for bulk persistence afterward.
type EvalGenerationEngine struct {
	Processors []*EvalProcessor
}

func NewEvalGenerationEngine(loaders []UnitLoader, evalConfig *EvaluatorConfig, selector *Selector) *EvalGenerationEngine {
	processors := make([]*EvalProcessor, len(loaders))
	for i := range loaders {
		processors[i] = NewEvalProcessor(loaders[i], evalConfig, selector)
	}
	return &EvalGenerationEngine{
		Processors: processors,
	}
}

func NewEvalGenerationEngineWithCells(loaders []UnitLoader, evalConfig *EvaluatorConfig, selector *Selector, inputCells, outputCells uint) *EvalGenerationEngine {
	processors := make([]*EvalProcessor, len(loaders))
	for i := range loaders {
		processors[i] = NewEvalProcessorWithCells(loaders[i], evalConfig, selector, inputCells, outputCells)
	}
	return &EvalGenerationEngine{
		Processors: processors,
	}
}

func (ge *EvalGenerationEngine) Run(ctx context.Context) {
	var wg sync.WaitGroup
	count := uint(len(ge.Processors))
	for i, processor := range ge.Processors {
		wg.Add(1)
		p := processor
		go func(id, total uint) {
			defer wg.Done()
			p.Run(ctx, id, total)
		}(uint(i), count)
	}

	wg.Wait()
}

// CollectResults gathers all evaluated units from all processors.
func (ge *EvalGenerationEngine) CollectResults() []*Unit {
	var all []*Unit
	for _, p := range ge.Processors {
		all = append(all, p.Results...)
	}
	return all
}
