package genetic_sort

import (
    "math/rand"
)

type META_OP int

const (
    PUSH_OP = META_OP(0)
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

type Mutation interface {
    Apply(interface{})
}

type InstructionMutation struct {
    Positions []int
    MetaOP META_OP
    Op OP
}

func NewInstructionMutation() Mutation {
    m := &Mutation{
        MetaOP: META_OP_SET[rand.Intn(len(META_OP_SET))],
        Op: OP_SET[rand.Intn(len(OP_SET))],
    }
    return m
}

func (m *InstructionMutation) Apply (i *Instruction) {
    length := len(i.Ops)
    m.Positions = []int{rand.Intn(length), rand.Intn(length)}

    switch m.MetaOP {
        case PUSH_OP:
            i.Ops = append(i.Ops, m.Op)
        case POP_OP:
            i.Ops = i.Ops[:len(i.Ops)-1]
        case SHIFT_OP:
            i.Ops = i.Ops[1:]
        case UNSHIFT_OP:
            i.Ops = append([]OP{m.Op}, i.Ops)
        case INSERT_OP:
            index := m.Positions[0]
            i.Ops = append(i.Ops, "")
            copy(i.Ops[index+1:], i.Ops[index:])
            i.Ops[index] = m.Op
        case DELETE_OP:
            index := m.Positions[0]
            copy(i.Ops[index:], i.Ops[index+1:])
            i.Ops[len(i.Ops)-1] = ""
            i.Ops = i.Ops[:len(i.Ops)-1]
        case SWAP_OP:
            i1 = m.Positions[0]
            i2 = m.Positions[1]
            first := i.Ops[i1]
            second := i.Ops[i2]
            i.Ops[i1] = second
            i.Ops[i2] = first
        case REPLACE_OP:
            i.Ops[m.Position[0]] = m.Op
    }

    i.Mutations = append(i.Mutations, m)
}
