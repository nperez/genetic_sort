package brainfuck

import (
	"testing"
)

func TestNewMemoryFromConfig(t *testing.T) {
	memory := NewMemory(100)
	if memory == nil {
		t.Errorf("NewMemoryFromConfig returned nil")
	}
}

func TestMemoryReset(t *testing.T) {
	memory := NewMemory(100)
	memory.Cells[0] = 1
	memory.MemoryPointer = 20
	memory.BookmarkRegister = 21

	memory.Reset()
	for i := 0; i < len(memory.Cells); i++ {
		if memory.Cells[i] != 0 {
			t.Fatalf("Memory cells didn't reset: %v", memory.Cells)
		}
	}

	if memory.MemoryPointer != 0 {
		t.Errorf("Memory pointer didn't reset: %v", memory.MemoryPointer)
	}

	if memory.BookmarkRegister != 0 {
		t.Errorf("Memory bookmark register didn't reset: %v", memory.BookmarkRegister)
	}
}

func TestIncrement(t *testing.T) {
	memory := NewMemory(100)
	if ok, err := memory.Increment(); !ok {
		t.Errorf("Increment failed: %v", err)
	}

	if ok, val, err := memory.GetCurrentCell(); !ok || val != 1 {
		t.Errorf("Increment failed. Value is [%d]. Expected value to be [1]. %v", val, err)
	}

	memory.Cells[0] = 255
	if ok, _ := memory.Increment(); ok {
		t.Errorf("Increment succeeded when it shouldn't.")
	} else {
	}
}

func TestDecrement(t *testing.T) {
	memory := NewMemory(100)
	memory.Cells[0] = 2
	if ok, err := memory.Decrement(); !ok {
		t.Errorf("Increment failed: %v", err)
	}

	if ok, val, err := memory.GetCurrentCell(); !ok || val != 1 {
		t.Errorf("Increment failed. Value is [%d]. Expected value to be [1]. %v", val, err)
	}

	memory.Cells[0] = 0
	if ok, _ := memory.Decrement(); ok {
		t.Errorf("Decrement succeeded when it shouldn't.")
	} else {
	}
}

func TestMovePointerRight(t *testing.T) {
	memory := NewMemory(100)
	if ok, err := memory.MovePointerRight(); !ok {
		t.Errorf("Moving memory pointer to the right failed. %v", err)
	}

	if memory.MemoryPointer != 1 {
		t.Errorf("Moving memory pointer to the right failed. Expected MemoryPointer to be [1] but was [%d]", memory.MemoryPointer)
	}

	memory.MemoryPointer = 99

	if ok, _ := memory.MovePointerRight(); ok {
		t.Errorf("Moving memory pointer to the right successed unexpectedly. Expected MemoryPointer to be out of bounds but is [%d] and CellCount is [%d]", memory.MemoryPointer, memory.CellCount)
	}
}

func TestMovePointerLeft(t *testing.T) {
	memory := NewMemory(100)
	memory.MemoryPointer = 99
	if ok, err := memory.MovePointerLeft(); !ok {
		t.Errorf("Moving memory pointer to the left failed. %v", err)
	}

	if memory.MemoryPointer != 98 {
		t.Errorf("Moving memory pointer to the left failed. Expected MemoryPointer to be [98] but was [%d]", memory.MemoryPointer)
	}

	memory.MemoryPointer = 0
	if ok, _ := memory.MovePointerLeft(); ok {
		t.Errorf("Moving memory pointer to the left successed unexpectedly. Expected MemoryPointer to be out of bounds but is [%d]", memory.MemoryPointer)
	}
}

func TestBookmarks(t *testing.T) {
	memory := NewMemory(100)

	memory.MemoryPointer = 10
	if ok, err := memory.StoreBookmark(); !ok {
		t.Errorf("Unexpected failure when calling Memory.StoreBookmark(). %v", err)
	}

	if memory.BookmarkRegister != 10 {
		t.Errorf("Failed to store bookmark. Expected BookmarkRegister to be [10], but was [%d]", memory.BookmarkRegister)
	}

	memory.MemoryPointer = 50
	if ok, err := memory.BookmarkJump(); !ok {
		t.Errorf("Unexpected failure when calling Memory.BookmarkJump(). %v", err)
	}

	if memory.MemoryPointer != 10 {
		t.Errorf("Failed to jump MemoryPointer to bookmark. Expected MemoryPointer to be [10], but was [%d]", memory.MemoryPointer)
	}

	if memory.BookmarkRegister != 50 {
		t.Errorf("Failed to store MemoryPointer to bookmark while jumping. Expected BookmarkRegister to be [50], but was [%d]", memory.BookmarkRegister)
	}

	memory.MemoryPointer = 999
	if ok, err := memory.StoreBookmark(); ok {
		t.Errorf("Unexpected success when calling Memory.StoreBookmark()")
	} else {
		if err.Error() != "Failed to store to bookmark. Current memory pointer [999] out of bounds (Memory length: [100])" {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}

	if ok, err := memory.BookmarkJump(); ok {
		t.Errorf("Unexpected success when calling Memory.BookmarkJump()")
	} else {
		if err.Error() != "Failed to jump to bookmark. Current memory pointer [999] out of bounds (Memory length: [100])" {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}

	memory.MemoryPointer = 0
	memory.BookmarkRegister = 999

	if ok, err := memory.BookmarkJump(); ok {
		t.Errorf("Unexpected success when calling Memory.BookmarkJump()")
	} else {
		if err.Error() != "Failed to jump to bookmark. Bookmark memory pointer [999] out of bounds (Memory length: [100])" {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}
}
