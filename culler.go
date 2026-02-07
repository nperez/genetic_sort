package genetic_sort

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"sync"
)

// CompetitiveCuller implements Phase 2: rank alive units by fitness, protect
// elites, and kill units beyond the carrying capacity.
type CompetitiveCuller struct {
	persist          *Persistence
	PopulationID     uint
	CarryingCapacity uint
	Elitism          uint
	Ranker           *FitnessRanker
	BatchSize        uint
}

func NewCompetitiveCuller(persist *Persistence, popID, carryingCapacity, elitism, batchSize uint, ranker *FitnessRanker) *CompetitiveCuller {
	return &CompetitiveCuller{
		persist:          persist,
		PopulationID:     popID,
		CarryingCapacity: carryingCapacity,
		Elitism:          elitism,
		Ranker:           ranker,
		BatchSize:        batchSize,
	}
}

// Cull ranks all alive units by fitness and kills excess beyond carrying
// capacity. Returns the number of units culled.
func (cc *CompetitiveCuller) Cull() (uint, error) {
	// Query latest evaluation for each alive unit across all shards
	type shardResult struct {
		evals []Evaluation
		err   error
	}
	results := make([]shardResult, cc.persist.NumShards)
	var wg sync.WaitGroup

	for i := uint(0); i < cc.persist.NumShards; i++ {
		wg.Add(1)
		go func(shard uint) {
			defer wg.Done()
			db := cc.persist.Shards[shard]
			evals, err := queryCullEvals(db, cc.PopulationID)
			results[shard] = shardResult{evals: evals, err: err}
		}(i)
	}
	wg.Wait()

	var allEvals []Evaluation
	for _, r := range results {
		if r.err != nil {
			return 0, r.err
		}
		allEvals = append(allEvals, r.evals...)
	}

	culled, _, err := cc.cullFromEvals(allEvals)
	return culled, err
}

func queryCullEvals(db *sql.DB, popID uint) ([]Evaluation, error) {
	rows, err := db.Query(`SELECT e.unit_id, e.machine_run, e.set_fidelity, e.sortedness,
		e.instruction_count, e.instructions_executed
		FROM evaluations e
		JOIN (
			SELECT MAX(evaluations.id) as id
			FROM evaluations
			JOIN units ON units.id = evaluations.unit_id
			WHERE units.population_id = ? AND units.alive = ?
			GROUP BY evaluations.unit_id
		) latest ON e.id = latest.id`, popID, Alive)
	if err != nil {
		return nil, fmt.Errorf("failed to query evaluations for culling: %w", err)
	}
	defer rows.Close()

	var evals []Evaluation
	for rows.Next() {
		var e Evaluation
		var machineRun int
		if err := rows.Scan(&e.UnitID, &machineRun, &e.SetFidelity, &e.Sortedness,
			&e.InstructionCount, &e.InstructionsExecuted); err != nil {
			return nil, err
		}
		e.MachineRun = machineRun != 0
		evals = append(evals, e)
	}
	return evals, rows.Err()
}

// CullFromUnits ranks alive units using their in-memory evaluations and kills
// excess beyond carrying capacity. Marks killed units dead in-memory too so
// downstream consumers see the correct alive set. Returns culled count.
func (cc *CompetitiveCuller) CullFromUnits(units []*Unit) (uint, error) {
	// Extract the latest evaluation for each alive unit
	var evals []Evaluation
	for _, u := range units {
		if u.Alive != Alive || len(u.Evaluations) == 0 {
			continue
		}
		evals = append(evals, *u.Evaluations[len(u.Evaluations)-1])
	}
	culled, killIDs, err := cc.cullFromEvals(evals)
	if err != nil {
		return 0, err
	}
	// Mark killed units dead in memory
	if len(killIDs) > 0 {
		killSet := make(map[uint]bool, len(killIDs))
		for _, id := range killIDs {
			killSet[id] = true
		}
		for _, u := range units {
			if killSet[u.ID] {
				u.Alive = Dead
			}
		}
	}
	return culled, nil
}

func (cc *CompetitiveCuller) cullFromEvals(evals []Evaluation) (uint, []uint, error) {
	aliveCount := uint(len(evals))
	if aliveCount <= cc.CarryingCapacity {
		return 0, nil, nil
	}

	// Sort by fitness: best first (CompareEvaluations returns -1 if a is better)
	sort.Slice(evals, func(i, j int) bool {
		return cc.Ranker.CompareEvaluations(&evals[i], &evals[j]) < 0
	})

	// Units beyond carrying capacity get culled (but protect elites)
	toKill := aliveCount - cc.CarryingCapacity
	protect := cc.Elitism
	if protect > cc.CarryingCapacity {
		protect = cc.CarryingCapacity
	}

	var killIDs []uint
	// Walk from worst (end) to best, skipping the protected elites at the top
	for i := len(evals) - 1; i >= 0 && uint(len(killIDs)) < toKill; i-- {
		if uint(i) < protect {
			break // these are elite, don't kill
		}
		killIDs = append(killIDs, evals[i].UnitID)
	}

	if len(killIDs) == 0 {
		return 0, nil, nil
	}

	culled := uint(len(killIDs))
	log.Printf("Competitive cull: killing %d units (alive: %d, capacity: %d, elites: %d)",
		culled, aliveCount, cc.CarryingCapacity, cc.Elitism)

	// Batch update units to dead and create tombstones, sharded by unit ID
	err := cc.persist.writeShardedByID(killIDs, func(tx *sql.Tx, ids []uint) error {
		for _, uid := range ids {
			if _, err := tx.Exec("UPDATE units SET alive = ? WHERE id = ?", Dead, uid); err != nil {
				return fmt.Errorf("failed to mark unit dead during cull: %w", err)
			}
			if _, err := tx.Exec("INSERT INTO tombstones (id, unit_id, reason) VALUES (?, ?, ?)",
				cc.persist.TombstoneIDs.Next(), uid, FailedCompetition); err != nil {
				return fmt.Errorf("failed to create tombstone during cull: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return 0, nil, err
	}

	return culled, killIDs, nil
}
