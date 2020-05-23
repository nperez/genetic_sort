package genetic_sort

import "math/rand"

type Population struct {
	Units          []*Unit
	Survivors      []*Unit
	Dead           []*Unit
	MutationChance float32
	Fitness        *Fitness
}

type PopulationConfig struct {
	UnitCount      int
	UnitConfig     *UnitConfig
	MutationChange float32
	Fitness        *Fitness
}

func NewPopulationFromConfig(config *PopulationConfig) *Population {
	units := make([]*Unit, config.UnitCount)

	for i := 0; i < unitCount; i++ {
		units[i] = NewUnitFromConfig(config.UnitConfig)
	}

	return &Population{
		Units:          units,
		Survivors:      make([]*Unit, unitCount),
		Dead:           make([]*Unit, unitCount),
		MutationChance: config.MutationChange,
		Fitness:        config.Fitness,
	}
}

// The Breed function has this algorithm:
//  Take the survivors and randomize them
//  Split the survivors into two groups
//  Sort each group by age, ascending (Very slight breeding preference to younger survivors)
//  Pair off the survivors (odd survivor out does not breed, but is still kept)
//  Compare the genes for each pair
//  If the genes in the same position are a close match according levenstein distance [0,1]:
//      Flip a coin and copy the one of the genes into the new collection
//  Otherwise if they are not a close match:
//      Compare the age, taking the older of the two (preference for repeat success)
//      And if they are the same age, flip another coin
//  If one of the survivors has more genes, those are appened to the end
//  Iterate all of the gene:
//      Increment the age
//      Check mutation chance and apply mutation if true
//  Create new memeber of the population with copied genes

// Breed - Take the successful units and create new amalgam units to add to the population
func (p *Population) Breed() {
	for i := len(p.Survivors) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		p.Survivors[i], p.Survivors[j] = p.Survivors[j], p.Survivors[i]
	}
	groupOne := p.Survivors[:len(p.Survivors)/2-1]
	groupTwo := p.Survivors[(len(p.Survivors)/2-1)+1:]

	if len(groupOne) < len(groupTwo) {
		for groupIndex, survivor1 := range groupOne {
			survivor2 := groupTwo[groupIndex]
			newUnit := survivor1.Breed(survivor2, p.MutationChance)
			p.Units = append(p.Units, newUnit, survivor1, survivor2)
		}
		p.Units = append(p.Units, groupTwo[len(groupOne)-1:])
	} else if len(groupOne) > len(groupTwo) {
		for groupIndex, survivor2 := range groupTwo {
			survivor1 := groupOne[groupIndex]
			newUnit := survivor2.Breed(survivor1, p.MutationChance)
			p.Units = append(p.Units, newUnit, survivor2, survivor1)
		}
		p.Units = append(p.Units, groupOne[len(groupTwo)-1:])
	} else {
		for groupIndex, survivor2 := range groupTwo {
			survivor1 := groupOne[groupIndex]
			newUnit := survivor2.Breed(survivor1, p.MutationChance)
			p.Units = append(p.Units, newUnit, survivor2, survivor1)
		}
	}

	p.Survivors = make([]*Unit, 0)
}

// ProcessGeneration - Iterate the current units and check their fitness
func (p *Population) ProcessGeneration() {
	for unit := range p.Units {
		currentReport = unit.CurrentReport()

		p.Fitness.Process(currentReport)

		if currentReport.Alive {
			p.Survivors = append(p.Survivors, unit)
		} else {
			p.Dead = append(p.Dead, unit)
		}
	}

	p.Units = make([]*Unit, 0)
}
