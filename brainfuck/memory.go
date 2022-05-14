package brainfuck

import (
	"fmt"
)

type MemoryConfig struct {
	CellCount  int
	UpperBound int
	LowerBound int
}

type Memory struct {
	Cells            []int
	MemoryConfig     *MemoryConfig
	MemoryPointer    int
	BookmarkRegister int
}

func NewMemoryFromConfig(c *MemoryConfig) *Memory {
	return &Memory{
		Cells:            make([]int, c.CellCount),
		MemoryConfig:     c,
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

func (m *Memory) GetCurrentCell() (bool, int, error) {
	if ok := m.MemoryInBounds(m.MemoryPointer); !ok {
		return false, -1, fmt.Errorf("Memory pointer [%d] out of bounds (Memory length: [%d])", m.MemoryPointer, len(m.Cells))
	}
	return true, m.Cells[m.MemoryPointer], nil
}

func (m *Memory) MemoryInBounds(new_val int) bool {
	return new_val >= 0 && new_val <= len(m.Cells)-1
}

func (m *Memory) MovePointerLeft() (bool, error) {
	if !m.MemoryInBounds(m.MemoryPointer - 1) {
		return false, fmt.Errorf("Failed to move memory pointer [%d] left. Out of bounds (Memory length: [%d])", m.MemoryPointer, len(m.Cells))
	}
	m.MemoryPointer = m.MemoryPointer - 1
	return true, nil
}

func (m *Memory) MovePointerRight() (bool, error) {
	if !m.MemoryInBounds(m.MemoryPointer + 1) {
		return false, fmt.Errorf("Failed to move memory pointer [%d] right. Out of bounds (Memory length: [%d])", m.MemoryPointer, len(m.Cells))
	}
	m.MemoryPointer = m.MemoryPointer + 1
	return true, nil
}

func (m *Memory) StoreBookmark() (bool, error) {
	if !m.MemoryInBounds(m.MemoryPointer) {
		return false, fmt.Errorf("Failed to store to bookmark. Current memory pointer [%d] out of bounds (Memory length: [%d])", m.MemoryPointer, len(m.Cells))
	}
	m.BookmarkRegister = m.MemoryPointer
	return true, nil
}

func (m *Memory) BookmarkJump() (bool, error) {
	if !m.MemoryInBounds(m.MemoryPointer) {
		return false, fmt.Errorf("Failed to jump to bookmark. Current memory pointer [%d] out of bounds (Memory length: [%d])", m.MemoryPointer, len(m.Cells))
	}

	if !m.MemoryInBounds(m.BookmarkRegister) {
		return false, fmt.Errorf("Failed to jump to bookmark. Bookmark memory pointer [%d] out of bounds (Memory length: [%d])", m.BookmarkRegister, len(m.Cells))
	}
	current := m.MemoryPointer
	m.MemoryPointer = m.BookmarkRegister
	m.BookmarkRegister = current

	return true, nil
}

func (m *Memory) CellInBounds(new_val int) bool {
	return new_val >= m.MemoryConfig.LowerBound && new_val <= m.MemoryConfig.UpperBound
}

func (m *Memory) Increment() (bool, error) {
	if ok, val, err := m.GetCurrentCell(); ok {
		if ok := m.CellInBounds(val + 1); ok {
			m.Cells[m.MemoryPointer] = val + 1
			return true, nil
		} else {
			return false, fmt.Errorf("Increment failed. Cell value [%d] at UpperBound [%d]", val, m.MemoryConfig.UpperBound)
		}
	} else {
		return false, err
	}
}

func (m *Memory) Decrement() (bool, error) {
	if ok, val, err := m.GetCurrentCell(); ok {
		if ok := m.CellInBounds(val - 1); ok {
			m.Cells[m.MemoryPointer] = val - 1
			return true, nil
		} else {
			return false, fmt.Errorf("Decrement failed. Cell value [%d] at LowerBound [%d]", val, m.MemoryConfig.LowerBound)
		}
	} else {
		return false, err
	}
}
