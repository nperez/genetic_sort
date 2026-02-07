package genetic_sort

import (
	"context"
	"log"
)

type Processor struct {
	Input      UnitLoader
	Persistor  UnitPersistor
	Evaluator  *Evaluator
	Selector   *Selector
	Generation uint
}

func (p *Processor) Run(ctx context.Context, id, total uint) {
	input := p.Input(id, total)
FOR:
	for {
		select {
		case units := <-input:
			if units == nil {
				if DEBUG {
					log.Printf("Closing processor %d", id)
				}
				break FOR
			}
			var offspring []*Unit
			for _, unit := range units {
				eval := p.Evaluator.Evaluate(unit)
				reason := p.Selector.Select(unit, eval, p.Generation)
				if reason != 0 {
					unit.Die(reason)
					continue
				}
				// Survived — age and reproduce
				unit.IncrementAge()
				if !unit.CheckAge() {
					unit.Die(FailedLifespan)
					continue
				}
				offspring = append(offspring, unit.Mitosis(nil, nil))
			}
			p.Persistor(&units)
			if len(offspring) > 0 {
				p.Persistor(&offspring)
			}
		case <-ctx.Done():
			break FOR
		}
	}
}

func NewProcessor(loader UnitLoader, persistor UnitPersistor, evaluator *Evaluator, selector *Selector) *Processor {
	return &Processor{
		Input:     loader,
		Evaluator: evaluator,
		Selector:  selector,
		Persistor: persistor,
	}
}

// EvalProcessor evaluates units and applies threshold selection but does NOT
// reproduce. Instead of persisting per-batch, it collects all evaluated units
// into Results for bulk persistence after all evaluation is complete.
type EvalProcessor struct {
	Input           UnitLoader
	Evaluator       *Evaluator
	Selector        *Selector
	InputCellCount  uint // 0 means use evaluator defaults
	OutputCellCount uint
	Generation      uint
	Results         []*Unit
}

func (p *EvalProcessor) Run(ctx context.Context, id, total uint) {
	input := p.Input(id, total)
FOR:
	for {
		select {
		case units := <-input:
			if units == nil {
				if DEBUG {
					log.Printf("Closing eval processor %d", id)
				}
				break FOR
			}
			for _, unit := range units {
				var eval *Evaluation
				if p.InputCellCount > 0 {
					eval = p.Evaluator.EvaluateWithCellCounts(unit, p.InputCellCount, p.OutputCellCount)
				} else {
					eval = p.Evaluator.Evaluate(unit)
				}
				reason := p.Selector.Select(unit, eval, p.Generation)
				if reason != 0 {
					unit.Die(reason)
					continue
				}
				unit.IncrementAge()
				if !unit.CheckAge() {
					unit.Die(FailedLifespan)
				}
			}
			p.Results = append(p.Results, units...)
		case <-ctx.Done():
			break FOR
		}
	}
}

func NewEvalProcessor(loader UnitLoader, evalConfig *EvaluatorConfig, selector *Selector) *EvalProcessor {
	return &EvalProcessor{
		Input:     loader,
		Evaluator: NewEvaluator(evalConfig),
		Selector:  selector,
	}
}

func NewEvalProcessorWithCells(loader UnitLoader, evalConfig *EvaluatorConfig, selector *Selector, inputCells, outputCells uint) *EvalProcessor {
	return &EvalProcessor{
		Input:           loader,
		Evaluator:       NewEvaluator(evalConfig),
		Selector:        selector,
		InputCellCount:  inputCells,
		OutputCellCount: outputCells,
	}
}
