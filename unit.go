package genetic_sort

import (
	"log"
)

type UnitConfig struct {
	MutationChance    float32            `toml:"mutation_chance"`
	InstructionCount  uint               `toml:"instruction_count"`
	InstructionConfig *InstructionConfig `toml:"instruction"`
	Lifespan          uint               `toml:"lifespan"`
}

type Unit struct {
	ID             uint
	PopulationID   uint
	Parent         *Unit
	ParentID       *uint
	Instructions   []*Instruction
	Age            uint
	Generation     uint
	Lifespan       uint
	MutationChance float32
	Alive          uint
	Evaluations    []*Evaluation
	Tombstone      *Tombstone
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
		Alive:          Alive,
		Lifespan:       lifeSpan,
	}
}

func (u *Unit) IncrementAge() {
	u.Age = u.Age + 1
}

func (u *Unit) CheckAge() bool {
	return u.Age < u.Lifespan
}

// Clone creates a deep copy without reflection (no copier library).
func (u *Unit) Clone() *Unit {
	clone := &Unit{
		ID:             u.ID,
		PopulationID:   u.PopulationID,
		Age:            u.Age,
		Generation:     u.Generation,
		Lifespan:       u.Lifespan,
		MutationChance: u.MutationChance,
		Alive:          u.Alive,
	}
	if u.ParentID != nil {
		pid := *u.ParentID
		clone.ParentID = &pid
	}
	clone.Instructions = make([]*Instruction, len(u.Instructions))
	for i, ins := range u.Instructions {
		clone.Instructions[i] = ins.Clone()
	}
	return clone
}

func (u *Unit) Die(reason SelectFailReason) *Tombstone {
	if reason == 0 {
		log.Fatalf("Unit needs a death reason")
	}
	if DEBUG {
		log.Printf("Killing unit [%d]", u.ID)
	}
	u.Alive = Dead
	u.Tombstone = NewTombstone(u, reason)
	return u.Tombstone
}

// Asexual reproduction. unitIDs and insIDs assign permanent IDs to the
// child and its instructions. Pass nil to leave IDs at 0.
func (u *Unit) Mitosis(unitIDs, insIDs *IDGenerator) *Unit {
	u2 := u.Clone()

	if unitIDs != nil {
		u2.ID = unitIDs.Next()
	} else {
		u2.ID = 0
	}
	u2.ParentID = &u.ID
	u2.Generation = u.Generation + 1
	u2.Age = 0
	u2.Alive = Alive
	u2.Evaluations = nil
	u2.Tombstone = nil

	for _, gene := range u2.Instructions {
		if insIDs != nil {
			gene.ID = insIDs.Next()
		} else {
			gene.ID = 0
		}
		gene.UnitID = u2.ID
		gene.Mutations = nil
		gene.IncrementAge()
		chance := rng.Float32()
		if chance < u2.MutationChance {
			NewMutation(chance).Apply(gene)
		}
	}

	return u2
}
