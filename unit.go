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
	Instructions   Instructions
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

	var ins Instructions = make(Instructions, instructionCount)
	for i := 0; uint(i) < instructionCount; i++ {
		ins[i] = NewInstructionFromConfig(config)
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
