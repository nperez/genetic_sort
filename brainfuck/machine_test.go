package brainfuck

import (
	"reflect"
	"testing"
)

func TestBasicMachine(t *testing.T) {
	m := NewMachine(string(SET_TO_ZERO), &MemoryConfig{CellCount: 100, UpperBound: 255, LowerBound: 0})

	if m == nil {
		t.Errorf("NewMachine returned nil")
	}
}

func TestLoadMemory(t *testing.T) {
	m := NewMachine(string(SET_TO_ZERO), &MemoryConfig{CellCount: 1, UpperBound: 255, LowerBound: 0})

	if ok, err := m.LoadMemory([]int{1}); !ok {
		t.Errorf("Unexpected failure calling Machine.LoadMemory. %v", err)
	}

	if m.Memory.Cells[0] != 1 {
		t.Errorf("Failed to store value. Expected memory cell index [0] value [%d] isn't [1]", m.Memory.Cells[0])
	}

	if ok, err := m.LoadMemory([]int{1, 2}); ok {
		t.Errorf("Unexpected success calling Machine.LoadMemory. CellCount 1, input lenth is 2")
	} else {
		if err.Error() != "Failed to load memory. Input length [2] is greater than memory capacity [1]" {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}

	if ok, err := m.LoadMemory([]int{256}); ok {
		t.Errorf("Unexpected success calling Machine.LoadMemory. Input value is 256, UpperBound is 255")
	} else {
		if err.Error() != "Failed to load memory. Input value [256] is out of bounds [0, 255]" {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}
}

func TestReadMemory(t *testing.T) {
	m := NewMachine(string(SET_TO_ZERO), &MemoryConfig{CellCount: 1, UpperBound: 255, LowerBound: 0})

	m.Memory.Cells[0] = 1

	if ok, values, err := m.ReadMemory(1); !ok {
		t.Errorf("Unexpected failure calling Machine.ReadMemory. %v", err)
	} else {
		if len(values) > 1 {
			t.Errorf("Return values length [%d] is greater than 1", len(values))
		}

		if values[0] != 1 {
			t.Errorf("Returned value [%d] is not 1", values[0])
		}
	}

	if ok, _, err := m.ReadMemory(2); ok {
		t.Errorf("Unexpected success calling Machine.ReadMemory")
	} else {
		if err.Error() != "Failed to read memory. Read count [2] is greater than memory capacity [1]" {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}
}

func TestBasicMachineLoadRunRead(t *testing.T) {

	m := NewMachine(string(SET_TO_ZERO), &MemoryConfig{CellCount: 1, UpperBound: 255, LowerBound: 0})

	if ok, err := m.LoadMemory([]int{1}); !ok {
		t.Errorf("Unexpected failure calling Machine.LoadMemory(). %v", err)
	}

	if ok, err := m.Run(); !ok {
		t.Errorf("Unexpected failure calling Machine.Run(). %v", err)
	}

	if ok, val, err := m.ReadMemory(1); !ok {
		t.Errorf("Unexpected failure calling Machine.ReadMemory. %v", err)
	} else {
		if len(val) > 1 {
			t.Errorf("Return values length [%d] is greater than 1", len(val))
		}

		if val[0] != 0 {
			t.Errorf("Returned value [%d] is not 0", val[0])
		}
	}
}

func TestHelloWorldMachineLoadRunRead(t *testing.T) {

	m := NewMachine("++++++++[>++++[>++>+++>+++>+<<<<-]>+>+>->>+[<]<-]", &MemoryConfig{CellCount: 100, UpperBound: 255, LowerBound: 0})

	if ok, err := m.LoadMemory([]int{0}); !ok {
		t.Errorf("Unexpected failure calling Machine.LoadMemory(). %v", err)
	}

	if ok, err := m.Run(); !ok {
		t.Errorf("Unexpected failure calling Machine.Run(). %v \nINSTRUCTION COUNTER: %v \nMEMORY DUMP:\n%v\n", err, m.InstructionCount, m.Memory.Cells)
	}

	if ok, val, err := m.ReadMemory(7); !ok {
		t.Errorf("Unexpected failure calling Machine.ReadMemory. %v", err)
	} else {
		if len(val) != 7 {
			t.Errorf("Return values length [%d] is not 7", len(val))
		}

		if !reflect.DeepEqual(val, [...]int{0, 0, 72, 104, 88, 32, 8}) {
			t.Errorf("Returned value [%v] is not equal to expected [%v]", val, [...]int{0, 0, 72, 104, 88, 32, 8})
		}
	}
}
