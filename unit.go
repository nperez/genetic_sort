package genetic_sort

import (
	"math/rand"

	// "github.com/xrash/smetrics"
	cp "github.com/jinzhu/copier"
)

type UnitConfig struct {
	MutationChance    float32
	InstructionCount  uint
	InstructionConfig *InstructionConfig
	Lifespan          uint
}

type Unit struct {
	ID             uint
	PopulationID   uint
	Parent         *Unit
	ParentID       *uint
	Instructions   []*Instruction
	Age            uint
	Lifespan       uint
	MutationChance float32
	Alive          bool
}

func NewUnitFromConfig(config *UnitConfig) *Unit {
	return NewUnitFromRandom(
		config.MutationChance,
		config.InstructionCount,
		config.InstructionConfig,
		config.Lifespan)
}

func NewUnitFromRandom(
	mutationChance float32,
	instructionCount uint,
	config *InstructionConfig,
	lifeSpan uint) *Unit {

	var ins []*Instruction
	for i := 0; uint(i) < instructionCount; i++ {
		ins = append(ins, NewInstructionFromConfig(config))
	}

	return &Unit{
		Instructions:   ins,
		MutationChance: mutationChance,
		Alive:          true,
		Lifespan:       lifeSpan,
	}
}

func (u *Unit) IncrementAge() {
	u.Age = u.Age + 1
}

func (u *Unit) CheckAge() bool {
	return u.Age < u.Lifespan
}

func (u *Unit) Clone() *Unit {
	clone := &Unit{}
	cp.Copy(clone, u)
	return clone
}

// Asexual reproduction is phase one
func (u *Unit) Mitosis() *Unit {
	u2 := u.Clone()
	u2.Parent = u
	for _, gene := range u2.Instructions {
		gene.IncrementAge()
		chance := rand.Float32()
		if chance < u2.MutationChance {
			NewMutation(chance).Apply(gene)
		}
	}

	return u2
}

// Sexual repoduction should be a phase two
//func (u *Unit) Breed(p *Unit, mutationChance float32) *Unit {
//
//	// Extract this out into a function that can be influenced via config? That
//	// way we can see if this creates selection pressure, maybe?
//	instruct1, instruct2 := u.Instructions, p.Instructions
//	var left, right Instructions
//	if len(instruct1) == len(instruct2) {
//		left, right = instruct1, instruct2
//	} else if len(instruct1) > len(instruct2) {
//		left, right = instruct1, instruct2
//	} else {
//		right, left = instruct1, instruct2
//	}
//
//	var copiedGenes Instructions
//
//	for i2, gene1 := range right {
//		gene2 := left[i2]
//
//		// Same here. I feel like, Breed() should really execute a series of
//		// evaluations and selections that are tunable and maybe even
//		// inheritble.
//		if smetrics.WagnerFischer(gene1.String(), gene2.String(), 1, 1, 2) <= 1 {
//			if rand.Intn(2) == 0 {
//				copiedGenes = append(copiedGenes, gene1)
//			} else {
//				copiedGenes = append(copiedGenes, gene2)
//			}
//		} else {
//			if gene1.Age > gene2.Age {
//				copiedGenes = append(copiedGenes, gene1)
//			} else if gene2.Age > gene1.Age {
//				copiedGenes = append(copiedGenes, gene2)
//			} else {
//				if rand.Intn(2) == 0 {
//					copiedGenes = append(copiedGenes, gene1)
//				}
//				copiedGenes = append(copiedGenes, gene2)
//			}
//		}
//	}
//
//	for _, gene := range copiedGenes {
//		gene.IncrementAge()
//		if float32(rand.Intn(100)/100) < mutationChance {
//			NewMutation().Apply(gene)
//		}
//	}
//
//	return NewUnitFromBreed(copiedGenes, []*Unit{u, p})
//}
