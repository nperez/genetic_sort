package brainfuck

import (
	"fmt"
)

type Machine struct {
	Tape             *Tape
	Memory           *Memory
	InstructionCount int
}

func NewMachine(program string, config *MemoryConfig) *Machine {
	return &Machine{
		Tape:             NewTape(program),
		Memory:           NewMemoryFromConfig(config),
		InstructionCount: 0,
	}
}

func (m *Machine) LoadMemory(input []int) (bool, error) {

	if len(input) > len(m.Memory.Cells) {
		return false, fmt.Errorf("Failed to load memory. Input length [%d] is greater than memory capacity [%d]", len(input), len(m.Memory.Cells))
	}

	for i, val := range input {
		if !m.Memory.CellInBounds(val) {
			return false, fmt.Errorf("Failed to load memory. Input value [%d] is out of bounds [%d, %d]", val, m.Memory.MemoryConfig.LowerBound, m.Memory.MemoryConfig.UpperBound)
		}

		m.Memory.Cells[i] = val
	}
	return true, nil
}

func (m *Machine) ReadMemory(count int) (bool, []int, error) {

	if count > len(m.Memory.Cells) {
		return false, []int{}, fmt.Errorf("Failed to read memory. Read count [%d] is greater than memory capacity [%d]", count, len(m.Memory.Cells))
	}

	return true, m.Memory.Cells[0:count], nil
}

func (m *Machine) Run() (bool, error) {

	var exception error

	halt := false
	for !halt {
		if ok, err := m.Tape.Execute(m.Memory); !ok {
			halt = true
			exception = err
		}
		m.InstructionCount = m.InstructionCount + 1
	}

	if exception != nil {
		return false, exception
	}

	return true, nil
}
