package genetic_sort

import (
	"fmt"
	"log"
	str "strings"

	bf "nickandperla.net/brainfuck"
)

// Lookup tables for 4-bit nibble <-> BF op conversion.
// Avoids encoding/binary + strings.Builder overhead.
var nibbleToOp [16]byte
var opToNibble [256]byte

func init() {
	nibbleToOp[1] = bf.OP_POINTER_LEFT
	nibbleToOp[2] = bf.OP_POINTER_RIGHT
	nibbleToOp[3] = bf.OP_INC
	nibbleToOp[4] = bf.OP_DEC
	nibbleToOp[5] = bf.OP_WHILE
	nibbleToOp[6] = bf.OP_WHILE_END
	nibbleToOp[7] = bf.OP_JUMP
	nibbleToOp[8] = bf.OP_BOOKMARK
	nibbleToOp[9] = bf.NO_OP

	opToNibble[bf.OP_POINTER_LEFT] = 1
	opToNibble[bf.OP_POINTER_RIGHT] = 2
	opToNibble[bf.OP_INC] = 3
	opToNibble[bf.OP_DEC] = 4
	opToNibble[bf.OP_WHILE] = 5
	opToNibble[bf.OP_WHILE_END] = 6
	opToNibble[bf.OP_JUMP] = 7
	opToNibble[bf.OP_BOOKMARK] = 8
	opToNibble[bf.NO_OP] = 9
}

type InstructionConfig struct {
	OpSetCount int `toml:"op_set_count"`
}

type Instruction struct {
	ID           uint
	UnitID       uint
	Mutations    []*Mutation
	Age          uint
	InitialOpSet []byte
	Ops          []byte
	cachedOps    []byte // decompressed ops; primary form during in-memory work
}

type Instructions []*Instruction

func NewInstructionFromConfig(config *InstructionConfig) *Instruction {
	return NewRandomInstruction(config.OpSetCount)
}

func NewRandomInstruction(opSetCount int) *Instruction {
	instruction := &Instruction{Age: 0}

	var sb str.Builder

	for i := 0; i < opSetCount; i++ {
		sb.WriteString(bf.PREFAB_OPSETS[rng.Intn(len(bf.PREFAB_OPSETS))])
	}

	raw := []byte(sb.String())
	instruction.Ops = makeOpsSmallBytes(raw)
	instruction.InitialOpSet = makeOpsSmallBytes(raw)
	instruction.cachedOps = raw
	return instruction
}

func NewInstruction(ops string) *Instruction {
	instruction := &Instruction{Age: 0}
	raw := []byte(ops)
	instruction.Ops = makeOpsSmallBytes(raw)
	instruction.InitialOpSet = makeOpsSmallBytes(raw)
	instruction.cachedOps = raw
	return instruction
}

func (ins Instructions) OpsCount() uint {
	var count uint
	for _, i := range ins {
		if i.cachedOps != nil {
			count += uint(len(i.cachedOps))
		} else {
			count += uint(len(i.Ops) * 2)
		}
	}
	return count
}

func (ins Instructions) ToProgram() string {
	program := make([]byte, ins.OpsCount())
	var offset uint
	for _, i := range ins {
		sub := i.ToProgram()
		copy(program[offset:], sub)
		offset += uint(len(sub))
	}
	return string(program[:offset])
}

// ToProgram returns the decompressed BF ops for this instruction.
// Lazily decompresses from packed form and caches the result.
func (i *Instruction) ToProgram() []byte {
	if i.cachedOps != nil {
		return i.cachedOps
	}
	i.cachedOps = makeOpsBig(i.Ops)
	return i.cachedOps
}

// EnsureDecompressed populates cachedOps from the packed Ops if not already cached.
func (i *Instruction) EnsureDecompressed() {
	if i.cachedOps == nil {
		i.cachedOps = makeOpsBig(i.Ops)
	}
}

// EnsureCompressed repacks cachedOps into the Ops field for DB persistence.
// Only does work if the compressed form is stale (Ops == nil after mutation).
func (i *Instruction) EnsureCompressed() {
	if i.Ops == nil && i.cachedOps != nil {
		i.Ops = makeOpsSmallBytes(i.cachedOps)
	}
}

// makeOpsBig decompresses 4-bit packed ops into raw BF op bytes.
// Uses lookup table instead of encoding/binary + strings.Builder.
func makeOpsBig(stuff []byte) []byte {
	result := make([]byte, 0, len(stuff)*2)

	if DEBUG {
		log.Printf("Making things big. Count: %v, Original: %v", len(stuff), stuff)
	}

	for i := 0; i+4 <= len(stuff); i += 4 {
		packed := uint32(stuff[i])<<24 | uint32(stuff[i+1])<<16 | uint32(stuff[i+2])<<8 | uint32(stuff[i+3])
		for j := uint(0); j < 8; j++ {
			symbol := (packed >> (28 - 4*j)) & 0xF
			if symbol == 0 {
				continue
			}
			op := nibbleToOp[symbol]
			if op == 0 {
				panic(fmt.Sprintf("Unknown symbol [%v] encountered!", symbol))
			}
			result = append(result, op)
		}
	}

	if DEBUG {
		log.Printf("Making things big. Count: %v, Unpacked: %v", len(result), result)
	}
	return result
}

// makeOpsSmallBytes compresses raw BF op bytes into 4-bit packed format.
// Uses lookup table instead of encoding/binary + switch statements.
func makeOpsSmallBytes(raw []byte) []byte {
	if len(raw) == 0 {
		return nil
	}
	outLen := ((len(raw) + 7) / 8) * 4
	result := make([]byte, outLen)

	if DEBUG {
		log.Printf("Making things small. Count: %v, Original: %v", len(raw), raw)
	}

	for i := 0; i < len(raw); i += 8 {
		var packed uint32
		end := i + 8
		if end > len(raw) {
			end = len(raw)
		}
		for j := 0; j < end-i; j++ {
			symbol := opToNibble[raw[i+j]]
			if symbol == 0 && raw[i+j] != 0 {
				panic(fmt.Sprintf("Unknown OP [%v] encountered!", raw[i+j]))
			}
			packed |= uint32(symbol) << (28 - uint(4*j))
		}
		outIdx := (i / 8) * 4
		result[outIdx] = byte(packed >> 24)
		result[outIdx+1] = byte(packed >> 16)
		result[outIdx+2] = byte(packed >> 8)
		result[outIdx+3] = byte(packed)
	}

	if DEBUG {
		log.Printf("Making things small. Count: %v, Packed: %v", len(result), result)
	}
	return result
}

// makeOpsSmall is a wrapper for backward compatibility with string input.
func makeOpsSmall(stuff string) []byte {
	return makeOpsSmallBytes([]byte(stuff))
}

// Clone creates a deep copy of this Instruction without reflection.
func (i *Instruction) Clone() *Instruction {
	clone := &Instruction{
		ID:     i.ID,
		UnitID: i.UnitID,
		Age:    i.Age,
	}
	if i.Ops != nil {
		clone.Ops = make([]byte, len(i.Ops))
		copy(clone.Ops, i.Ops)
	}
	if i.InitialOpSet != nil {
		clone.InitialOpSet = make([]byte, len(i.InitialOpSet))
		copy(clone.InitialOpSet, i.InitialOpSet)
	}
	if i.cachedOps != nil {
		clone.cachedOps = make([]byte, len(i.cachedOps))
		copy(clone.cachedOps, i.cachedOps)
	}
	return clone
}

func (i *Instruction) IncrementAge() {
	i.Age = i.Age + 1
}
