package genetic_sort

import (
	"math/rand"

	"github.com/xrash/smetrics"
)

type UnitConfig struct {
	InstructionCount  int
	InstructionConfig *InstructionConfig
}

type Unit struct {
	Parents      []*Unit
	Instructions Instructions
	Reports      []*GenerationReport
}

func NewUnitFromBreed(i Instructions, p []*Unit) *Unit {
	return &Unit{
		Parents:      p,
		Instructions: i,
	}
}

func NewUnitFromConfig(config *UnitConfig) *Unit {
	return NewUnitFromRandom(config.InstructionCount, config.InstructionConfig)
}

func NewUnitFromRandom(instructionCount, config *InstructionConfig) *Unit {
	var ins Instructions
	for i := 0; i < instructionCount; i++ {
		ins = append(ins, NewInstructionFromConfig(config))
	}

	return &Unit{
		Instructions: ins,
	}
}

func (u *Unit) Age() int {
	return len(u.Reports)
}

func (u *Unit) AddReport(g *GenerationReport) {
	u.Reports = append(u.Reports, g)
}

func (u *Unit) CurrentReport() *GenerationReport {
	return u.Reports[:len(u.Reports)-1]
}

func (u *Unit) Breed(p *Unit, mutationChance float32) *Unit {
	instruct1, instruct2 := u.Instructions, p.Instructions
	var left, right Instructions
	if len(instruct1) == len(instruct2) {
		left, right = instruct1, instruct2
	} else if len(instruct1) > len(instruct2) {
		left, right = instruct1, instruct2
	} else {
		right, left = instruct1, instruct2
	}

	var copiedGenes Instructions

	for i2, gene1 := range right {
		gene2 := left[i2]

		if smetrics.WagnerFischer(gene1, gene2, 1, 1, 1) <= 1 {
			if rand.Intn(2) == 0 {
				copiedGenes = append(copiedGenes, gene1)
			} else {
				copiedGenes = append(copiedGenes, gene2)
			}
		} else {
			if gene1.Age > gene2.Age {
				copiedGenes = append(copiedGenes, gene1)
			} else if gene2.Age > gene1.Age {
				copiedGenes = append(copiedGenes, gene2)
			} else {
				if rand.Intn(2) == 0 {
					copiedGenes = append(copiedGenes, gene1)
				}
				copiedGenes = append(copiedGenes, gene2)
			}
		}
	}

	for gene := range copiedGenes {
		gene.IncrementAge()
		if rand.Intn(100)/100 < mutationChance {
			NewInstructionMutation().Apply(gene)
		}
	}

	return NewUnitFromBreed(copiedGenes, []*Unit{u, p})
}
