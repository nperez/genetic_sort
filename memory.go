package genetic_sort

type MemoryCellConfig struct {
	LowerBound int
	UpperBound int
}

type MemoryCell struct {
	LowerBound int
	UpperBound int
	Value      int
}

func NewMemoryCellFromConfig(c *MemoryCellConfig) *MemoryCell {
	return &MemoryCell{
		LowerBound: c.LowerBound,
		UpperBound: c.UpperBound,
		Value:      0,
	}
}

func (mc *MemoryCell) InBounds() bool {
	return mc.Value > mc.LowerBound && mc.Value < mc.UpperBound
}

func (mc *MemoryCell) Increment() bool {
	mc.Value = mc.Value + 1
	return mc.InBounds()
}

func (mc *MemoryCell) Decrement() bool {
	mc.Value = mc.Value - 1
	return mc.InBounds()
}

type MemoryConfig struct {
	CellCount  int
	CellConfig *MemoryCellConfig
}

type Memory struct {
	Cells            []*MemoryCell
	CurrentCellIndex int
}

func NewMemoryFromConfig(c *MemoryConfig) *Memory {
	mem := make([]*MemoryCell, c.CellCount)
	for i := 0; i < c.CellCount; {
		mem[i] = NewMemoryCellFromConfig(c.CellConfig)
	}

	return &Memory{
		Cells:       mem,
		CurrentCell: 0,
	}
}

func (m *Memory) GetCurrentCell() *MemoryCell {
	return m.Cells[m.CurrentCellIndex]
}

func (m *Memory) InBounds() bool {
	return m.CurrentCellIndex > 0 && m.CurrentCellIndex < len(m.Cells)-1
}

func (m *Memory) MovePointerLeft() bool {
	m.CurrentCellIndex = m.CurrentCellIndex - 1
	return m.InBounds()
}

func (m *Memory) MovePointerRight() bool {
	m.CurrentCellIndex = m.CurrentCellIndex + 1
	return m.InBounds()
}
