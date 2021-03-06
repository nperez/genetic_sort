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
	tape.PushWhile()
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

func TestWhileBlocks(t *testing.T) {

	tape := NewTape("[[-]]")

	if ok, err := tape.PushWhile(); !ok {
		t.Errorf("Unexpected failure when calling Tape.PushWhile(). %v", err)
	}

	tape.InstructionPointer = 999

	if ok, err := tape.PushWhile(); ok {
		t.Errorf("Unexpected success when calling Tape.PushWhile().")
	} else {
		if err.Error() != "Failed to store current InstructionPointer [999] on while stack. Out of bounds (Instruction length: [5]" {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}

	tape.InstructionPointer = 0

	if ok, err := tape.AdvanceToWhileEnd(); !ok {
		t.Errorf("Unexpected failure when calling Tape.AdvanceToWhileEnd(). %v", err)
	}

	if tape.InstructionPointer != 3 {
		t.Errorf("Instruction pointer [%d] is not at expected value [1]", tape.InstructionPointer)
	}

	if len(tape.WhileIndexStack) != 1 {
		t.Errorf("While index stack [%d] does not have expected length [1]", len(tape.WhileIndexStack))
	}

	if ok, err := tape.FallbackToWhileStart(); !ok {
		t.Errorf("Unexpected failure when calling Tape.FallbackToWhileStart(). %v", err)
	}

	if len(tape.WhileIndexStack) != 0 {
		t.Errorf("While index stack [%d] does not have expected length [1]", len(tape.WhileIndexStack))
	}

	if tape.InstructionPointer != 0 {
		t.Errorf("Instruction pointer [%d] is not at expected value [1]", tape.InstructionPointer)
	}

	tape.InstructionPointer = 4

	if ok, err := tape.AdvanceToWhileEnd(); ok {
		t.Errorf("Unexpected success when calling Tape.AdvanceToWhileEnd().")
	} else {
		if err.Error() != "Failed to advance to OP_WHILE_END instruction. Tape end reached." {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}

	tape.InstructionPointer = 0
	tape.Instructions = "###"

	if ok, err := tape.AdvanceToWhileEnd(); ok {
		t.Errorf("Unexpected success when calling Tape.AdvanceToWhileEnd().")
	} else {
		if err.Error() != "Failed to advance to OP_WHILE_END instruction. Tape end reached." {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}

	if ok, err := tape.FallbackToWhileStart(); ok {
		t.Errorf("Unexpected success when calling Tape.FallbackToWhileStart().")
	} else {
		if err.Error() != "Failed to pop while stack." {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}

	tape.WhileIndexStack = []int{999}

	if ok, err := tape.FallbackToWhileStart(); ok {
		t.Errorf("Unexpected success when calling Tape.FallbackToWhileStart().")
	} else {
		if err.Error() != "InstructionPointer [999] from while stack is out of bounds (Instruction length: [3]" {
			t.Errorf("Error string doesn't match: %v", err)
		}
	}
}
