package brainfuck

import (
	"fmt"
	"os"
)

type Tape struct {
	Instructions       string
	InstructionPointer int
	WhileIndexStack    []int
}

const WHILE_STACK_CAP = 10

func NewTape(instructions string) *Tape {
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

func (t *Tape) GetCurrentInstruction() (bool, byte, error) {
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

func (t *Tape) PopWhile() (bool, error) {
	var while_start int
	while_start, t.WhileIndexStack = t.WhileIndexStack[len(t.WhileIndexStack)-1], t.WhileIndexStack[:len(t.WhileIndexStack)-1]

	if !t.InBounds(while_start) {
		return false, fmt.Errorf("InstructionPointer [%d] from while stack is out of bounds (Instruction length: [%d]", while_start, len(t.Instructions))
	}

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

var DEBUG bool = false

func (t *Tape) Execute(memory *Memory) (bool, error) {
	ok, o, err := t.GetCurrentInstruction()

	if !ok {
		return false, err
	}

	switch o {
	case OP_INC:
		if ok, err := memory.Increment(); !ok {
			return false, fmt.Errorf("OP_INC at tape index [%d] failed to increment memory cell index [%d]. %v", t.InstructionPointer, memory.MemoryPointer, err)
		}
	case OP_DEC:
		if ok, err := memory.Decrement(); !ok {
			return false, fmt.Errorf("OP_DEC at tape index [%d] failed to decrement memory cell index [%d]. %v", t.InstructionPointer, memory.MemoryPointer, err)
		}
	case OP_POINTER_LEFT:
		if ok, err := memory.MovePointerLeft(); !ok {
			return false, fmt.Errorf("OP_POINTER_LEFT at tape index [%d] failed to move memory pointer left. %v", t.InstructionPointer, err)
		}
	case OP_POINTER_RIGHT:
		if ok, err := memory.MovePointerRight(); !ok {
			return false, fmt.Errorf("OP_POINTER_RIGHT at tape index [%d] failed to move memory pointer right. %v", t.InstructionPointer, err)
		}
	case OP_WHILE:
		if ok, val, err := memory.GetCurrentCell(); ok {
			if val != 0 {
				if ok, err := t.PushWhile(); !ok {
					return false, fmt.Errorf("OP_WHILE at tape index [%d] failed to push while. %v", t.InstructionPointer, err)
				}
			} else {
				if ok, err := t.AdvanceToWhileEnd(); !ok {
					return false, fmt.Errorf("OP_WHILE at tape index [%d] failed to advance to matching OP_WHILE_END. %v", t.InstructionPointer, err)
				}
			}
		} else {
			return false, fmt.Errorf("OP_WHILE at tape index [%d] failed to get current memory cell at index [%d] during OP_WHILE evaluation. %v", memory.MemoryPointer, t.InstructionPointer, err)
		}

	case OP_WHILE_END:
		if ok, val, err := memory.GetCurrentCell(); ok {
			if val != 0 {
				if ok, err := t.FallbackToWhileStart(); !ok {
					return false, fmt.Errorf("OP_WHILE_END at tape index [%d] failed to fallback. %v", t.InstructionPointer, err)
				}
				// Don't advance since we just moved the tape pointer directly to OP_WHILE
				return true, nil
			}
			// Value is zero, we're escaping the loop, pop the while stack
			if ok, err := t.PopWhile(); !ok {
				return false, fmt.Errorf("OP_WHILE_END at tape index [%d] failed to escape scope. %v", t.InstructionPointer, err)
			}
		} else {
			return false, fmt.Errorf("OP_WHILE at tape index [%d] failed to get current memory cell at index [%d] during OP_WHILE evaluation. %v", memory.MemoryPointer, t.InstructionPointer, err)
		}
	case OP_JUMP:
		if ok, err := memory.BookmarkJump(); !ok {
			return false, fmt.Errorf("OP_JUMP at tape index [%d] failed to jump. %v", t.InstructionPointer, err)
		}
	case OP_BOOKMARK:
		if ok, err := memory.StoreBookmark(); !ok {
			return false, fmt.Errorf("OP_BOOKMARK at tape index [%d] failed to store. %v", t.InstructionPointer, err)
		}
	case NO_OP:
		if DEBUG {
			fmt.Fprintf(os.Stderr, "\n---\nMACHINE STATE:\nMEMORY DUMP: %v\nMEMORY POINTER: %v\nINSTRUCTION DUMP: %v\nINSTRUCTION POINTER: %v\nWHILE STACK: %v\nBOOKMARK: %v\n", memory.Cells, memory.MemoryPointer, t.Instructions, t.InstructionPointer, t.WhileIndexStack, memory.BookmarkRegister)
		}
	default:
		panic(fmt.Sprintf("Unknown OP [%v] encountered!", o))
	}

	if !t.Advance() {
		return false, nil
	}

	return true, nil
}
