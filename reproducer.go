package genetic_sort

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
)

// Reproducer implements Phase 3: fitness-proportional reproduction.
// Survivors are ranked by fitness; higher-ranked units produce more offspring.
type Reproducer struct {
	persist      *Persistence
	PopulationID uint
	MaxOffspring uint
	Ranker       *FitnessRanker
	BatchSize    uint
	UnitIDs      *IDGenerator
	InsIDs       *IDGenerator
}

func NewReproducer(persist *Persistence, popID, maxOffspring, batchSize uint, ranker *FitnessRanker,
	unitIDs, insIDs *IDGenerator) *Reproducer {
	return &Reproducer{
		persist:      persist,
		PopulationID: popID,
		MaxOffspring: maxOffspring,
		Ranker:       ranker,
		BatchSize:    batchSize,
		UnitIDs:      unitIDs,
		InsIDs:       insIDs,
	}
}

// Reproduce creates offspring for all alive units. Higher-fitness units
// produce more offspring (up to MaxOffspring). Returns total offspring count.
func (r *Reproducer) Reproduce() (uint, error) {
	maxOffspring := r.MaxOffspring
	if maxOffspring == 0 {
		maxOffspring = 1
	}

	// Get latest evaluations for alive units across all shards
	type shardResult struct {
		evals []Evaluation
		err   error
	}
	evalResults := make([]shardResult, r.persist.NumShards)
	var wg sync.WaitGroup

	for i := uint(0); i < r.persist.NumShards; i++ {
		wg.Add(1)
		go func(shard uint) {
			defer wg.Done()
			db := r.persist.Shards[shard]
			evals, err := queryReproduceEvals(db, r.PopulationID)
			evalResults[shard] = shardResult{evals: evals, err: err}
		}(i)
	}
	wg.Wait()

	var evals []Evaluation
	for _, sr := range evalResults {
		if sr.err != nil {
			return 0, sr.err
		}
		evals = append(evals, sr.evals...)
	}

	if len(evals) == 0 {
		return 0, nil
	}

	// Load all alive units with instructions across all shards
	type unitResult struct {
		units []*Unit
		err   error
	}
	type insResult struct {
		instructions []*Instruction
		err          error
	}
	unitResults := make([]unitResult, r.persist.NumShards)
	insResults := make([]insResult, r.persist.NumShards)

	for i := uint(0); i < r.persist.NumShards; i++ {
		wg.Add(1)
		go func(shard uint) {
			defer wg.Done()
			db := r.persist.Shards[shard]
			units, err := queryUnits(db, r.PopulationID)
			unitResults[shard] = unitResult{units: units, err: err}
		}(i)
		wg.Add(1)
		go func(shard uint) {
			defer wg.Done()
			db := r.persist.Shards[shard]
			instructions, err := queryInstructions(db, r.PopulationID)
			insResults[shard] = insResult{instructions: instructions, err: err}
		}(i)
	}
	wg.Wait()

	var allUnits []*Unit
	for _, ur := range unitResults {
		if ur.err != nil {
			return 0, fmt.Errorf("failed to load units for reproduction: %w", ur.err)
		}
		allUnits = append(allUnits, ur.units...)
	}

	unitMap := make(map[uint]*Unit, len(allUnits))
	for _, u := range allUnits {
		u.Instructions = make([]*Instruction, 0, 10)
		unitMap[u.ID] = u
	}
	for _, ir := range insResults {
		if ir.err != nil {
			return 0, fmt.Errorf("failed to load instructions for reproduction: %w", ir.err)
		}
		for _, ins := range ir.instructions {
			if u, ok := unitMap[ins.UnitID]; ok {
				u.Instructions = append(u.Instructions, ins)
			}
		}
	}

	return r.reproduceFromData(allUnits, evals, maxOffspring)
}

