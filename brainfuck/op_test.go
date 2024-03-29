package brainfuck

import (
	"testing"
)

func MakeTapeAndMemory(op ...rune) (*Tape, *Memory) {
	return NewTape(string(op)), NewMemory(10)
}

func Test_OP_INC(t *testing.T) {

	tape, mem := MakeTapeAndMemory(OP_INC)

	if ok, err := tape.Execute(mem); !ok {
		t.Errorf("Unexpected failure when calling OP_INC.Execute() %v.", err)
	}

	if mem.Cells[0] != 1 {
		t.Errorf("Memory cell at index [0] (%d) wasn't incremented.", mem.Cells[0])
	}

	if tape.InstructionPointer != 0 {
		t.Errorf("Instruction pointer [%d] is not at expected value [0]", tape.InstructionPointer)
	}

	mem.Cells[0] = 255

	if ok, err := tape.Execute(mem); ok {
		t.Errorf("Unexpected success when calling OP_INC.Execute().")
	} else {
		if err.Error() != "OP_INC at tape index [0] failed to increment memory cell index [0]. Increment failed. Cell value [255] at UpperBound [255]" {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}
}

func Test_OP_DEC(t *testing.T) {

	tape, mem := MakeTapeAndMemory(OP_DEC)

	mem.Cells[0] = 2

	if ok, err := tape.Execute(mem); !ok {
		t.Errorf("Unexpected failure when calling OP_DEC.Execute(). %v", err)
	}

	if mem.Cells[0] != 1 {
		t.Errorf("Memory cell at index [0] (%d) wasn't incremented.", mem.Cells[0])
	}

	if tape.InstructionPointer != 0 {
		t.Errorf("Instruction pointer [%d] is not at expected value [0]", tape.InstructionPointer)
	}

	mem.Cells[0] = 0

	if ok, err := tape.Execute(mem); ok {
		t.Errorf("Unexpected success when calling OP_DEC.Execute().")
	} else {
		if err.Error() != "OP_DEC at tape index [0] failed to decrement memory cell index [0]. Decrement failed. Cell value [0] at LowerBound [0]" {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}
}

func Test_OP_POINTER_LEFT(t *testing.T) {

	tape, mem := MakeTapeAndMemory(OP_POINTER_LEFT)

	mem.MemoryPointer = 2

	if ok, err := tape.Execute(mem); !ok {
		t.Errorf("Unexpected failure when calling OP_POINTER_LEFT.Execute(). %v", err)
	}

	if mem.MemoryPointer != 1 {
		t.Errorf("Memory pointer [%d] is not at expected value [1].", mem.MemoryPointer)
	}

	if tape.InstructionPointer != 0 {
		t.Errorf("Instruction pointer [%d] is not at expected value [0]", tape.InstructionPointer)
	}

	mem.MemoryPointer = 0
	tape.InstructionPointer = 0

	if ok, err := tape.Execute(mem); ok {
		t.Errorf("Unexpected success when calling OP_POINTER_LEFT.Execute().")
	} else {
		if err.Error() != "OP_POINTER_LEFT at tape index [0] failed to move memory pointer left. Failed to move memory pointer [0] left. Out of bounds (Memory length: [10])" {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}
}

func Test_OP_POINTER_RIGHT(t *testing.T) {

	tape, mem := MakeTapeAndMemory(OP_POINTER_RIGHT)

	if ok, err := tape.Execute(mem); !ok {
		t.Errorf("Unexpected success when calling OP_POINTER_RIGHT.Execute(). %v", err)
	}

	if mem.MemoryPointer != 1 {
		t.Errorf("Memory pointer [%d] is not at expected value [1].", mem.MemoryPointer)
	}

	if tape.InstructionPointer != 0 {
		t.Errorf("Instruction pointer [%d] is not at expected value [0]", tape.InstructionPointer)
	}

	mem.MemoryPointer = 9

	if ok, err := tape.Execute(mem); ok {
		t.Errorf("Unexpected success when calling OP_POINTER_RIGHT.Execute().")
	} else {
		if err.Error() != "OP_POINTER_RIGHT at tape index [0] failed to move memory pointer right. Failed to move memory pointer [9] right. Out of bounds (Memory length: [10])" {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}
}

func Test_OP_WHILE(t *testing.T) {

	tape, mem := MakeTapeAndMemory(OP_WHILE)

	mem.Cells[0] = 1

	if ok, err := tape.Execute(mem); !ok {
		t.Errorf("Unexpected failure when calling OP_WHILE.Execute(). %v", err)
	}

	if mem.MemoryPointer != 0 {
		t.Errorf("Memory pointer [%d] is not at expected value [0].", mem.MemoryPointer)
	}

	if tape.InstructionPointer != 0 {
		t.Errorf("Instruction pointer [%d] is not at expected value [0]", tape.InstructionPointer)
	}

	if len(tape.WhileIndexStack) != 1 {
		t.Errorf("While index stack [%d] does not have expected length [1]", len(tape.WhileIndexStack))
	}

	tape.InstructionPointer = 0
	mem.Cells[0] = 0
	tape.Instructions = SET_TO_ZERO

	if ok, err := tape.Execute(mem); !ok && err != nil {
		t.Errorf("Unexpected failure when calling OP_WHILE.Execute(). %v", err)
	}

	if mem.MemoryPointer != 0 {
		t.Errorf("Memory pointer [%d] is not at expected value [0].", mem.MemoryPointer)
	}

	if tape.InstructionPointer != 1 {
		t.Errorf("Instruction pointer [%d] is not at expected value [1]", tape.InstructionPointer)
	}

	if len(tape.WhileIndexStack) != 1 {
		t.Errorf("While index stack [%d] does not have expected length [1]", len(tape.WhileIndexStack))
	}

	mem.MemoryPointer = 99
	tape.InstructionPointer = 0

	if ok, err := tape.Execute(mem); ok {
		t.Errorf("Unexpected success when calling OP_WHILE.Execute().")
	} else {
		if err.Error() != "OP_WHILE at tape index [0] failed to get current memory cell at index [99] during OP_WHILE evaluation. Memory pointer [99] out of bounds (Memory length: [10])" {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}
}

func Test_OP_WHILE_END(t *testing.T) {

	tape, mem := MakeTapeAndMemory([]rune(SET_TO_ZERO)...)

	mem.Cells[0] = 1
	tape.WhileIndexStack = append(tape.WhileIndexStack, tape.InstructionPointer)
	tape.InstructionPointer = 2

	if ok, err := tape.Execute(mem); !ok {
		t.Errorf("Unexpected failure when calling OP_WHILE_END.Execute(). %v", err)
	}

	if len(tape.WhileIndexStack) != 0 {
		t.Errorf("While index stack [%d] does not have expected length [1]", len(tape.WhileIndexStack))
	}

	if tape.InstructionPointer != -1 {
		t.Errorf("Instruction pointer [%d] is not at expected value [0]", tape.InstructionPointer)
	}

	tape.Instructions = "]"
	tape.InstructionPointer = 0

	if ok, err := tape.Execute(mem); ok {
		t.Errorf("Unexpected success when calling OP_WHILE_END.Execute().")
	} else {
		if err.Error() != "OP_WHILE_END at tape index [0] failed to fallback. Failed to pop while stack." {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}

	mem.Cells[0] = 0
	tape.WhileIndexStack = append(tape.WhileIndexStack, tape.InstructionPointer)
	tape.Instructions = SET_TO_ZERO
	tape.InstructionPointer = 2

	if ok, err := tape.Execute(mem); !ok && err != nil {
		t.Errorf("Unexpected failure when calling OP_WHILE_END.Execute(). %v", err)
	}
	if len(tape.WhileIndexStack) != 0 {
		t.Errorf("While index stack [%d] does not have expected length [1]", len(tape.WhileIndexStack))
	}
	if tape.InstructionPointer != 2 {
		t.Errorf("Instruction pointer [%d] is not at expected value [2]", tape.InstructionPointer)
	}
}

func Test_OP_JUMP(t *testing.T) {

	tape, mem := MakeTapeAndMemory(OP_JUMP)

	mem.BookmarkRegister = 0
	mem.MemoryPointer = 2

	if ok, err := tape.Execute(mem); !ok {
		t.Errorf("Unexpected failure when calling OP_JUMP.Execute(). %v", err)
	}

	if mem.MemoryPointer != 0 {
		t.Errorf("Failed to jump MemoryPointer to bookmark. Expected MemoryPointer to be [0], but was [%d]", mem.MemoryPointer)
	}

	if mem.BookmarkRegister != 2 {
		t.Errorf("Failed to store MemoryPointer to bookmark while jumping. Expected BookmarkRegister to be [2], but was [%d]", mem.BookmarkRegister)
	}

	mem.MemoryPointer = 999
	if ok, err := tape.Execute(mem); ok {
		t.Errorf("Unexpected success when calling OP_JUMP()")
	} else {
		if err.Error() != "OP_JUMP at tape index [0] failed to jump. Failed to jump to bookmark. Current memory pointer [999] out of bounds (Memory length: [10])" {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}

	mem.MemoryPointer = 0
	mem.BookmarkRegister = 999
	if ok, err := tape.Execute(mem); ok {
		t.Errorf("Unexpected success when calling OP_JUMP()")
	} else {
		if err.Error() != "OP_JUMP at tape index [0] failed to jump. Failed to jump to bookmark. Bookmark memory pointer [999] out of bounds (Memory length: [10])" {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}
}

func Test_OP_BOOKMARK(t *testing.T) {

	tape, mem := MakeTapeAndMemory(OP_BOOKMARK)

	mem.MemoryPointer = 1

	if ok, err := tape.Execute(mem); !ok {
		t.Errorf("Unexpected failure when calling OP_BOOKMARK.Execute(). %v", err)
	}

	if mem.BookmarkRegister != 1 {
		t.Errorf("Failed to store bookmark. Expected BookmarkRegister to be [1], but was [%d]", mem.BookmarkRegister)
	}

	mem.MemoryPointer = 999
	if ok, err := tape.Execute(mem); ok {
		t.Errorf("Unexpected success when calling OP_BOOKMARK.Execute()")
	} else {
		if err.Error() != "OP_BOOKMARK at tape index [0] failed to store. Failed to store to bookmark. Current memory pointer [999] out of bounds (Memory length: [10])" {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}
}

func Test_NO_OP(t *testing.T) {
	tape, mem := NewTape("##"), NewMemory(1)

	if ok, err := tape.Execute(mem); !ok {
		t.Errorf("Unexpected failure when calling NO_OP.Execute(). %v", err)
	}

	if tape.InstructionPointer != 0 {
		t.Errorf("Instruction pointer [%d] is not at expected value [0]", tape.InstructionPointer)
	}

	if len(tape.Instructions) != 2 {
		t.Errorf("Expected instruction count to be [2], but was length [%d]", len(tape.Instructions))
	}

	if len(tape.WhileIndexStack) != 0 {
		t.Errorf("Expected while stack to be empty, but was length [%d]", len(tape.WhileIndexStack))
	}

	if mem.Cells[0] != 0 {
		t.Errorf("Memory cell at index [0] (%d) is not at expected value [0].", mem.Cells[0])
	}

	if mem.BookmarkRegister != 0 {
		t.Errorf("Expected BookmarkRegister to be [0], but was [%d]", mem.BookmarkRegister)
	}

	if mem.MemoryPointer != 0 {
		t.Errorf("Expected MemoryPointer to be [0], but was [%d]", mem.MemoryPointer)
	}

	if len(mem.Cells) != 1 {
		t.Errorf("Expected memory cell count to be [1], but was [%d]", len(mem.Cells))
	}

	if mem.CellCount != 1 {
		t.Errorf("Expected memory cell count to be %v, but was %v", 1, mem.CellCount)
	}

}
