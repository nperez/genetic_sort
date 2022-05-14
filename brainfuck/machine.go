package brainfuck

import (
	"fmt"
)

type Machine struct {
	Tape             *Tape
	Memory           *Memory
	Config           *MachineConfig
	InstructionCount uint
}

type MachineConfig struct {
	MaxInstructionExecutionCount uint
	MemoryConfig                 *MemoryConfig
}

func NewMachine(mc *MachineConfig) *Machine {
	return &Machine{
		Memory: NewMemoryFromConfig(mc.MemoryConfig),
		Config: mc,
	}
}

func (m *Machine) Reset() {
	m.Tape.Reset()
	m.Memory.Reset()
}

func (m *Machine) LoadProgram(instructions string) {
	if m.Tape == nil {
		m.Tape = NewTape(instructions)
	} else {
		m.Tape.Instructions = instructions
		m.Tape.Reset()
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
		if m.InstructionCount >= m.Config.MaxInstructionExecutionCount {
			halt = true
			exception = fmt.Errorf("Instruction execution count limit reached: %v", m.InstructionCount)
		}
	}

	if exception != nil {
		return false, exception
	}

	return true, nil
}
