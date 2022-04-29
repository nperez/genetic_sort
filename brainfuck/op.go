package brainfuck

import (
	"fmt"
	//"os"
)

// The OPs for Brainfuck. Also in here are OP sets for predefined functions. I
// didn't want to try to evolve (and identify) instructions from random
// permutations. So we have some starter functions for swapping items,
// searching for empty cells, assignment, etc.

// There two special instructions added: * ^
// Additionally, there is now a new bookmark register to store/recall a memory tape index. The ^ instruction will recall the stored memory tape index, store the current index, then jump the memory tape pointer to the recalled index. The * instruction will just store the current memory index to the register.

// The uninitialized bookmark is always zero

//  *
// [1][2][3][4][5][6][7][8][9][0]
// *[>]
// Bookmark the start. Search for a zero cell to the right

//                             *
// [1][2][3][4][5][6][7][8][9][0]
// ^[-^+^]
// Bookmark dance. Enter loop, loop eval: now one, dec 1, bookmark dance, inc 1, bookmark dance, loop eval: now zero, break loop

//     *
// [0][2][3][4][5][6][7][8][9][1]
// *>[-^+^]
// Bookmark. Move right one, enter loop, dec 1, jump back left, inc 1, loop, etc.

//     *
// [2][0][3][4][5][6][7][8][9][1]

// [-^+^] == very concise 0:N swap

type OP string
type OPS string

const (
	OP_POINTER_LEFT  = OP("<")
	OP_POINTER_RIGHT = OP(">")
	OP_INC           = OP("+")
	OP_DEC           = OP("-")
	OP_WHILE         = OP("[")
	OP_WHILE_END     = OP("]")
	OP_JUMP          = OP("^")
	OP_BOOKMARK      = OP("*")
	NO_OP            = OP("#")
)

const (
	SET_TO_ZERO        = OPS(`[-]`)
	FIND_ZERO_RIGHT    = OPS(`[>]`)
	FIND_ZERO_LEFT     = OPS(`[<]`)
	SWAP_RIGHT         = OPS(`*[>]^[-^+^]^[-^>+^]`)
	SWAP_LEFT          = OPS(`*[>]^[-^+^]^[-^<+^]`)
	MOVE_TO_ZERO_RIGHT = OPS(`*[>]^[-^+^]`)
	MOVE_TO_ZERO_LEFT  = OPS(`*[<]^[-^+^]`)
)

var OP_SET [9]OP = [...]OP{
	OP_POINTER_LEFT,
	OP_POINTER_RIGHT,
	OP_INC,
	OP_DEC,
	OP_WHILE,
	OP_WHILE_END,
	OP_JUMP,
	OP_BOOKMARK,
	NO_OP,
}

var PREFAB_OPSETS [7]OPS = [...]OPS{
	SET_TO_ZERO,
	FIND_ZERO_RIGHT,
	FIND_ZERO_LEFT,
	SWAP_RIGHT,
	SWAP_LEFT,
	MOVE_TO_ZERO_RIGHT,
	MOVE_TO_ZERO_LEFT,
}

func (o OPS) ToOPs() []OP {
	ops := []OP{}
	for _, r := range o {
		ops = append(ops, OP(r))
	}
	return ops
}

func (o OP) Execute(tape *Tape, memory *Memory) (bool, error) {
	switch o {
	case OP_INC:
		if ok, err := memory.Increment(); !ok {
			return false, fmt.Errorf("OP_INC at tape index [%d] failed to increment memory cell index [%d]. %v", tape.InstructionPointer, memory.MemoryPointer, err)
		}
	case OP_DEC:
		if ok, err := memory.Decrement(); !ok {
			return false, fmt.Errorf("OP_DEC at tape index [%d] failed to decrement memory cell index [%d]. %v", tape.InstructionPointer, memory.MemoryPointer, err)
		}
	case OP_POINTER_LEFT:
		if ok, err := memory.MovePointerLeft(); !ok {
			return false, fmt.Errorf("OP_POINTER_LEFT at tape index [%d] failed to move memory pointer left. %v", tape.InstructionPointer, err)
		}
	case OP_POINTER_RIGHT:
		if ok, err := memory.MovePointerRight(); !ok {
			return false, fmt.Errorf("OP_POINTER_RIGHT at tape index [%d] failed to move memory pointer right. %v", tape.InstructionPointer, err)
		}
	case OP_WHILE:
		if ok, val, err := memory.GetCurrentCell(); ok {
			if val != 0 {
				if ok, err := tape.PushWhile(); !ok {
					return false, fmt.Errorf("OP_WHILE at tape index [%d] failed to push while. %v", tape.InstructionPointer, err)
				}
			} else {
				if ok, err := tape.AdvanceToWhileEnd(); !ok {
					return false, fmt.Errorf("OP_WHILE at tape index [%d] failed to advance to matching OP_WHILE_END. %v", tape.InstructionPointer, err)
				}
			}
		} else {
			return false, fmt.Errorf("OP_WHILE at tape index [%d] failed to get current memory cell at index [%d] during OP_WHILE evaluation. %v", memory.MemoryPointer, tape.InstructionPointer, err)
		}

	case OP_WHILE_END:
		if ok, val, err := memory.GetCurrentCell(); ok {
			if val != 0 {
				if ok, err := tape.FallbackToWhileStart(); !ok {
					return false, fmt.Errorf("OP_WHILE_END at tape index [%d] failed to fallback. %v", tape.InstructionPointer, err)
				}
				// Don't advance since we just moved the tape pointer directly to OP_WHILE
				return true, nil
			}
			// Value is zero, we're escaping the loop, pop the while stack
			if ok, err := tape.PopWhile(); !ok {
				return false, fmt.Errorf("OP_WHILE_END at tape index [%d] failed to escape scope. %v", tape.InstructionPointer, err)
			}
		} else {
			return false, fmt.Errorf("OP_WHILE at tape index [%d] failed to get current memory cell at index [%d] during OP_WHILE evaluation. %v", memory.MemoryPointer, tape.InstructionPointer, err)
		}
	case OP_JUMP:
		if ok, err := memory.BookmarkJump(); !ok {
			return false, fmt.Errorf("OP_JUMP at tape index [%d] failed to jump. %v", tape.InstructionPointer, err)
		}
	case OP_BOOKMARK:
		if ok, err := memory.StoreBookmark(); !ok {
			return false, fmt.Errorf("OP_BOOKMARK at tape index [%d] failed to store. %v", tape.InstructionPointer, err)
		}
	case NO_OP:
		// fmt.Fprintf(os.Stderr, "MACHINE STATE:\nMEMORY DUMP: %v\nMEMORY POINTER: %v\nINSTRUCTION DUMP: %v\nINSTRUCTION POINTER: %v\nWHILE STACK: %v\n", memory.Cells, memory.MemoryPointer, tape.Instructions, tape.InstructionPointer, tape.WhileIndexStack)
	default:
		panic(fmt.Sprintf("Unknown OP [%s] encountered!", o))
	}

	if !tape.Advance() {
		return false, nil
	}

	return true, nil
}
