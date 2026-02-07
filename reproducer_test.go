package genetic_sort

import (
	"database/sql"
	test "testing"

	_ "github.com/glebarez/go-sqlite"
	bf "nickandperla.net/brainfuck"
)

func setupReproducerTestDB(t *test.T) (*sql.DB, *Persistence) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Failed to open in-memory DB: %v", err)
	}
	if err := createTestSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}
	return db, testPersistence(db)
}

// seedUnitsForReproduction creates n alive units with instructions and
// evaluations so they can undergo Mitosis.
func seedUnitsForReproduction(t *test.T, db *sql.DB, popID uint, n int, unitIDs, insIDs *IDGenerator) []*Unit {
	rng = newPooledRand(99)
	units := make([]*Unit, n)
	for i := 0; i < n; i++ {
		u := NewUnitFromConfig(&UnitConfig{
			MutationChance:    0.1,
			InstructionCount:  2,
			InstructionConfig: &InstructionConfig{OpSetCount: 2},
			Lifespan:          100,
		})
		u.PopulationID = popID
		u.ID = unitIDs.Next()
		// Insert unit
		if _, err := db.Exec(`INSERT INTO units (id, population_id, alive, mutation_chance, lifespan) VALUES (?, ?, ?, ?, ?)`,
			u.ID, u.PopulationID, Alive, u.MutationChance, u.Lifespan); err != nil {
			t.Fatalf("Failed to create unit %d: %v", i, err)
		}
		// Insert instructions
		for _, ins := range u.Instructions {
			ins.ID = insIDs.Next()
			ins.UnitID = u.ID
			ins.EnsureCompressed()
			if _, err := db.Exec(`INSERT INTO instructions (id, unit_id, age, initial_op_set, ops) VALUES (?, ?, ?, ?, ?)`,
				ins.ID, ins.UnitID, ins.Age, ins.InitialOpSet, ins.Ops); err != nil {
				t.Fatalf("Failed to create instruction for unit %d: %v", i, err)
			}
		}
		// Insert evaluation
		if _, err := db.Exec(`INSERT INTO evaluations (unit_id, machine_run, sortedness, set_fidelity, instructions_executed, instruction_count)
			VALUES (?, ?, ?, ?, ?, ?)`,
			u.ID, 1, byte(i*10), 50, 100, 0); err != nil {
			t.Fatalf("Failed to create evaluation for unit %d: %v", i, err)
		}
		units[i] = u
	}
	return units
}

func insertReproducerTestPopulation(t *test.T, db *sql.DB, unitCount uint) *Population {
	pop := &Population{
		ID: 1,
		PopulationConfig: &PopulationConfig{
			UnitCount:       unitCount,
			UnitConfig:      &UnitConfig{InstructionConfig: &InstructionConfig{}},
			EvaluatorConfig: &EvaluatorConfig{MachineConfig: &bf.MachineConfig{}},
			SelectorConfig:  &SelectorConfig{},
			FitnessConfig:   &FitnessConfig{},
		},
	}
	if err := insertPopulation(db, pop); err != nil {
		t.Fatalf("Failed to insert population: %v", err)
	}
	return pop
}

func TestReproduceMaxOffspringOne(t *test.T) {
	db, persist := setupReproducerTestDB(t)
	defer db.Close()
	rng = newPooledRand(42)

	pop := insertReproducerTestPopulation(t, db, 5)
	seedUnitsForReproduction(t, db, pop.ID, 5, persist.UnitIDs, persist.InstructionIDs)

	reproducer := NewReproducer(persist, pop.ID, 1, 100, NewFitnessRanker(nil), persist.UnitIDs, persist.InstructionIDs)
	offspring, err := reproducer.Reproduce()
	if err != nil {
		t.Fatalf("Reproduce returned error: %v", err)
	}

	// max_offspring=1 → ceil(1 * (1 - rank/total)) for each rank
	// All get 1 offspring each = 5 total
	if offspring != 5 {
		t.Errorf("Expected 5 offspring with max_offspring=1, got %d", offspring)
	}

	// Total alive should be 5 parents + 5 offspring = 10
	alive := countAlive(t, db, pop.ID)
	if alive != 10 {
		t.Errorf("Expected 10 alive (5 parents + 5 offspring), got %d", alive)
	}
}

