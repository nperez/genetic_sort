package genetic_sort

import (
	//bf "nickandperla.net/brainfuck"
	rnd "math/rand"
	mop "reflect"
	//re "regexp"
	str "strings"
	test "testing"
)

var SEED42_INSTRUCTION_STRING string = `*[>]^[-^+^][<]>*[>]^<[-^+^]>[-<+>]^[-^+^][>]*[>]^[-^+^][-]*[<]^[-^+^]>*[>]^<[-^+^]>[-<+>]^[-^+^]>*[>]^<[-^+^]>[-<+>]^[-^+^]*[>]^[-^+^]>[-<+>]^[-^+^][-]*[>]^[-^+^]>[-<+>]^[-^+^][<]>*[>]^<[-^+^]>[-<+>]^[-^+^]*[>]^[-^+^]*[<]^[-^+^]*[>]^[-^+^]>[-<+>]^[-^+^]*[<]^[-^+^]*[>]^[-^+^]>[-<+>]^[-^+^]*[>]^[-^+^]*[>]^[-^+^][-]*[>]^[-^+^][-]>*[>]^<[-^+^]>[-<+>]^[-^+^][-][-][<][<]*[<]^[-^+^][-][<]*[>]^[-^+^]>[-<+>]^[-^+^]*[>]^[-^+^]>[-<+>]^[-^+^][>][>][-]>*[>]^<[-^+^]>[-<+>]^[-^+^]*[>]^[-^+^]>[-<+>]^[-^+^]*[>]^[-^+^]>[-<+>]^[-^+^]>*[>]^<[-^+^]>[-<+>]^[-^+^]*[>]^[-^+^]>*[>]^<[-^+^]>[-<+>]^[-^+^][<][<][<][>]*[<]^[-^+^][-][-]`

func TestNewUnitFuncs(t *test.T) {
	// Fixed seed to get determinent results
	rnd.Seed(42)
	config := &UnitConfig{
		MutationChance:    0.25,
		InstructionCount:  10,
		InstructionConfig: &InstructionConfig{OpSetCount: 5},
	}

	unit1 := NewUnitFromConfig(config)

	var sb str.Builder
	for _, instruction := range unit1.Instructions {
		sb.WriteString(instruction.Ops)
	}

	if SEED42_INSTRUCTION_STRING != sb.String() {
		t.Errorf("Unit instructions do not match expected:\nExpected: %v\nActual: %v ",
			SEED42_INSTRUCTION_STRING, sb.String())
	}

	if len(unit1.Instructions) != 10 {
		t.Errorf("Gene count [%v] is not expected value [10]", len(unit1.Instructions))
	}

	if unit1.MutationChance != 0.25 {
		t.Errorf("MutationChance [%v] is not expected value [0.25]", unit1.MutationChance)
	}
}

func TestUnitClone(t *test.T) {
	rnd.Seed(42)

	config := &UnitConfig{
		MutationChance:    0.25,
		InstructionCount:  10,
		InstructionConfig: &InstructionConfig{OpSetCount: 5},
	}

	unit1 := NewUnitFromConfig(config)

	unit2 := unit1.Clone()

	var sb str.Builder
	for _, instruction := range unit2.Instructions {
		sb.WriteString(instruction.Ops)
	}

	if SEED42_INSTRUCTION_STRING != sb.String() {
		t.Errorf("Unit instructions do not match expected:\nExpected: %v\nActual: %v ",
			SEED42_INSTRUCTION_STRING, sb.String())
	}

	if len(unit2.Instructions) != 10 {
		t.Errorf("Gene count [%v] is not expected value [10]", len(unit2.Instructions))
	}

	if unit2.MutationChance != 0.25 {
		t.Errorf("MutationChance [%v] is not expected value [0.25]", unit2.MutationChance)
	}

	if !mop.DeepEqual(unit1, unit2) {
		t.Errorf("Unit clone does not match original:\nOriginal: %v\nActual: %v ", unit1, unit2)
	}
}

func TestMitosis(t *test.T) {
	rnd.Seed(42)

	config := &UnitConfig{
		MutationChance:    0.25,
		InstructionCount:  10,
		InstructionConfig: &InstructionConfig{OpSetCount: 5},
	}

	unit1 := NewUnitFromConfig(config)

	unit2 := unit1.Mitosis()

	if mop.DeepEqual(unit1, unit2) {
		t.Errorf("Unexpected DeepEqual between original and offspring units from Mitosis()")
	}

	expectedMutations := [][]int{{0, 1}, {1, 1}, {2, 0}, {3, 0}, {4, 1}, {5, 0}, {6, 0}, {7, 0}, {8, 0}, {9, 0}}

	actualMutations := [][]int{}
	for i, gene := range unit2.Instructions {
		actualMutations = append(actualMutations, []int{i, len(gene.Mutations)})
	}

	if !mop.DeepEqual(expectedMutations, actualMutations) {
		t.Errorf("Unexpected gene mutations from calling Unit.Mitosis()\nExpected: %v\nActual:%v", expectedMutations, actualMutations)
	}
}
