package genetic_sort

import (
	"bytes"
	bin "encoding/binary"
	"fmt"
	"io"
	"math/rand"
	str "strings"

	cp "github.com/jinzhu/copier"
	//gorm "gorm.io/gorm"
	"log"

	bf "nickandperla.net/brainfuck"
)

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
}

type Instructions []*Instruction

func NewInstructionFromConfig(config *InstructionConfig) *Instruction {
	return NewRandomInstruction(config.OpSetCount)
}

func NewRandomInstruction(opSetCount int) *Instruction {
	instruction := &Instruction{Age: 0}

	var sb str.Builder

	for i := 0; i < opSetCount; i++ {
		sb.WriteString(bf.PREFAB_OPSETS[rand.Intn(len(bf.PREFAB_OPSETS))])
	}

	instruction.Ops = makeOpsSmall(sb.String())
	instruction.InitialOpSet = makeOpsSmall(sb.String())
	return instruction
}

func NewInstruction(ops string) *Instruction {
	instruction := &Instruction{Age: 0}
	instruction.Ops = makeOpsSmall(ops)
	instruction.InitialOpSet = makeOpsSmall(ops)
	return instruction
}

func (ins Instructions) OpsCount() uint {
	var count uint
	for _, i := range ins {
		count += uint(len(i.Ops) * 2)
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

func (i *Instruction) ToProgram() []byte {
	return makeOpsBig(i.Ops)
}

func makeOpsBig(stuff []byte) []byte {
	buffer := bytes.NewBuffer(stuff)

	var sb str.Builder

	if DEBUG {
		log.Printf("Making things big. Count: %v, Original: %v", len(stuff), stuff)
	}
	for {
		var packed uint32 = 0
		err := bin.Read(buffer, bin.BigEndian, &packed)
		if err == io.EOF {
			break
		}
		for i := 0; i < 8; i++ {
			symbol := ((15 << (28 - (4 * i))) & packed) >> (28 - (4 * i))
			switch symbol {
			case 0:
				break
			case 1:
				sb.WriteRune(bf.OP_POINTER_LEFT)
			case 2:
				sb.WriteRune(bf.OP_POINTER_RIGHT)
			case 3:
				sb.WriteRune(bf.OP_INC)
			case 4:
				sb.WriteRune(bf.OP_DEC)
			case 5:
				sb.WriteRune(bf.OP_WHILE)
			case 6:
				sb.WriteRune(bf.OP_WHILE_END)
			case 7:
				sb.WriteRune(bf.OP_JUMP)
			case 8:
				sb.WriteRune(bf.OP_BOOKMARK)
			case 9:
				sb.WriteRune(bf.NO_OP)
			default:
				panic(fmt.Sprintf("Unknown symbol [%v] encountered!", symbol))
			}
		}
		if err == io.ErrUnexpectedEOF {
			break
		}
	}

	uncompressed := []byte(sb.String())

	if DEBUG {
		log.Printf("Making things big. Count: %v, Unpacked: %v", len(uncompressed), uncompressed)
	}
	return uncompressed
}

func makeOpsSmall(stuff string) []byte {
	buffer := bytes.NewBuffer([]byte(stuff))
	window := make([]byte, 8)
	compressBuffer := bytes.NewBuffer(make([]byte, 0, len(stuff)/2+1))

	if DEBUG {
		log.Printf("Making things small. Count: %v, Original: %v", len(stuff), []byte(stuff))
	}
	for {
		var packed uint32
		_, err := buffer.Read(window)
		if err == io.EOF {
			break
		}
		for i, bits := range window {
			var symbol byte
			switch byte(bits) {
			case 0:
				break
			case bf.OP_POINTER_LEFT:
				symbol = 1
			case bf.OP_POINTER_RIGHT:
				symbol = 2
			case bf.OP_INC:
				symbol = 3
			case bf.OP_DEC:
				symbol = 4
			case bf.OP_WHILE:
				symbol = 5
			case bf.OP_WHILE_END:
				symbol = 6
			case bf.OP_JUMP:
				symbol = 7
			case bf.OP_BOOKMARK:
				symbol = 8
			case bf.NO_OP:
				symbol = 9
			default:
				panic(fmt.Sprintf("Unknown OP [%v] encountered!", bits))
			}

			packed += (15 << (28 - (4 * (i % 8)))) & (uint32(symbol) << (28 - (4 * (i % 8))))
			window[i] = 0
		}
		bin.Write(compressBuffer, bin.BigEndian, packed)
	}

	compressed := compressBuffer.Bytes()
	if DEBUG {
		log.Printf("Making things small. Count: %v, Packed: %v", len(compressed), compressed)
	}
	return compressed
}

func (i *Instruction) Clone() *Instruction {
	clone := &Instruction{}
	cp.Copy(clone, i)
	return clone
}

func (i *Instruction) IncrementAge() {
	i.Age = i.Age + 1
}
