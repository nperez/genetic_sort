package genetic_sort

import (
	bf "nickandperla.net/brainfuck"
	mop "reflect"
	re "regexp"
	"testing"
)

func TestNewInstruction(t *testing.T) {
	instruct1 := NewInstructionFromConfig(&InstructionConfig{OpSetCount: 10})

	if instruct1 == nil {
		t.Errorf("Unexpected failure from NewInstructionFromConfig(). Returned nil")
	}

	if instruct1.Age != 0 {
		t.Errorf("Age [%v] is not equal to 0", instruct1.Age)
	}

	var matcher = re.MustCompile(`\s*[><\]\[+-^*#]+\s*`)
	for _, op := range instruct1.ToProgram() {
		if !matcher.MatchString(string(op)) {
			t.Errorf("Somehow generated invalid machine ops: %v", string(instruct1.ToProgram()))
		}
	}

	if instruct1.InitialOpSet != instruct1.Ops {
		t.Errorf("InitialOpSet and Ops disagree. InitialOpSet: [%v], Ops: [%v]", []byte(instruct1.InitialOpSet), []byte(instruct1.Ops))
	}
}

func TestInstructionClone(t *testing.T) {

	instruct1 := NewInstructionFromConfig(&InstructionConfig{OpSetCount: 10})

	if instruct1 == nil {
		t.Errorf("Unexpected failure from NewInstructionFromConfig(). Returned nil")
	}

	instruct2 := instruct1.Clone()
	if instruct2 == nil {
		t.Errorf("Clone() returned nil")
	}

	if !mop.DeepEqual(instruct1, instruct2) {
		t.Errorf("Cloned Instruction [%v] differs from original [%v]", instruct2, instruct1)
	}
}

func TestInstructionStringer(t *testing.T) {

	instruct1 := NewInstructionFromConfig(&InstructionConfig{OpSetCount: 10})

	if instruct1 == nil {
		t.Errorf("Unexpected failure from NewInstructionFromConfig(). Returned nil")
	}

	instruct2 := instruct1.Clone()
	if instruct2 == nil {
		t.Errorf("Clone() returned nil")
	}

	str1, str2 := instruct1.Ops, instruct2.Ops
	if len(str1) != len(instruct1.Ops) {
		t.Errorf("Stringified instruction length [%v] doesn't match Ops length [%v]",
			len(str1), len(instruct1.Ops))
	}
	if len(str2) != len(instruct2.Ops) {
		t.Errorf("Stringified instruction length [%v] doesn't match Ops length [%v]",
			len(str2), len(instruct2.Ops))
	}

	if str1 != str2 {
		t.Errorf("Cloned instructions .String() do not match:\ninstruct1: [%v]\ninstruct2: [%v]",
			str1, str2)
	}
}

func TestInstructionAge(t *testing.T) {

	instruct1 := NewInstructionFromConfig(&InstructionConfig{OpSetCount: 10})
	if instruct1 == nil {
		t.Errorf("Unexpected failure from NewInstructionFromConfig(). Returned nil")
	}

	if instruct1.Age != 0 {
		t.Errorf("Age [%v] is not equal to 0", instruct1.Age)
	}

	instruct1.IncrementAge()

	if instruct1.Age != 1 {
		t.Errorf("Age [%v] is not equal to 1", instruct1.Age)
	}
}

func TestMakeOpsSmallAndBig(t *testing.T) {
	ops := bf.MOVE_TO_ZERO_LEFT
	compressed := makeOpsSmall(ops)
	uncompressed := makeOpsBig(string(compressed))

	if !mop.DeepEqual(ops, string(uncompressed)) {
		t.Errorf("Failed to roundtrip Ops for database encoding.\nOrig: %v\nCompressed: %v\nUncompressed: %v\n",
			ops, compressed, string(uncompressed))
	}
}

func TestInstructionsToProgram(t *testing.T) {

	var ins Instructions = make(Instructions, 3)
	for i := 0; i < 3; i++ {
		ins[i] = NewInstruction(bf.MOVE_TO_ZERO_LEFT)
	}

	move3x := `*[<]^[-^+^]*[<]^[-^+^]*[<]^[-^+^]`

	check := ins.ToProgram()

	if move3x != string(check) {
		t.Errorf("Instructions.ToProgram() failed to produce expected program.\nGot: %v\nExpected: %v", string(check), move3x)
	}

}
