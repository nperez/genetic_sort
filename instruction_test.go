package genetic_sort

import (
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
	for i, op := range instruct1.Ops {
		if !matcher.MatchString(string(op)) {
			t.Errorf("Somehow generated invalid machine ops: %v", instruct1.Ops)
		}
		if instruct1.InitialOpSet[i] != byte(op) {
			t.Errorf("InitialOpSet and Ops disagree at index [%v]. InitialOpSet: [%v], Ops: [%v]", i, instruct1.InitialOpSet[i], op)
		}
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
