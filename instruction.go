package genetic_sort

import (
	"bytes"
	bin "encoding/binary"
	"fmt"
	"io"
	"math/rand"
	str "strings"

	cp "github.com/jinzhu/copier"
	gorm "gorm.io/gorm"
	"log"
	bf "nickandperla.net/brainfuck"
)

type InstructionConfig struct {
	OpSetCount int
}

type Instruction struct {
	ID           uint
	UnitID       uint
	Mutations    []*Mutation
	Age          int
	InitialOpSet string `gorm:"type:blob"`
	Ops          string `gorm:"type:blob"`
}

func NewInstructionFromConfig(config *InstructionConfig) *Instruction {
	return NewRandomInstruction(config.OpSetCount)
}

func NewRandomInstruction(opSetCount int) *Instruction {
	instruction := &Instruction{Age: 0}

	var sb str.Builder

	for i := 0; i < opSetCount; i++ {
		sb.WriteString(bf.PREFAB_OPSETS[rand.Intn(len(bf.PREFAB_OPSETS))])
	}

	instruction.Ops = sb.String()
	instruction.InitialOpSet = sb.String()
	return instruction
}

func opsToSymbols(ops string) []byte {
	ret := make([]byte, len(ops))
	for i, op := range ops {
		switch byte(op) {
		case bf.OP_POINTER_LEFT:
			ret[i] = 1
		case bf.OP_POINTER_RIGHT:
			ret[i] = 2
		case bf.OP_INC:
			ret[i] = 3
		case bf.OP_DEC:
			ret[i] = 4
		case bf.OP_WHILE:
			ret[i] = 5
		case bf.OP_WHILE_END:
			ret[i] = 6
		case bf.OP_JUMP:
			ret[i] = 7
		case bf.OP_BOOKMARK:
			ret[i] = 8
		case bf.NO_OP:
			ret[i] = 9
		default:
			panic(fmt.Sprintf("Unknown OP [%v] encountered!", op))
		}
	}

	return ret
}

const (
	DEBUG = false
)

func makeOpsSmall(stuff []byte) []byte {
	buffer := bytes.NewBuffer(stuff)
	window := make([]byte, 8)
	compressBuffer := bytes.NewBuffer([]byte{})

	if DEBUG {
		log.Printf("Making things small. Count: %v, Original: %v", len(stuff), stuff)
	}
	for {
		var packed uint32
		_, err := buffer.Read(window)
		if err == io.EOF {
			break
		}
		for i, bits := range window {
			switch i % 8 {
			case 0:
				packed += 0b11110000000000000000000000000000 & (uint32(bits) << 28)
			case 1:
				packed += 0b00001111000000000000000000000000 & (uint32(bits) << 24)
			case 2:
				packed += 0b00000000111100000000000000000000 & (uint32(bits) << 20)
			case 3:
				packed += 0b00000000000011110000000000000000 & (uint32(bits) << 16)
			case 4:
				packed += 0b00000000000000001111000000000000 & (uint32(bits) << 12)
			case 5:
				packed += 0b00000000000000000000111100000000 & (uint32(bits) << 8)
			case 6:
				packed += 0b00000000000000000000000011110000 & (uint32(bits) << 4)
			case 7:
				packed += 0b00000000000000000000000000001111 & (uint32(bits))
			}
		}
		bin.Write(compressBuffer, bin.BigEndian, packed)
	}

	compressed := compressBuffer.Bytes()
	if DEBUG {
		log.Printf("Making things small. Count: %v, Packed: %v", len(compressed), compressed)
	}
	return compressed
}

func (i *Instruction) BeforeSave(tx *gorm.DB) error {
	i.Ops = string(makeOpsSmall(opsToSymbols(i.Ops)))
	i.InitialOpSet = string(makeOpsSmall(opsToSymbols(i.InitialOpSet)))
	return nil
}

func (i *Instruction) Clone() *Instruction {
	clone := &Instruction{}
	cp.Copy(clone, i)
	return clone
}

func (i *Instruction) IncrementAge() {
	i.Age = i.Age + 1
}