func TestReproduceFitnessProportional(t *test.T) {
	db, persist := setupReproducerTestDB(t)
	defer db.Close()
	rng = newPooledRand(42)

	pop := insertReproducerTestPopulation(t, db, 5)
	seedUnitsForReproduction(t, db, pop.ID, 5, persist.UnitIDs, persist.InstructionIDs)

	reproducer := NewReproducer(persist, pop.ID, 5, 100, NewFitnessRanker(nil), persist.UnitIDs, persist.InstructionIDs)
	offspring, err := reproducer.Reproduce()
	if err != nil {
		t.Fatalf("Reproduce returned error: %v", err)
	}

	// max_offspring=5, 5 survivors:
	// rank 0 (best):  ceil(5 * (1 - 0/5)) = ceil(5.0) = 5
	// rank 1:         ceil(5 * (1 - 1/5)) = ceil(4.0) = 4
	// rank 2:         ceil(5 * (1 - 2/5)) = ceil(3.0) = 3
	// rank 3:         ceil(5 * (1 - 3/5)) = ceil(2.0) = 2
	// rank 4 (worst): ceil(5 * (1 - 4/5)) = ceil(1.0) = 1
	// Total: 5+4+3+2+1 = 15
	if offspring != 15 {
		t.Errorf("Expected 15 offspring with max_offspring=5, got %d", offspring)
	}

	alive := countAlive(t, db, pop.ID)
	if alive != 20 {
		t.Errorf("Expected 20 alive (5 parents + 15 offspring), got %d", alive)
	}
}

func TestReproduceNoSurvivors(t *test.T) {
	db, persist := setupReproducerTestDB(t)
	defer db.Close()

	pop := insertReproducerTestPopulation(t, db, 5)
	// Don't create any units

	reproducer := NewReproducer(persist, pop.ID, 3, 100, NewFitnessRanker(nil), persist.UnitIDs, persist.InstructionIDs)
	offspring, err := reproducer.Reproduce()
	if err != nil {
		t.Fatalf("Reproduce returned error: %v", err)
	}
	if offspring != 0 {
		t.Errorf("Expected 0 offspring with no survivors, got %d", offspring)
	}
}

func TestReproduceOffspringHaveParent(t *test.T) {
	db, persist := setupReproducerTestDB(t)
	defer db.Close()
	rng = newPooledRand(42)

	pop := insertReproducerTestPopulation(t, db, 1)
	parents := seedUnitsForReproduction(t, db, pop.ID, 1, persist.UnitIDs, persist.InstructionIDs)

	reproducer := NewReproducer(persist, pop.ID, 1, 100, NewFitnessRanker(nil), persist.UnitIDs, persist.InstructionIDs)
	_, err := reproducer.Reproduce()
	if err != nil {
		t.Fatalf("Reproduce returned error: %v", err)
	}

	// Find the offspring (the unit that has parent_id = parent's ID)
	var childID uint
	var parentID sql.NullInt64
	var generation uint
	err = db.QueryRow("SELECT id, parent_id, generation FROM units WHERE population_id = ? AND parent_id = ?",
		pop.ID, parents[0].ID).Scan(&childID, &parentID, &generation)
	if err != nil {
		t.Fatalf("No offspring found with parent_id = %d: %v", parents[0].ID, err)
	}
	if !parentID.Valid || uint(parentID.Int64) != parents[0].ID {
		t.Errorf("Offspring parent_id should be %d, got %v", parents[0].ID, parentID)
	}
	if generation != parents[0].Generation+1 {
		t.Errorf("Offspring generation should be %d, got %d", parents[0].Generation+1, generation)
	}
}
