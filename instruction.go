package genetic_sort

import (
	"math/rand"
	"strings"

	copier "github.com/jinzhu/copier"
)

type InstructionConfig struct {
	OpCount int
}

type Instruction struct {
	Ops          []OP
	Mutations    []Mutation
	InitialOpSet []OP
	Age          int
}

func (i *Instruction) String() string {
	var b strings.Builder
	for _, op := range i.Ops {
		b.WriteString(op)
	}
	return b.String()
}

type Instructions []*Instruction

func (ins Instructions) String() string {
	var b strings.Builder
	for _, i := range ins {
		b.WriteString(i.String())
	}
	return b.String()
}

func NewInstructionFromConfig(config *InstructionConfig) *Instruction {
	return NewRandomInstruction(config.OpCount)
}

func NewRandomInstruction(opCount int) *Instruction {
	self = &Instruction{Age: 0}
	for i := 0; i < opCount; {
		opset := PREFAB_OPSETS[rand.Intn(len(PREFAB_OPSETS))].ToOPs()
		self.Ops = append(self.Ops, opset)
		i += len(opset)
	}

	copier.Copy(self.InitialOpSet, Ops)
	return self
}

func (i *Instruction) Clone() *Instruction {
	clone := &Instruction{}
	copier.Copy(clone, i)
	return clone
}

func (i *Instruction) IncrementAge() {
	i.Age = i.Age + 1
}
