package brainfuck

import (
	"fmt"
	"math"
)

type MemoryConfig struct {
	CellCount uint
}

type Memory struct {
	Cells            []uint8
	CellCount        uint
	MemoryPointer    uint
	BookmarkRegister uint
}

func NewMemory(cell_count uint) *Memory {
	return &Memory{
		Cells:            make([]uint8, cell_count),
		CellCount:        cell_count,
		MemoryPointer:    0,
		BookmarkRegister: 0,
	}
}

func (m *Memory) Reset() {
	for i := 0; i < len(m.Cells); i++ {
		m.Cells[i] = 0
	}
	m.MemoryPointer = 0
	m.BookmarkRegister = 0
}

func (m *Memory) GetCurrentCell() (bool, uint8, error) {
	if m.MemoryPointer < 0 || m.MemoryPointer > m.CellCount-1 {
		return false, 0, fmt.Errorf("Memory pointer [%d] out of bounds (Memory length: [%d])", m.MemoryPointer, len(m.Cells))
	}
	return true, m.Cells[m.MemoryPointer], nil
}

func (m *Memory) MovePointerLeft() (bool, error) {
	if m.MemoryPointer == 0 {
		return false, fmt.Errorf("Failed to move memory pointer [%d] left. Out of bounds (Memory length: [%d])", m.MemoryPointer, len(m.Cells))
	}
	m.MemoryPointer = m.MemoryPointer - 1
	return true, nil
}

func (m *Memory) MovePointerRight() (bool, error) {
	if m.MemoryPointer == m.CellCount-1 {
		return false, fmt.Errorf("Failed to move memory pointer [%d] right. Out of bounds (Memory length: [%d])", m.MemoryPointer, len(m.Cells))
	}
	m.MemoryPointer = m.MemoryPointer + 1
	return true, nil
}

func (m *Memory) StoreBookmark() (bool, error) {
	if m.MemoryPointer < 0 || m.MemoryPointer > m.CellCount-1 {
		return false, fmt.Errorf("Failed to store to bookmark. Current memory pointer [%d] out of bounds (Memory length: [%d])", m.MemoryPointer, len(m.Cells))
	}
	m.BookmarkRegister = m.MemoryPointer
	return true, nil
}

func (m *Memory) BookmarkJump() (bool, error) {
	if m.MemoryPointer < 0 || m.MemoryPointer > m.CellCount-1 {
		return false, fmt.Errorf("Failed to jump to bookmark. Current memory pointer [%d] out of bounds (Memory length: [%d])", m.MemoryPointer, len(m.Cells))
	}

	if m.BookmarkRegister < 0 || m.BookmarkRegister > m.CellCount-1 {
		return false, fmt.Errorf("Failed to jump to bookmark. Bookmark memory pointer [%d] out of bounds (Memory length: [%d])", m.BookmarkRegister, len(m.Cells))
	}
	current := m.MemoryPointer
	m.MemoryPointer = m.BookmarkRegister
	m.BookmarkRegister = current

	return true, nil
}

func (m *Memory) Increment() (bool, error) {
	if ok, val, err := m.GetCurrentCell(); ok {
		if val < math.MaxUint8 {
			m.Cells[m.MemoryPointer] = val + 1
			return true, nil
		} else {
			return false, fmt.Errorf("Increment failed. Cell value [%d] at UpperBound [%d]", val, math.MaxUint8)
		}
	} else {
		return false, err
	}
}

func (m *Memory) Decrement() (bool, error) {
	if ok, val, err := m.GetCurrentCell(); ok {
		if val > 0 {
			m.Cells[m.MemoryPointer] = val - 1
			return true, nil
		} else {
			return false, fmt.Errorf("Decrement failed. Cell value [%d] at LowerBound [%d]", val, 0)
		}
	} else {
		return false, err
	}
}
