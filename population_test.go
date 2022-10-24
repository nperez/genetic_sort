package genetic_sort

import (
	"fmt"
	t "testing"

	sqlite "github.com/glebarez/sqlite"
	gorm "gorm.io/gorm"
)

const (
	TEST_DB = "test.db"
	PRAGMAS = "_pragma=journal_mode=WAL&_pragma=journal_size_limit=4000000"
	OPTIONS = "cache=shared"
)

func TestPersist(t *t.T) {
	filename := fmt.Sprintf("%s?%s&%s", TEST_DB, PRAGMAS, OPTIONS)
	db, err := gorm.Open(sqlite.Open(filename), &gorm.Config{})

	if err != nil {
		t.Fatalf("Failed to open %s: %v", filename, err)
	}

	if err := db.AutoMigrate(&Population{}, &Unit{}, &Instruction{}, &Mutation{}); err != nil {
		t.Fatalf("Failed to AutoMigrate: %v", err)
	}

	pop1 := NewPopulationFromConfig(&PopulationConfig{UnitCount: 100, UnitConfig: &UnitConfig{MutationChance: 0.25, InstructionCount: 10, InstructionConfig: &InstructionConfig{OpSetCount: 10}, Lifespan: 100}, SelectorConfig: &SelectorConfig{}})
	db = db.Session(&gorm.Session{CreateBatchSize: 100})

	db.Create(pop1)

	if sqldb, err := db.DB(); err != nil {
		t.Fatalf("Failed to retrieve raw DB: %v", err)
	} else {
		sqldb.Close()
	}
}
