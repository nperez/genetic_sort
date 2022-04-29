package brainfuck

import (
	"fmt"
)

type Tape struct {
	Instructions       []OP
	InstructionPointer int
	WhileIndexStack    []int
}

const WHILE_STACK_CAP = 10

func NewTape(instructions []OP) *Tape {
	return &Tape{
		Instructions:       instructions,
		InstructionPointer: 0,
		WhileIndexStack:    make([]int, 0, WHILE_STACK_CAP),
	}
}

func (t *Tape) Advance() bool {
	if t.InBounds(t.InstructionPointer + 1) {
		t.InstructionPointer = t.InstructionPointer + 1
		return true
	} else {
		return false
	}
}

func (t *Tape) GetCurrentInstruction() (bool, OP, error) {
	if !t.InBounds(t.InstructionPointer) {
		if t.InstructionPointer == len(t.Instructions)-1 {
			return false, NO_OP, nil
		}
		return false, NO_OP, fmt.Errorf("InstructionPointer [%d] out of bounds (Instruction length: [%d]", t.InstructionPointer, len(t.Instructions))
	}
	return true, t.Instructions[t.InstructionPointer], nil
}

func (t *Tape) InBounds(new_val int) bool {
	return new_val >= 0 && new_val <= len(t.Instructions)-1
}

func (t *Tape) PushWhile() (bool, error) {
	if !t.InBounds(t.InstructionPointer) {
		return false, fmt.Errorf("Failed to store current InstructionPointer [%d] on while stack. Out of bounds (Instruction length: [%d]", t.InstructionPointer, len(t.Instructions))
	}
	t.WhileIndexStack = append(t.WhileIndexStack, t.InstructionPointer)
	return true, nil
}

func (t *Tape) AdvanceToWhileEnd() (bool, error) {
	if !t.InBounds(t.InstructionPointer + 1) {
		return false, fmt.Errorf("Failed to advance to OP_WHILE_END instruction. Tape end reached.")
	}

	for i, op := range t.Instructions[t.InstructionPointer:len(t.Instructions)] {
		if op == OP_WHILE_END {
			t.InstructionPointer = t.InstructionPointer + i
			return true, nil
		}
	}

	return false, fmt.Errorf("Failed to advance to OP_WHILE_END instruction. Tape end reached.")
}

func (t *Tape) FallbackToWhileStart() (bool, error) {
	if len(t.WhileIndexStack) > 0 {
		while_start := t.WhileIndexStack[len(t.WhileIndexStack)-1]
		if !t.InBounds(while_start) {
			return false, fmt.Errorf("InstructionPointer [%d] from while stack is out of bounds (Instruction length: [%d]", while_start, len(t.Instructions))
		}
		t.WhileIndexStack = t.WhileIndexStack[:len(t.WhileIndexStack)-1]
		t.InstructionPointer = while_start
		return true, nil
	}

	return false, fmt.Errorf("Failed to pop while stack.")
}
