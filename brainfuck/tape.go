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

func (t *Tape) Reset() {
	t.InstructionPointer = 0
	t.WhileIndexStack = t.WhileIndexStack[:0]
}

func (t *Tape) Advance() bool {
	if t.InstructionPointer < len(t.Instructions)-1 {
		t.InstructionPointer = t.InstructionPointer + 1
		return true
	} else {
		return false
	}
}

func (t *Tape) GetCurrentInstruction() (bool, byte, error) {
	if t.InstructionPointer < 0 || t.InstructionPointer > len(t.Instructions)-1 {
		if t.InstructionPointer == len(t.Instructions)-1 {
			return false, NO_OP, nil
		}
		return false, NO_OP, fmt.Errorf("InstructionPointer [%d] out of bounds (Instruction length: [%d]", t.InstructionPointer, len(t.Instructions))
	}
	return true, t.Instructions[t.InstructionPointer], nil
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
				t.WhileIndexStack = append(t.WhileIndexStack, t.InstructionPointer)
			} else {
				if t.InstructionPointer == len(t.Instructions)-1 {
					return false, fmt.Errorf("OP_WHILE at tape index [%d] failed to advance to OP_WHILE_END instruction. Tape end reached.", t.InstructionPointer)
				}

				for i, op := range t.Instructions[t.InstructionPointer:len(t.Instructions)] {
					if op == OP_WHILE_END {
						// Move the instruction pointer to just before OP_WHILE_END
						t.InstructionPointer = t.InstructionPointer + i - 1
						return true, nil
					}
				}

				return false, fmt.Errorf("OP_WHILE at tape index [%d] failed to advance to OP_WHILE_END instruction. Tape end reached.", t.InstructionPointer)
			}
		} else {
			return false, fmt.Errorf("OP_WHILE at tape index [%d] failed to get current memory cell at index [%d] during OP_WHILE evaluation. %v", t.InstructionPointer, memory.MemoryPointer, err)
		}

	case OP_WHILE_END:
		if ok, val, err := memory.GetCurrentCell(); ok {
			if val != 0 {
				if len(t.WhileIndexStack) > 0 {
					while_start := t.WhileIndexStack[len(t.WhileIndexStack)-1]
					if while_start < 0 || while_start > len(t.Instructions)-1 {
						return false, fmt.Errorf("OP_WHILE_END at tape index [%d] failed to fallback. InstructionPointer [%d] from while stack is out of bounds (Instruction length: [%d]", t.InstructionPointer, while_start, len(t.Instructions))
					}
					t.WhileIndexStack = t.WhileIndexStack[:len(t.WhileIndexStack)-1]
					// Move the instruction pointer to just before OP_WHILE
					t.InstructionPointer = while_start - 1
					return true, nil
				} else {
					return false, fmt.Errorf("OP_WHILE_END at tape index [%d] failed to fallback. Failed to pop while stack.", t.InstructionPointer)
				}
			} else {
				// Ensure we pop the while stack when current memory cell is zero and we advance past
				if len(t.WhileIndexStack) > 0 {
					t.WhileIndexStack = t.WhileIndexStack[:len(t.WhileIndexStack)-1]
				}
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

	return true, nil
}
