package genetic_sort

import (
	"database/sql"
	"fmt"
	"sync"
)

// PopulationMetrics holds aggregate fitness metrics for a population.
type PopulationMetrics struct {
	AliveCount     uint
	BestSortedness byte
	BestSetFidelity byte
	AvgSortedness  float64
	AvgSetFidelity float64
}

// shardMetrics holds per-shard aggregates that get merged into PopulationMetrics.
type shardMetrics struct {
	count          uint
	sumSortedness  uint64
	sumSetFidelity uint64
	maxSortedness  byte
	maxSetFidelity byte
}

// QueryMetrics queries aggregate fitness metrics for all alive units across
// all shards in parallel. Uses the same latest-evaluation join pattern as
// the competitive culler.
func (p *Population) QueryMetrics() (*PopulationMetrics, error) {
	results := make([]shardMetrics, p.persist.NumShards)
	errs := make([]error, p.persist.NumShards)
	var wg sync.WaitGroup

	for i := uint(0); i < p.persist.NumShards; i++ {
		wg.Add(1)
		go func(shard uint) {
			defer wg.Done()
			sm, err := queryShardMetrics(p.persist.Shards[shard], p.ID)
			if err != nil {
				errs[shard] = fmt.Errorf("shard %d: %w", shard, err)
				return
			}
			results[shard] = sm
		}(i)
	}
	wg.Wait()

	if err := firstError(errs); err != nil {
		return nil, err
	}

	// Merge shard results
	m := &PopulationMetrics{}
	var totalCount uint64
	var totalSortedness, totalFidelity uint64

	for _, sm := range results {
		m.AliveCount += sm.count
		totalCount += uint64(sm.count)
		totalSortedness += sm.sumSortedness
		totalFidelity += sm.sumSetFidelity
		if sm.maxSortedness > m.BestSortedness {
			m.BestSortedness = sm.maxSortedness
		}
		if sm.maxSetFidelity > m.BestSetFidelity {
			m.BestSetFidelity = sm.maxSetFidelity
		}
	}

	if totalCount > 0 {
		m.AvgSortedness = float64(totalSortedness) / float64(totalCount)
		m.AvgSetFidelity = float64(totalFidelity) / float64(totalCount)
	}

	return m, nil
}