func queryReproduceEvals(db *sql.DB, popID uint) ([]Evaluation, error) {
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
		return nil, fmt.Errorf("failed to query evaluations for reproduction: %w", err)
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

// ReproduceFromUnits creates offspring using units and evaluations already in
// memory — avoids re-querying the database. Units must have Instructions loaded.
func (r *Reproducer) ReproduceFromUnits(units []*Unit) (uint, error) {
	maxOffspring := r.MaxOffspring
	if maxOffspring == 0 {
		maxOffspring = 1
	}

	// Extract alive units and their latest evaluations
	var aliveUnits []*Unit
	var evals []Evaluation
	for _, u := range units {
		if u.Alive != Alive || len(u.Evaluations) == 0 {
			continue
		}
		aliveUnits = append(aliveUnits, u)
		evals = append(evals, *u.Evaluations[len(u.Evaluations)-1])
	}

	return r.reproduceFromData(aliveUnits, evals, maxOffspring)
}

// ReproduceStreaming creates offspring by streaming alive units from the DB
// in batches. First queries eval scores globally to build the offspring count
// map (lightweight — only scores, no full units). Then captures MAX(id) per
// shard to avoid iterating over newly-inserted offspring. Finally streams
// batches of units through Mitosis and persists offspring immediately.
func (r *Reproducer) ReproduceStreaming(evalBatchSize int) (uint, error) {
	maxOffspring := r.MaxOffspring
	if maxOffspring == 0 {
		maxOffspring = 1
	}

	// Step 1: Query eval scores globally to build offspring count map
	type shardResult struct {
		evals []Evaluation
		err   error
	}
	evalResults := make([]shardResult, r.persist.NumShards)
	var wg sync.WaitGroup

	for i := uint(0); i < r.persist.NumShards; i++ {
		wg.Add(1)
		go func(shard uint) {
			defer wg.Done()
			db := r.persist.Shards[shard]
			evals, err := queryReproduceEvals(db, r.PopulationID)
			evalResults[shard] = shardResult{evals: evals, err: err}
		}(i)
	}
	wg.Wait()

	var evals []Evaluation
	for _, sr := range evalResults {
		if sr.err != nil {
			return 0, sr.err
		}
		evals = append(evals, sr.evals...)
	}

	if len(evals) == 0 {
		return 0, nil
	}

	// Sort best-first and build offspring count map
	sort.Slice(evals, func(i, j int) bool {
		return r.Ranker.CompareEvaluations(&evals[i], &evals[j]) < 0
	})

	total := float64(len(evals))
	offspringMap := make(map[uint]uint, len(evals))
	for rank, eval := range evals {
		count := uint(math.Ceil(float64(maxOffspring) * (1.0 - float64(rank)/total)))
		if count < 1 {
			count = 1
		}
		offspringMap[eval.UnitID] = count
	}

	// Step 2: Capture MAX(id) per shard before reproduction starts
	maxIDs := make([]uint, r.persist.NumShards)
	for i := uint(0); i < r.persist.NumShards; i++ {
		wg.Add(1)
		go func(shard uint) {
			defer wg.Done()
			maxID, err := queryMaxUnitID(r.persist.Shards[shard], r.PopulationID)
			if err != nil {
				// Will be caught in ForEachUnitBatch
				maxIDs[shard] = 0
			} else {
				maxIDs[shard] = maxID
			}
		}(i)
	}
	wg.Wait()

	// Step 3: Stream alive units in batches, produce offspring, persist immediately
	var totalOffspring atomic.Uint64
	pop := &Population{ID: r.PopulationID, persist: r.persist}

	err := pop.ForEachUnitBatch(evalBatchSize, maxIDs, func(units []*Unit) error {
		// Parallel Mitosis for this batch
		cpus := runtime.NumCPU()
		chunkSize := len(units) / cpus
		if chunkSize == 0 {
			chunkSize = 1
		}

		type offspringChunk struct {
			units []*Unit
		}
		chunks := make([]offspringChunk, cpus)
		var batchWG sync.WaitGroup

		for i := 0; i < cpus; i++ {
			start := i * chunkSize
			if start >= len(units) {
				break
			}
			end := start + chunkSize
			if i == cpus-1 || end > len(units) {
				end = len(units)
			}
			batchWG.Add(1)
			go func(idx int, chunk []*Unit) {
				defer batchWG.Done()
				var local []*Unit
				for _, unit := range chunk {
					count := offspringMap[unit.ID]
					if count == 0 {
						count = 1
					}
					for n := uint(0); n < count; n++ {
						local = append(local, unit.Mitosis(r.UnitIDs, r.InsIDs))
					}
				}
				chunks[idx].units = local
			}(i, units[start:end])
		}
		batchWG.Wait()

		// Collect offspring for this batch
		var batchOffspring []*Unit
		for _, c := range chunks {
			batchOffspring = append(batchOffspring, c.units...)
		}

		if len(batchOffspring) == 0 {
			return nil
		}

		// Set population ID and compress instructions
		for _, u := range batchOffspring {
			u.PopulationID = r.PopulationID
			for _, ins := range u.Instructions {
				ins.EnsureCompressed()
			}
		}

		// Persist this batch of offspring immediately using bulk inserts
		err := r.persist.writeSharded(batchOffspring, func(tx *sql.Tx, batch []*Unit) error {
			return bulkInsertUnits(tx, batch)
		})
		if err != nil {
			return fmt.Errorf("failed to save offspring batch: %w", err)
		}

		totalOffspring.Add(uint64(len(batchOffspring)))
		return nil
	})
	if err != nil {
		return 0, err
	}

	result := uint(totalOffspring.Load())
	log.Printf("Streaming reproduction: %d offspring from %d survivors (max_offspring: %d)",
		result, len(evals), maxOffspring)

	return result, nil
}

func (r *Reproducer) reproduceFromData(allUnits []*Unit, evals []Evaluation, maxOffspring uint) (uint, error) {
	if len(evals) == 0 {
		return 0, nil
	}

	// Sort best-first
	sort.Slice(evals, func(i, j int) bool {
		return r.Ranker.CompareEvaluations(&evals[i], &evals[j]) < 0
	})

	// Build map of unitID -> offspring count
	total := float64(len(evals))
	offspringMap := make(map[uint]uint, len(evals))
	for rank, eval := range evals {
		count := uint(math.Ceil(float64(maxOffspring) * (1.0 - float64(rank)/total)))
		if count < 1 {
			count = 1
		}
		offspringMap[eval.UnitID] = count
	}

	// Parallel Mitosis: split units across CPUs
	cpus := runtime.NumCPU()
	chunkSize := len(allUnits) / cpus
	if chunkSize == 0 {
		chunkSize = 1
	}
	type offspringChunk struct {
		units []*Unit
	}
	chunks := make([]offspringChunk, cpus)
	var wg sync.WaitGroup

	for i := 0; i < cpus; i++ {
		start := i * chunkSize
		if start >= len(allUnits) {
			break
		}
		end := start + chunkSize
		if i == cpus-1 || end > len(allUnits) {
			end = len(allUnits)
		}
		wg.Add(1)
		go func(idx int, chunk []*Unit) {
			defer wg.Done()
			var local []*Unit
			for _, unit := range chunk {
				count := offspringMap[unit.ID]
				if count == 0 {
					count = 1
				}
				for n := uint(0); n < count; n++ {
					local = append(local, unit.Mitosis(r.UnitIDs, r.InsIDs))
				}
			}
			chunks[idx].units = local
		}(i, allUnits[start:end])
	}
	wg.Wait()

	// Collect all offspring
	var allOffspring []*Unit
	for _, c := range chunks {
		allOffspring = append(allOffspring, c.units...)
	}
	totalOffspring := uint(len(allOffspring))

	// Set population ID on all offspring
	for _, u := range allOffspring {
		u.PopulationID = r.PopulationID
		for _, ins := range u.Instructions {
			ins.EnsureCompressed()
		}
	}

	// Persist using sharded bulk inserts
	if totalOffspring > 0 {
		err := r.persist.writeSharded(allOffspring, func(tx *sql.Tx, batch []*Unit) error {
			return bulkInsertUnits(tx, batch)
		})
		if err != nil {
			return 0, fmt.Errorf("failed to save offspring: %w", err)
		}
	}

	log.Printf("Reproduction: %d offspring from %d survivors (max_offspring: %d)",
		totalOffspring, len(evals), maxOffspring)

	return totalOffspring, nil
}
