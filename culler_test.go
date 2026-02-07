package genetic_sort

import (
	"database/sql"
	test "testing"

	_ "github.com/glebarez/go-sqlite"
	bf "nickandperla.net/brainfuck"
)

func setupCullerTestDB(t *test.T) (*sql.DB, *Persistence) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Failed to open in-memory DB: %v", err)
	}
	if err := createTestSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}
	return db, testPersistence(db)
}

// seedUnitsWithEvals creates n alive units for the given population, each with
// an evaluation. Sortedness values are assigned 0..n-1 so units have distinct
// fitness rankings.
func seedUnitsWithEvals(t *test.T, db *sql.DB, popID uint, n int) []*Unit {
	units := make([]*Unit, n)
	for i := 0; i < n; i++ {
		res, err := db.Exec(`INSERT INTO units (population_id, alive, mutation_chance, lifespan) VALUES (?, ?, ?, ?)`,
			popID, Alive, 0.1, 100)
		if err != nil {
			t.Fatalf("Failed to create unit %d: %v", i, err)
		}
		id, _ := res.LastInsertId()
		u := &Unit{ID: uint(id), PopulationID: popID, Alive: Alive}

		if _, err := db.Exec(`INSERT INTO evaluations (unit_id, machine_run, sortedness, set_fidelity, instructions_executed, instruction_count)
			VALUES (?, ?, ?, ?, ?, ?)`,
			u.ID, 1, byte(i), 50, 100, 0); err != nil {
			t.Fatalf("Failed to create evaluation for unit %d: %v", i, err)
		}
		units[i] = u
	}
	return units
}

func countAlive(t *test.T, db *sql.DB, popID uint) int64 {
	var count int64
	db.QueryRow("SELECT COUNT(*) FROM units WHERE population_id = ? AND alive = ?", popID, Alive).Scan(&count)
	return count
}

func countTombstones(t *test.T, db *sql.DB, reason SelectFailReason) int64 {
	var count int64
	db.QueryRow("SELECT COUNT(*) FROM tombstones WHERE reason = ?", reason).Scan(&count)
	return count
}

func insertTestPopulation(t *test.T, db *sql.DB, unitCount uint) *Population {
	pop := &Population{
		ID:               1,
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

func TestCullNoOpWhenUnderCapacity(t *test.T) {
	db, persist := setupCullerTestDB(t)
	defer db.Close()

	pop := insertTestPopulation(t, db, 10)
	seedUnitsWithEvals(t, db, pop.ID, 5)

	culler := NewCompetitiveCuller(persist, pop.ID, 10, 0, 100, NewFitnessRanker(nil))
	culled, err := culler.Cull()
	if err != nil {
		t.Fatalf("Cull returned error: %v", err)
	}
	if culled != 0 {
		t.Errorf("Expected 0 culled when under capacity, got %d", culled)
	}
	if alive := countAlive(t, db, pop.ID); alive != 5 {
		t.Errorf("Expected 5 alive, got %d", alive)
	}
}

func TestCullKillsExcessByFitness(t *test.T) {
	db, persist := setupCullerTestDB(t)
	defer db.Close()

	pop := insertTestPopulation(t, db, 10)
	units := seedUnitsWithEvals(t, db, pop.ID, 10)

	// Capacity 6 means 4 should die. Worst fitness = lowest sortedness (units[0]..units[3])
	culler := NewCompetitiveCuller(persist, pop.ID, 6, 0, 100, NewFitnessRanker(nil))
	culled, err := culler.Cull()
	if err != nil {
		t.Fatalf("Cull returned error: %v", err)
	}
	if culled != 4 {
		t.Errorf("Expected 4 culled, got %d", culled)
	}
	if alive := countAlive(t, db, pop.ID); alive != 6 {
		t.Errorf("Expected 6 alive after cull, got %d", alive)
	}

	// Verify the dead ones are the worst-fitness units (sortedness 0..3)
	for i := 0; i < 4; i++ {
		var aliveVal uint
		db.QueryRow("SELECT alive FROM units WHERE id = ?", units[i].ID).Scan(&aliveVal)
		if aliveVal != Dead {
			t.Errorf("Unit %d (sortedness %d) should be dead but is alive", units[i].ID, i)
		}
	}
	// Best units should still be alive
	for i := 4; i < 10; i++ {
		var aliveVal uint
		db.QueryRow("SELECT alive FROM units WHERE id = ?", units[i].ID).Scan(&aliveVal)
		if aliveVal != Alive {
			t.Errorf("Unit %d (sortedness %d) should be alive but is dead", units[i].ID, i)
		}
	}

	// Verify tombstones with correct reason
	if tombstones := countTombstones(t, db, FailedCompetition); tombstones != 4 {
		t.Errorf("Expected 4 FailedCompetition tombstones, got %d", tombstones)
	}
}

func TestCullProtectsElites(t *test.T) {
	db, persist := setupCullerTestDB(t)
	defer db.Close()

	pop := insertTestPopulation(t, db, 10)
	seedUnitsWithEvals(t, db, pop.ID, 10)

	// Capacity 3, elitism 3: want to kill 7, but top 3 are protected.
	// That means we can kill units at ranks 3..9 (7 units), all of them.
	culler := NewCompetitiveCuller(persist, pop.ID, 3, 3, 100, NewFitnessRanker(nil))
	culled, err := culler.Cull()
	if err != nil {
		t.Fatalf("Cull returned error: %v", err)
	}
	if culled != 7 {
		t.Errorf("Expected 7 culled, got %d", culled)
	}
	if alive := countAlive(t, db, pop.ID); alive != 3 {
		t.Errorf("Expected 3 alive after cull, got %d", alive)
	}
}

func TestCullElitismLargerThanExcess(t *test.T) {
	db, persist := setupCullerTestDB(t)
	defer db.Close()

	pop := insertTestPopulation(t, db, 10)
	seedUnitsWithEvals(t, db, pop.ID, 10)

	// Capacity 7, elitism 8: want to kill 3, but top 7 are protected (elitism
	// capped to capacity). Units at ranks 7..9 get killed = 3 units.
	culler := NewCompetitiveCuller(persist, pop.ID, 7, 8, 100, NewFitnessRanker(nil))
	culled, err := culler.Cull()
	if err != nil {
		t.Fatalf("Cull returned error: %v", err)
	}
	if culled != 3 {
		t.Errorf("Expected 3 culled, got %d", culled)
	}
	if alive := countAlive(t, db, pop.ID); alive != 7 {
		t.Errorf("Expected 7 alive, got %d", alive)
	}
}