func queryShardMetrics(db *sql.DB, popID uint) (shardMetrics, error) {
	var sm shardMetrics
	row := db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(e.sortedness), 0),
		COALESCE(SUM(e.set_fidelity), 0), COALESCE(MAX(e.sortedness), 0),
		COALESCE(MAX(e.set_fidelity), 0)
		FROM evaluations e
		JOIN (
			SELECT MAX(evaluations.id) as id
			FROM evaluations
			JOIN units ON units.id = evaluations.unit_id
			WHERE units.population_id = ? AND units.alive = ?
			GROUP BY evaluations.unit_id
		) latest ON e.id = latest.id`, popID, Alive)

	var count int64
	var sumSort, sumFid int64
	var maxSort, maxFid int
	if err := row.Scan(&count, &sumSort, &sumFid, &maxSort, &maxFid); err != nil {
		return sm, err
	}
	sm.count = uint(count)
	sm.sumSortedness = uint64(sumSort)
	sm.sumSetFidelity = uint64(sumFid)
	sm.maxSortedness = byte(maxSort)
	sm.maxSetFidelity = byte(maxFid)
	return sm, nil
}

// QueryBestUnit finds the best alive unit by sortedness+fidelity across all
// shards. Returns the unit with its instructions loaded and decompressed,
// along with the evaluation. Returns nil, nil if no alive units exist.
func (p *Population) QueryBestUnit() (*Unit, *Evaluation, error) {
	type shardBest struct {
		unitID     uint
		sortedness byte
		fidelity   byte
		evalID     uint
		shard      uint
		found      bool
	}

	results := make([]shardBest, p.persist.NumShards)
	errs := make([]error, p.persist.NumShards)
	var wg sync.WaitGroup

	for i := uint(0); i < p.persist.NumShards; i++ {
		wg.Add(1)
		go func(shard uint) {
			defer wg.Done()
			sb, err := queryShardBestUnit(p.persist.Shards[shard], p.ID, shard)
			if err != nil {
				errs[shard] = fmt.Errorf("shard %d: %w", shard, err)
				return
			}
			results[shard] = sb
		}(i)
	}
	wg.Wait()

	if err := firstError(errs); err != nil {
		return nil, nil, err
	}

	// Find global best
	var best *shardBest
	for i := range results {
		if !results[i].found {
			continue
		}
		if best == nil || (uint(results[i].sortedness)+uint(results[i].fidelity)) >
			(uint(best.sortedness)+uint(best.fidelity)) {
			best = &results[i]
		}
	}

	if best == nil {
		return nil, nil, nil
	}

	// Load the full unit + instructions from the correct shard
	db := p.persist.Shards[best.shard]
	unit, err := loadSingleUnit(db, best.unitID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load best unit: %w", err)
	}

	instructions, err := queryInstructionsForUnits(db, []uint{best.unitID})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load instructions for best unit: %w", err)
	}
	unit.Instructions = instructions
	for _, ins := range unit.Instructions {
		ins.EnsureDecompressed()
	}

	// Load the evaluation
	eval, err := loadSingleEvaluation(db, best.evalID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load evaluation for best unit: %w", err)
	}

	return unit, eval, nil
}

func queryShardBestUnit(db *sql.DB, popID uint, shard uint) (struct {
	unitID     uint
	sortedness byte
	fidelity   byte
	evalID     uint
	shard      uint
	found      bool
}, error) {
	type result = struct {
		unitID     uint
		sortedness byte
		fidelity   byte
		evalID     uint
		shard      uint
		found      bool
	}

	row := db.QueryRow(`SELECT e.id, e.unit_id, e.sortedness, e.set_fidelity
		FROM evaluations e
		JOIN (
			SELECT MAX(evaluations.id) as id
			FROM evaluations
			JOIN units ON units.id = evaluations.unit_id
			WHERE units.population_id = ? AND units.alive = ?
			GROUP BY evaluations.unit_id
		) latest ON e.id = latest.id
		ORDER BY (e.sortedness + e.set_fidelity) DESC
		LIMIT 1`, popID, Alive)

	var evalID, unitID uint
	var sortedness, fidelity int
	if err := row.Scan(&evalID, &unitID, &sortedness, &fidelity); err != nil {
		if err == sql.ErrNoRows {
			return result{}, nil
		}
		return result{}, err
	}
	return result{
		unitID:     unitID,
		sortedness: byte(sortedness),
		fidelity:   byte(fidelity),
		evalID:     evalID,
		shard:      shard,
		found:      true,
	}, nil
}

func loadSingleUnit(db *sql.DB, unitID uint) (*Unit, error) {
	u := &Unit{}
	var parentID sql.NullInt64
	err := db.QueryRow(`SELECT id, population_id, parent_id, age, generation, lifespan, mutation_chance, alive
		FROM units WHERE id = ?`, unitID).
		Scan(&u.ID, &u.PopulationID, &parentID, &u.Age, &u.Generation, &u.Lifespan, &u.MutationChance, &u.Alive)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		pid := uint(parentID.Int64)
		u.ParentID = &pid
	}
	return u, nil
}

func loadSingleEvaluation(db *sql.DB, evalID uint) (*Evaluation, error) {
	e := &Evaluation{}
	var machineRun int
	err := db.QueryRow(`SELECT id, unit_id, machine_run, set_fidelity, sortedness,
		instruction_count, instructions_executed, machine_error
		FROM evaluations WHERE id = ?`, evalID).
		Scan(&e.ID, &e.UnitID, &machineRun, &e.SetFidelity, &e.Sortedness,
			&e.InstructionCount, &e.InstructionsExecuted, &e.MachineError)
	if err != nil {
		return nil, err
	}
	e.MachineRun = machineRun != 0
	return e, nil
}
