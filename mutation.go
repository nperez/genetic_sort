package genetic_sort

import (
	"fmt"

	bf "nickandperla.net/brainfuck"
)

var META_OP_SET = []byte{
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
	MetaOP        byte
	Op            byte
	Chance        float32
}

func (m *Mutation) String() string {
	return fmt.Sprintf("{Position1: %v, Position2: %v, MetaOP: %v, Op: %v, Chance: %v}", m.Position1, m.Position2, m.MetaOP, m.Op, m.Chance)
}

func NewMutation(chance float32) *Mutation {
	m := &Mutation{
		MetaOP: META_OP_SET[rng.Intn(len(META_OP_SET))],
		Op:     bf.OP_SET[rng.Intn(len(bf.OP_SET))],
		Chance: chance,
	}
	return m
}

// Apply mutates the instruction's ops in decompressed form.
// Does NOT recompress — sets Ops=nil so EnsureCompressed will repack on demand
// (only needed for DB persistence). This avoids makeOpsSmall in the hot path.
func (m *Mutation) Apply(i *Instruction) {
	i.EnsureDecompressed()
	raw := i.cachedOps
	length := len(raw)
	if length == 0 {
		i.Mutations = append(i.Mutations, m)
		return
	}
	pos1, pos2 := uint(rng.Intn(length)), uint(rng.Intn(length))

	switch m.MetaOP {
	case PUSH_OP:
		raw = append(raw, m.Op)
	case POP_OP:
		raw = raw[:len(raw)-1]
	case SHIFT_OP:
		raw = raw[1:]
	case UNSHIFT_OP:
		raw = append([]byte{m.Op}, raw...)
	case INSERT_OP:
		index := pos1
		first := make([]byte, index)
		copy(first, raw[:index])
		second := raw[index:]
		temp := append(first, m.Op)
		raw = append(temp, second...)
		m.Position1 = &pos1
	case DELETE_OP:
		index := pos1
		first := make([]byte, index)
		copy(first, raw[:index])
		second := raw[index+1:]
		raw = append(first, second...)
		m.Position1 = &pos1
	case SWAP_OP:
		raw[pos1], raw[pos2] = raw[pos2], raw[pos1]
		m.Position1 = &pos1
		m.Position2 = &pos2
	case REPLACE_OP:
		raw[pos1] = m.Op
		m.Position1 = &pos1
	}

	i.cachedOps = raw
	i.Ops = nil // mark compressed form as stale
	i.Mutations = append(i.Mutations, m)
}
