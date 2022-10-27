package genetic_sort

import (
	"context"
	"log"
)

type Processor struct {
	Input     UnitLoader
	Output    chan<- *Unit
	Evaluator *Evaluator
	Selector  *Selector
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
			for _, unit := range units {
				eval := p.Evaluator.Evaluate(unit)
				reason := p.Selector.Select(unit, eval)
				if reason != 0 {
					unit.Die(reason)
				}
				p.Output <- unit
			}
		case <-ctx.Done():
			break FOR
		}
	}
}

func NewProcessor(loader UnitLoader, output chan<- *Unit, evaluator *Evaluator, selector *Selector) *Processor {
	return &Processor{
		Input:     loader,
		Evaluator: evaluator,
		Selector:  selector,
		Output:    output,
	}
}
