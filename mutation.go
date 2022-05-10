package genetic_sort

import (
	"fmt"
	"math/rand"
	bf "nickandperla.net/brainfuck"
	str "strings"
)

type META_OP byte

const (
	PUSH_OP META_OP = iota
	POP_OP
	SHIFT_OP
	UNSHIFT_OP
	INSERT_OP
	DELETE_OP
	SWAP_OP
	REPLACE_OP
	META_NO_OP
)

var META_OP_SET = []META_OP{
	PUSH_OP,
	POP_OP,
	SHIFT_OP,
	UNSHIFT_OP,
	INSERT_OP,
	DELETE_OP,
	SWAP_OP,
	REPLACE_OP,
	META_NO_OP,
}

type Mutation struct {
	ID            uint
	InstructionID uint
	Position1     *uint
	Position2     *uint
	MetaOP        META_OP
	Op            byte
	Chance        float32
}

func (m *Mutation) String() string {
	return fmt.Sprintf("{Position1: %v, Position2: %v, MetaOP: %v, Op: %v, Chance: %v}", m.Position1, m.Position2, m.MetaOP, m.Op, m.Chance)
}

func NewMutation(chance float32) *Mutation {
	m := &Mutation{
		MetaOP: META_OP_SET[rand.Intn(len(META_OP_SET))],
		Op:     bf.OP_SET[rand.Intn(len(bf.OP_SET))],
		Chance: chance,
	}
	return m
}

func (m *Mutation) Apply(i *Instruction) {
	length := len(i.Ops)
	pos1, pos2 := uint(rand.Intn(length)), uint(rand.Intn(length))

	switch m.MetaOP {
	case PUSH_OP:
		i.Ops = string(append([]byte(i.Ops), m.Op))
	case POP_OP:
		i.Ops = i.Ops[:len(i.Ops)-1]
	case SHIFT_OP:
		i.Ops = i.Ops[1:]
	case UNSHIFT_OP:
		i.Ops = string(append([]byte{m.Op}, []byte(i.Ops)...))
	case INSERT_OP:
		index := pos1
		first := i.Ops[:index]
		second := i.Ops[index:]
		i.Ops = str.Join([]string{first, string(m.Op), second}, "")
		m.Position1 = &pos1
	case DELETE_OP:
		index := pos1
		first := i.Ops[:index]
		second := i.Ops[index+1:]
		i.Ops = str.Join([]string{first, second}, "")
		m.Position1 = &pos1
	case SWAP_OP:
		i1 := pos1
		i2 := pos2
		bslice := []byte(i.Ops)
		first := i.Ops[i1]
		second := i.Ops[i2]
		bslice[i1] = second
		bslice[i2] = first
		i.Ops = string(bslice)
		m.Position1 = &pos1
		m.Position2 = &pos2
	case REPLACE_OP:
		bslice := []byte(i.Ops)
		bslice[pos1] = m.Op
		i.Ops = string(bslice)
		m.Position1 = &pos1
	}

	i.Mutations = append(i.Mutations, m)
}
