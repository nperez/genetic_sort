package brainfuck

import (
	"fmt"
)

var ErrMaxInstructionExecutionCountReached error = fmt.Errorf("Instruction execution count limit reached")

type Machine struct {
	Tape             *Tape
	Memory           *Memory
	Config           *MachineConfig
	InstructionCount uint
}

type MachineConfig struct {
	MaxInstructionExecutionCount uint `toml:"max_instruction_execution_count"`
	MemoryCellCount              uint `toml:"memory_cell_count"`
}

func NewMachine(mc *MachineConfig) *Machine {
	return &Machine{
		Memory: NewMemory(mc.MemoryCellCount),
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

func (m *Machine) LoadMemory(input []uint8) (bool, error) {

	if uint(len(input)) > m.Config.MemoryCellCount {
		return false, fmt.Errorf("Failed to load memory. Input length [%d] is greater than memory capacity [%d]", len(input), len(m.Memory.Cells))
	}

	for i, val := range input {
		m.Memory.Cells[i] = val
	}
	return true, nil
}

func (m *Machine) ReadMemory(count uint) (bool, []uint8, error) {

	if count > uint(len(m.Memory.Cells)) {
		return false, []uint8{}, fmt.Errorf("Failed to read memory. Read count [%d] is greater than memory capacity [%d]", count, len(m.Memory.Cells))
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
			exception = ErrMaxInstructionExecutionCountReached
		}

		if !m.Tape.Advance() {
			halt = true
		}
	}

	if exception != nil {
		return false, exception
	}

	return true, nil
}
