package genetic_sort

import (
	"database/sql"
	"reflect"
	t "testing"

	bf "nickandperla.net/brainfuck"

	_ "github.com/glebarez/go-sqlite"
)

func TestPopulationConfigRoundTrip(t *t.T) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Failed to open in-memory DB: %v", err)
	}
	defer db.Close()

	if err := createTestSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	original := &PopulationConfig{
		UnitCount:        200,
		CarryingCapacity: 150,
		Elitism:          10,
		MaxOffspring:     3,
		UnitConfig: &UnitConfig{
			MutationChance:   0.35,
			InstructionCount: 15,
			InstructionConfig: &InstructionConfig{
				OpSetCount: 8,
			},
			Lifespan: 50,
		},
		EvaluatorConfig: &EvaluatorConfig{
			MachineConfig: &bf.MachineConfig{
				MaxInstructionExecutionCount: 5000,
				MemoryCellCount:              30,
			},
			InputCellCount:  12,
			OutputCellCount: 12,
			InputCellStart:  2,
			InputCellStep:   10,
		},
		SelectorConfig: &SelectorConfig{
			MachineRun:           true,
			SetFidelity:          60,
			Sortedness:           40,
			InstructionCount:     200,
			InstructionsExecuted: 8000,
		},
		FitnessConfig: &FitnessConfig{
			SortednessPriority:  1,
			SetFidelityPriority: 2,
			EfficiencyPriority:  3,
		},
	}

	pop := NewPopulationFromConfig(original)
	pop.ID = 1
	if err := insertPopulation(db, pop); err != nil {
		t.Fatalf("Failed to create population: %v", err)
	}

	// Load it back
	loaded := &Population{}
	row := db.QueryRow(`SELECT id, current_generation,
		unit_count, synthesis_pool, carrying_capacity, elitism, max_offspring,
		unit_mutation_chance, unit_instruction_count, unit_ins_op_set_count, unit_lifespan,
		eval_machine_max_instruction_execution_count, eval_machine_memory_cell_count,
		eval_input_cell_count, eval_output_cell_count, eval_synthesis_input_cell_count,
		eval_input_cell_start, eval_input_cell_step, eval_eval_rounds,
		sel_machine_run, sel_set_fidelity, sel_sortedness,
		sel_set_fidelity_start, sel_set_fidelity_step, sel_sortedness_start, sel_sortedness_step,
		sel_instruction_count, sel_instructions_executed,
		fit_sortedness_priority, fit_set_fidelity_priority, fit_efficiency_priority
		FROM populations WHERE id = ?`, pop.ID)

	if err := scanPopulation(row, loaded); err != nil {
		t.Fatalf("Failed to load population: %v", err)
	}

	if !reflect.DeepEqual(original, loaded.PopulationConfig) {
		t.Errorf("PopulationConfig round-trip mismatch.\nOriginal: %+v\nLoaded:   %+v", original, loaded.PopulationConfig)
		if !reflect.DeepEqual(original.UnitConfig, loaded.PopulationConfig.UnitConfig) {
			t.Errorf("UnitConfig mismatch.\nOriginal: %+v\nLoaded:   %+v", original.UnitConfig, loaded.PopulationConfig.UnitConfig)
		}
		if !reflect.DeepEqual(original.EvaluatorConfig, loaded.PopulationConfig.EvaluatorConfig) {
			t.Errorf("EvaluatorConfig mismatch.\nOriginal: %+v\nLoaded:   %+v", original.EvaluatorConfig, loaded.PopulationConfig.EvaluatorConfig)
		}
		if !reflect.DeepEqual(original.SelectorConfig, loaded.PopulationConfig.SelectorConfig) {
			t.Errorf("SelectorConfig mismatch.\nOriginal: %+v\nLoaded:   %+v", original.SelectorConfig, loaded.PopulationConfig.SelectorConfig)
		}
		if !reflect.DeepEqual(original.FitnessConfig, loaded.PopulationConfig.FitnessConfig) {
			t.Errorf("FitnessConfig mismatch.\nOriginal: %+v\nLoaded:   %+v", original.FitnessConfig, loaded.PopulationConfig.FitnessConfig)
		}
	}
}
