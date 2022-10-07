package brainfuck

import (
	//"fmt"
	"testing"
)

func TestNewTape(t *testing.T) {
	tape := NewTape(SET_TO_ZERO)
	if tape == nil {
		t.Errorf("NewTape returned nil")
	}
}

func TestTapeReset(t *testing.T) {
	tape := NewTape(SET_TO_ZERO)
	tape.Advance()
	tape.WhileIndexStack = append(tape.WhileIndexStack, tape.InstructionPointer)
	tape.Reset()
	if tape.InstructionPointer != 0 {
		t.Errorf("Reset didn't reset InstructionPointer: %v", tape.InstructionPointer)
	}

	if len(tape.WhileIndexStack) != 0 {
		t.Errorf("Reset didn't reset WhileIndexStack: %v", tape.WhileIndexStack)
	}

}

func TestTapeAdvance(t *testing.T) {
	tape := NewTape(SET_TO_ZERO)
	tape.Advance()
	if tape.InstructionPointer != 1 {
		t.Errorf("Advance apparently didn't increment the InstructionPointer [%d]", tape.InstructionPointer)
	}
}

func TestGetCurrentInstruction(t *testing.T) {
	tape := NewTape(SET_TO_ZERO)
	if ok, op, err := tape.GetCurrentInstruction(); !ok {
		t.Errorf("GetCurrentInstruction returned !ok with OP |%v| and err |%v|", op, err)
	} else {
		if err != nil {
			t.Errorf("GetCurrentInstruction returned ok but with a defined err |%v|", err)
		}

		if op != OP_WHILE {
			t.Errorf("GetCurrentInstruction returned unexpected OP |%v|, expected OP |[|", op)
		}
	}

	tape.InstructionPointer = 10

	if ok, op, err := tape.GetCurrentInstruction(); ok {
		t.Errorf("GetCurrentInstruction returned ok with OP |%v| and err |%v| but expected !ok, OP |NO_OP|, 'out of bounds'", op, err)
	} else {
		if err == nil {
			t.Errorf("GetCurrentInstruction returned !ok but with an undefined err but expected 'out of bounds'")
		}

		if op != NO_OP {
			t.Errorf("GetCurrentInstruction returned unexpected OP |%v|, expected OP |NO_OP|", op)
		}
	}
}
