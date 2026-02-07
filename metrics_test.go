package genetic_sort

import (
	"database/sql"
	test "testing"

	_ "github.com/glebarez/go-sqlite"
	bf "nickandperla.net/brainfuck"
)

func setupMetricsTestDB(t *test.T) (*sql.DB, *Persistence) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("Failed to open in-memory DB: %v", err)
	}
	if err := createTestSchema(db); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}
	return db, testPersistence(db)
}

func metricsTestPopulation(t *test.T, db *sql.DB) *Population {
	pop := &Population{
		ID: 1,
		PopulationConfig: &PopulationConfig{
			UnitCount:       10,
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

func TestQueryMetricsEmpty(t *test.T) {
	db, persist := setupMetricsTestDB(t)
	defer db.Close()

	pop := metricsTestPopulation(t, db)
	pop.persist = persist

	m, err := pop.QueryMetrics()
	if err != nil {
		t.Fatalf("QueryMetrics returned error: %v", err)
	}
	if m.AliveCount != 0 {
		t.Errorf("Expected 0 alive, got %d", m.AliveCount)
	}
	if m.BestSortedness != 0 || m.BestSetFidelity != 0 {
		t.Errorf("Expected zero best metrics, got sortedness=%d fidelity=%d",
			m.BestSortedness, m.BestSetFidelity)
	}
}

func TestQueryMetricsWithUnits(t *test.T) {
	db, persist := setupMetricsTestDB(t)
	defer db.Close()

	pop := metricsTestPopulation(t, db)
	pop.persist = persist

	// Create 3 alive units with evaluations
	for i := 0; i < 3; i++ {
		res, err := db.Exec(`INSERT INTO units (population_id, alive, mutation_chance, lifespan) VALUES (?, ?, ?, ?)`,
			pop.ID, Alive, 0.1, 100)
		if err != nil {
			t.Fatalf("Failed to create unit %d: %v", i, err)
		}
		id, _ := res.LastInsertId()

		sortedness := byte(50 + i*10) // 50, 60, 70
		fidelity := byte(80 + i*5)    // 80, 85, 90

		if _, err := db.Exec(`INSERT INTO evaluations (unit_id, machine_run, sortedness, set_fidelity, instructions_executed, instruction_count)
			VALUES (?, ?, ?, ?, ?, ?)`,
			id, 1, sortedness, fidelity, 100, 50); err != nil {
			t.Fatalf("Failed to create evaluation for unit %d: %v", i, err)
		}
	}

	m, err := pop.QueryMetrics()
	if err != nil {
		t.Fatalf("QueryMetrics returned error: %v", err)
	}
	if m.AliveCount != 3 {
		t.Errorf("Expected 3 alive, got %d", m.AliveCount)
	}
	if m.BestSortedness != 70 {
		t.Errorf("Expected best sortedness 70, got %d", m.BestSortedness)
	}
	if m.BestSetFidelity != 90 {
		t.Errorf("Expected best set fidelity 90, got %d", m.BestSetFidelity)
	}

	// avg sortedness = (50+60+70)/3 = 60.0
	if m.AvgSortedness < 59.9 || m.AvgSortedness > 60.1 {
		t.Errorf("Expected avg sortedness ~60.0, got %.2f", m.AvgSortedness)
	}
	// avg fidelity = (80+85+90)/3 = 85.0
	if m.AvgSetFidelity < 84.9 || m.AvgSetFidelity > 85.1 {
		t.Errorf("Expected avg set fidelity ~85.0, got %.2f", m.AvgSetFidelity)
	}
}

func TestQueryMetricsUsesLatestEval(t *test.T) {
	db, persist := setupMetricsTestDB(t)
	defer db.Close()

	pop := metricsTestPopulation(t, db)
	pop.persist = persist

	// Create one unit with two evaluations; the latest should be used
	res, err := db.Exec(`INSERT INTO units (population_id, alive, mutation_chance, lifespan) VALUES (?, ?, ?, ?)`,
		pop.ID, Alive, 0.1, 100)
	if err != nil {
		t.Fatalf("Failed to create unit: %v", err)
	}
	id, _ := res.LastInsertId()

	// Older eval — lower scores
	if _, err := db.Exec(`INSERT INTO evaluations (unit_id, machine_run, sortedness, set_fidelity, instructions_executed, instruction_count)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, 1, 20, 30, 100, 50); err != nil {
		t.Fatalf("Failed to create old evaluation: %v", err)
	}
	// Newer eval — higher scores
	if _, err := db.Exec(`INSERT INTO evaluations (unit_id, machine_run, sortedness, set_fidelity, instructions_executed, instruction_count)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, 1, 80, 90, 100, 50); err != nil {
		t.Fatalf("Failed to create new evaluation: %v", err)
	}

	m, err := pop.QueryMetrics()
	if err != nil {
		t.Fatalf("QueryMetrics returned error: %v", err)
	}
	if m.BestSortedness != 80 {
		t.Errorf("Expected best sortedness 80 (latest eval), got %d", m.BestSortedness)
	}
	if m.BestSetFidelity != 90 {
		t.Errorf("Expected best set fidelity 90 (latest eval), got %d", m.BestSetFidelity)
	}
}

func TestQueryBestUnitEmpty(t *test.T) {
	db, persist := setupMetricsTestDB(t)
	defer db.Close()

	pop := metricsTestPopulation(t, db)
	pop.persist = persist

	unit, eval, err := pop.QueryBestUnit()
	if err != nil {
		t.Fatalf("QueryBestUnit returned error: %v", err)
	}
	if unit != nil || eval != nil {
		t.Errorf("Expected nil unit and eval for empty population")
	}
}

func TestQueryBestUnit(t *test.T) {
	db, persist := setupMetricsTestDB(t)
	defer db.Close()

	pop := metricsTestPopulation(t, db)
	pop.persist = persist

	// Create 3 units: sortedness+fidelity scores of 100, 150, 120
	type unitScore struct {
		sortedness byte
		fidelity   byte
	}
	scores := []unitScore{{50, 50}, {80, 70}, {60, 60}}

	for i, sc := range scores {
		res, err := db.Exec(`INSERT INTO units (population_id, alive, mutation_chance, lifespan) VALUES (?, ?, ?, ?)`,
			pop.ID, Alive, 0.1, 100)
		if err != nil {
			t.Fatalf("Failed to create unit %d: %v", i, err)
		}
		id, _ := res.LastInsertId()

		// Insert an instruction for the unit
		if _, err := db.Exec(`INSERT INTO instructions (unit_id, age, initial_op_set, ops) VALUES (?, ?, ?, ?)`,
			id, 0, []byte{0x12, 0x34, 0x00, 0x00}, []byte{0x12, 0x34, 0x00, 0x00}); err != nil {
			t.Fatalf("Failed to create instruction for unit %d: %v", i, err)
		}

		if _, err := db.Exec(`INSERT INTO evaluations (unit_id, machine_run, sortedness, set_fidelity, instructions_executed, instruction_count)
			VALUES (?, ?, ?, ?, ?, ?)`,
			id, 1, sc.sortedness, sc.fidelity, 100, 50); err != nil {
			t.Fatalf("Failed to create evaluation for unit %d: %v", i, err)
		}
	}

	unit, eval, err := pop.QueryBestUnit()
	if err != nil {
		t.Fatalf("QueryBestUnit returned error: %v", err)
	}
	if unit == nil {
		t.Fatal("Expected non-nil unit")
	}
	if eval == nil {
		t.Fatal("Expected non-nil eval")
	}

	// Best should be unit with sortedness=80, fidelity=70 (sum=150)
	if eval.Sortedness != 80 || eval.SetFidelity != 70 {
		t.Errorf("Expected best eval sortedness=80 fidelity=70, got sortedness=%d fidelity=%d",
			eval.Sortedness, eval.SetFidelity)
	}

	// Verify instructions were loaded
	if len(unit.Instructions) == 0 {
		t.Error("Expected instructions to be loaded for best unit")
	}
}
