package genetic_sort

import (
	"database/sql"
	"fmt"
	"log"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Population struct {
	ID                uint
	CurrentGeneration uint
	Units             []*Unit
	PopulationConfig  *PopulationConfig
	persist           *Persistence
}

type PopulationConfig struct {
	UnitCount        uint             `toml:"unit_count"`
	SynthesisPool    uint             `toml:"synthesis_pool"`
	CarryingCapacity uint             `toml:"carrying_capacity"`
	Elitism          uint             `toml:"elitism"`
	MaxOffspring     uint             `toml:"max_offspring"`
	UnitConfig       *UnitConfig      `toml:"unit"`
	EvaluatorConfig  *EvaluatorConfig `toml:"eval"`
	SelectorConfig   *SelectorConfig  `toml:"select"`
	FitnessConfig    *FitnessConfig   `toml:"fitness"`
}

func NewPopulationFromConfig(config *PopulationConfig) *Population {
	return &Population{
		PopulationConfig: config,
	}
}

type rankedUnit struct {
	unit    *Unit
	fitness uint
}

func (p *Population) SynthesizeUnits() error {
	return p.SynthesizeUnitsWithTimeout(2 * time.Minute)
}

func (p *Population) SynthesizeUnitsWithTimeout(timeout time.Duration) error {
	keep := p.PopulationConfig.UnitCount

	synthInput := p.PopulationConfig.EvaluatorConfig.SynthesisInputCellCount
	if synthInput == 0 {
		synthInput = p.PopulationConfig.EvaluatorConfig.InputCellCount
	}

	rounds := p.PopulationConfig.EvaluatorConfig.EvalRounds
	cpus := uint(runtime.NumCPU())

	log.Printf("Synthesizing %d viable units across %d cores (eval_rounds=%d, input_cells=%d, timeout=%v)",
		keep, cpus, rounds, synthInput, timeout)

	deadline := time.Now().Add(timeout)
	var found atomic.Uint64
	var timedOut atomic.Bool
	var mu sync.Mutex
	var candidates []rankedUnit
	var wg sync.WaitGroup

	for i := uint(0); i < cpus; i++ {
		wg.Add(1)
		go func(id uint) {
			defer wg.Done()
			evaluator := NewEvaluator(p.PopulationConfig.EvaluatorConfig)
			selector := NewSelector(p.PopulationConfig.SelectorConfig)
			var local []rankedUnit
			start := time.Now()
			var tested uint

			for found.Load() < uint64(keep) && !timedOut.Load() {
				tested++
				unit := NewUnitFromConfig(p.PopulationConfig.UnitConfig)

				var eval *Evaluation
				if rounds > 1 {
					eval = evaluator.EvaluateMultiRoundWithCellCounts(unit, rounds, synthInput, synthInput)
				} else {
					eval = evaluator.EvaluateWithCellCounts(unit, synthInput, synthInput)
				}

				if tested%10000 == 0 {
					elapsed := time.Since(start)
					log.Printf("Synthesizer %d: %d tested, %d viable (global: %d/%d), %d/sec",
						id, tested, len(local), found.Load(), keep,
						tested/uint(elapsed.Seconds()+1))
					if time.Now().After(deadline) {
						timedOut.Store(true)
						break
					}
				}

				if reason := selector.Select(unit, eval, 0); reason != 0 {
					continue
				}

				local = append(local, rankedUnit{unit: unit, fitness: eval.Fitness()})
				found.Add(1)
			}

			log.Printf("Synthesizer %d done: %d viable out of %d tested", id, len(local), tested)

			mu.Lock()
			candidates = append(candidates, local...)
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	if found.Load() == 0 {
		return fmt.Errorf("synthesis failed: 0 viable units found in %v", timeout)
	}
	if timedOut.Load() {
		log.Printf("Synthesis timeout: found %d/%d viable units", found.Load(), keep)
	}

	// Sort by fitness descending and trim to exact target
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].fitness > candidates[j].fitness
	})

	if uint(len(candidates)) > keep {
		candidates = candidates[:keep]
	}

	if len(candidates) == 0 {
		return fmt.Errorf("synthesis failed: no viable candidates found")
	}

	log.Printf("Keeping %d units (best fitness: %d, worst kept: %d)",
		len(candidates), candidates[0].fitness, candidates[len(candidates)-1].fitness)

	// Assign IDs and persist in batches
	batchSize := p.persist.Config.BatchSize
	batch := make([]*Unit, 0, batchSize)
	for _, ru := range candidates {
		u := ru.unit
		u.PopulationID = p.ID
		// Clear the synthesis evaluation — units will be re-evaluated
		// at full input size during generational processing.
		u.Evaluations = nil
		batch = append(batch, u)
		if uint(len(batch)) == batchSize {
			if err := p.persist.SaveUnits(batch); err != nil {
				panic(fmt.Errorf("Saving new Units failed! %w", err))
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		if err := p.persist.SaveUnits(batch); err != nil {
			panic(fmt.Errorf("Saving new Units failed! %w", err))
		}
	}

	return nil
}

func (p *Population) GetAliveCount() (count uint) {
	if p.persist.NumShards == 1 {
		var dbcnt int64
		if err := p.persist.shard0().QueryRow("SELECT COUNT(*) FROM units WHERE alive = 1 AND population_id = ?", p.ID).Scan(&dbcnt); err != nil {
			log.Printf("Warning: failed to count alive units: %v", err)
		}
		return uint(dbcnt)
	}

	counts := make([]int64, p.persist.NumShards)
	var wg sync.WaitGroup
	for i := uint(0); i < p.persist.NumShards; i++ {
		wg.Add(1)
		go func(shard uint) {
			defer wg.Done()
			var dbcnt int64
			if err := p.persist.Shards[shard].QueryRow("SELECT COUNT(*) FROM units WHERE alive = 1 AND population_id = ?", p.ID).Scan(&dbcnt); err != nil {
				log.Printf("Warning: failed to count alive units on shard %d: %v", shard, err)
			}
			counts[shard] = dbcnt
		}(i)
	}
	wg.Wait()

	var total int64
	for _, c := range counts {
		total += c
	}
	return uint(total)
}

// ProcessGenerationInMemory runs a full generation cycle entirely in memory.
// No database I/O during the generation — just CPU work. Returns the next
// generation's living units (survivors + offspring).
func (p *Population) ProcessGenerationInMemory(units []*Unit) []*Unit {
	config := p.PopulationConfig
	ranker := NewFitnessRanker(config.FitnessConfig)

	effectiveInput := config.EvaluatorConfig.ComputeEffectiveInputCellCount(p.CurrentGeneration)
	effectiveOutput := config.EvaluatorConfig.OutputCellCount
	if effectiveInput < effectiveOutput {
		effectiveOutput = effectiveInput
	}

	log.Printf("Generation %d: effective input cells = %d, units = %d, GOMAXPROCS = %d",
		p.CurrentGeneration, effectiveInput, len(units), runtime.GOMAXPROCS(0))

	genStart := time.Now()

	// Phase 1 — Evaluate & Threshold Select (parallel)
	evalStart := time.Now()
	selector := NewSelector(config.SelectorConfig)
	cpus := runtime.NumCPU()
	var wg sync.WaitGroup
	rounds := config.EvaluatorConfig.EvalRounds

	chunkSize := len(units) / cpus
	if chunkSize == 0 {
		chunkSize = 1
	}
	for i := 0; i < cpus; i++ {
		start := i * chunkSize
		if start >= len(units) {
			break
		}
		end := start + chunkSize
		if i == cpus-1 || end > len(units) {
			end = len(units)
		}
		wg.Add(1)
		go func(chunk []*Unit) {
			defer wg.Done()
			evaluator := NewEvaluator(config.EvaluatorConfig)
			for _, unit := range chunk {
				var eval *Evaluation
				if rounds > 1 {
					eval = evaluator.EvaluateMultiRoundWithCellCounts(unit, rounds, effectiveInput, effectiveOutput)
				} else if effectiveInput != config.EvaluatorConfig.InputCellCount {
					eval = evaluator.EvaluateWithCellCounts(unit, effectiveInput, effectiveOutput)
				} else {
					eval = evaluator.Evaluate(unit)
				}
				reason := selector.Select(unit, eval, p.CurrentGeneration)
				if reason != 0 {
					unit.Alive = Dead
					continue
				}
				unit.IncrementAge()
				if !unit.CheckAge() {
					unit.Alive = Dead
				}
			}
		}(units[start:end])
	}
	wg.Wait()
	evalTime := time.Since(evalStart)

	// Filter to alive only
	alive := make([]*Unit, 0, len(units)/2)
	for _, u := range units {
		if u.Alive == Alive {
			alive = append(alive, u)
		}
	}
	log.Printf("Phase 1: %d/%d alive (eval: %v)", len(alive), len(units), evalTime)

	// Phase 2 — Competitive Cull (in-memory sort + filter)
	if config.CarryingCapacity > 0 && uint(len(alive)) > config.CarryingCapacity {
		cullStart := time.Now()

		// Extract latest eval for each alive unit
		type rankedIdx struct {
			idx  int
			eval *Evaluation
		}
		ranked := make([]rankedIdx, len(alive))
		for i, u := range alive {
			ranked[i] = rankedIdx{idx: i, eval: u.Evaluations[len(u.Evaluations)-1]}
		}
		sort.Slice(ranked, func(i, j int) bool {
			return ranker.CompareEvaluations(ranked[i].eval, ranked[j].eval) < 0
		})

		// Keep top CarryingCapacity
		keep := config.CarryingCapacity
		survivors := make([]*Unit, keep)
		for i := uint(0); i < keep; i++ {
			survivors[i] = alive[ranked[i].idx]
		}
		culled := uint(len(alive)) - keep
		alive = survivors
		log.Printf("Phase 2: culled %d, %d alive (%v)", culled, len(alive), time.Since(cullStart))
	}

	// Phase 3 — Reproduce (parallel Mitosis, pure CPU)
	mitosisStart := time.Now()
	maxOffspring := config.MaxOffspring
	if maxOffspring == 0 {
		maxOffspring = 1
	}

	// Rank for fitness-proportional reproduction
	type rankedForRepro struct {
		unit *Unit
		eval *Evaluation
	}
	rankedUnits := make([]rankedForRepro, len(alive))
	for i, u := range alive {
		rankedUnits[i] = rankedForRepro{unit: u, eval: u.Evaluations[len(u.Evaluations)-1]}
	}
	sort.Slice(rankedUnits, func(i, j int) bool {
		return ranker.CompareEvaluations(rankedUnits[i].eval, rankedUnits[j].eval) < 0
	})

	// Build offspring count map
	total := float64(len(rankedUnits))
	offspringCounts := make([]uint, len(rankedUnits))
	for rank := range rankedUnits {
		count := uint(float64(maxOffspring) * (1.0 - float64(rank)/total))
		if count < 1 {
			count = 1
		}
		offspringCounts[rank] = count
	}

	// Parallel Mitosis — pass ID generators for permanent ID assignment
	unitIDs := p.persist.UnitIDs
	insIDs := p.persist.InstructionIDs

	type chunk struct {
		units []*Unit
	}
	chunks := make([]chunk, cpus)
	chunkSize = len(rankedUnits) / cpus
	if chunkSize == 0 {
		chunkSize = 1
	}
	for i := 0; i < cpus; i++ {
		start := i * chunkSize
		if start >= len(rankedUnits) {
			break
		}
		end := start + chunkSize
		if i == cpus-1 || end > len(rankedUnits) {
			end = len(rankedUnits)
		}
		wg.Add(1)
		go func(idx, s, e int) {
			defer wg.Done()
			var local []*Unit
			for j := s; j < e; j++ {
				for n := uint(0); n < offspringCounts[j]; n++ {
					local = append(local, rankedUnits[j].unit.Mitosis(unitIDs, insIDs))
				}
			}
			chunks[idx].units = local
		}(i, start, end)
	}
	wg.Wait()

	var allOffspring []*Unit
	for _, c := range chunks {
		allOffspring = append(allOffspring, c.units...)
	}
	log.Printf("Phase 3: %d offspring from %d survivors (%v)", len(allOffspring), len(alive), time.Since(mitosisStart))

	// Combine survivors + offspring, clear old evaluations and parent refs to save memory
	nextGen := make([]*Unit, 0, len(alive)+len(allOffspring))
	for _, u := range alive {
		u.Evaluations = nil
		u.Parent = nil
		nextGen = append(nextGen, u)
	}
	nextGen = append(nextGen, allOffspring...)

	p.CurrentGeneration++
	log.Printf("Generation %d total: %v — next gen: %d units",
		p.CurrentGeneration-1, time.Since(genStart), len(nextGen))

	return nextGen
}

// LoadUnitsIntoMemory loads all alive units with Instructions for in-memory processing.
// Queries all shards in parallel, then merges results.
// Then warms instruction caches in parallel across all cores.
func (p *Population) LoadUnitsIntoMemory() ([]*Unit, error) {
	type shardResult struct {
		units        []*Unit
		instructions []*Instruction
		err          error
	}

	results := make([]shardResult, p.persist.NumShards)
	var wg sync.WaitGroup

	for i := uint(0); i < p.persist.NumShards; i++ {
		wg.Add(1)
		go func(shard uint) {
			defer wg.Done()
			db := p.persist.Shards[shard]
			units, unitErr := queryUnits(db, p.ID)
			if unitErr != nil {
				results[shard] = shardResult{err: fmt.Errorf("shard %d units: %w", shard, unitErr)}
				return
			}
			instructions, insErr := queryInstructions(db, p.ID)
			if insErr != nil {
				results[shard] = shardResult{err: fmt.Errorf("shard %d instructions: %w", shard, insErr)}
				return
			}
			results[shard] = shardResult{units: units, instructions: instructions}
		}(i)
	}
	wg.Wait()

	// Merge all shards
	var allUnits []*Unit
	var allInstructions []*Instruction
	for _, r := range results {
		if r.err != nil {
			return nil, r.err
		}
		allUnits = append(allUnits, r.units...)
		allInstructions = append(allInstructions, r.instructions...)
	}

	if len(allUnits) == 0 {
		return allUnits, nil
	}
	log.Printf("Loaded %d unit rows, %d instruction rows, mapping...", len(allUnits), len(allInstructions))

	// Map instructions to units
	unitMap := make(map[uint]*Unit, len(allUnits))
	for _, u := range allUnits {
		u.Instructions = make([]*Instruction, 0, 10)
		unitMap[u.ID] = u
	}
	for _, ins := range allInstructions {
		if u, ok := unitMap[ins.UnitID]; ok {
			u.Instructions = append(u.Instructions, ins)
		}
	}

	// Warm instruction caches in parallel (decompress packed ops across all cores)
	log.Printf("Warming instruction caches...")
	cpus := runtime.NumCPU()
	chunkSize := len(allUnits) / cpus
	if chunkSize == 0 {
		chunkSize = len(allUnits)
	}
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
		go func(chunk []*Unit) {
			defer wg.Done()
			for _, u := range chunk {
				for _, ins := range u.Instructions {
					ins.EnsureDecompressed()
				}
			}
		}(allUnits[start:end])
	}
	wg.Wait()

	return allUnits, nil
}

func queryUnits(db *sql.DB, popID uint) ([]*Unit, error) {
	rows, err := db.Query(`SELECT id, population_id, parent_id, age, generation, lifespan, mutation_chance, alive
		FROM units WHERE population_id = ? AND alive = ?`, popID, Alive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var units []*Unit
	for rows.Next() {
		u := &Unit{}
		var parentID sql.NullInt64
		if err := rows.Scan(&u.ID, &u.PopulationID, &parentID, &u.Age, &u.Generation, &u.Lifespan, &u.MutationChance, &u.Alive); err != nil {
			return nil, err
		}
		if parentID.Valid {
			pid := uint(parentID.Int64)
			u.ParentID = &pid
		}
		units = append(units, u)
	}
	return units, rows.Err()
}

func queryInstructions(db *sql.DB, popID uint) ([]*Instruction, error) {
	rows, err := db.Query(`SELECT id, unit_id, age, initial_op_set, ops
		FROM instructions WHERE unit_id IN (SELECT id FROM units WHERE population_id = ? AND alive = ?)`,
		popID, Alive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instructions []*Instruction
	for rows.Next() {
		ins := &Instruction{}
		if err := rows.Scan(&ins.ID, &ins.UnitID, &ins.Age, &ins.InitialOpSet, &ins.Ops); err != nil {
			return nil, err
		}
		instructions = append(instructions, ins)
	}
	return instructions, rows.Err()
}

// PersistPopulationState saves the current generation state to the database.
// Survivors (units with IDs that were loaded from DB) get UPDATEd.
// Offspring (units with new IDs from IDGenerator) get INSERTed.
// Units that were loaded but are no longer present are marked dead.
func (p *Population) PersistPopulationState(units []*Unit) error {
	// Recompress any instructions that were mutated in-memory
	for _, u := range units {
		for _, ins := range u.Instructions {
			ins.EnsureCompressed()
		}
	}

	// Update population generation on shard0
	if _, err := p.persist.shard0().Exec("UPDATE populations SET current_generation = ? WHERE id = ?",
		p.CurrentGeneration, p.ID); err != nil {
		return fmt.Errorf("failed to save population: %w", err)
	}

	// Pre-assign IDs to all units so sharding works correctly
	for _, u := range units {
		u.ID = p.persist.UnitIDs.Next()
		u.PopulationID = p.ID
		u.ParentID = nil
		u.Parent = nil
		for _, ins := range u.Instructions {
			ins.ID = p.persist.InstructionIDs.Next()
			ins.UnitID = u.ID
		}
	}

	// Parallel shard writes
	return p.persist.writeSharded(units, func(tx *sql.Tx, batch []*Unit) error {
		// Mark all existing alive units as dead on this shard
		if _, err := tx.Exec("UPDATE units SET alive = ? WHERE population_id = ? AND alive = ?",
			Dead, p.ID, Alive); err != nil {
			return err
		}

		// Insert current generation as new records
		for _, u := range batch {
			if _, err := tx.Exec(`INSERT INTO units (id, population_id, parent_id, age, generation, lifespan, mutation_chance, alive)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				u.ID, u.PopulationID, nullableUint(u.ParentID), u.Age, u.Generation, u.Lifespan, u.MutationChance, u.Alive); err != nil {
				return fmt.Errorf("failed to insert unit: %w", err)
			}

			for _, ins := range u.Instructions {
				if _, err := tx.Exec(`INSERT INTO instructions (id, unit_id, age, initial_op_set, ops) VALUES (?, ?, ?, ?, ?)`,
					ins.ID, ins.UnitID, ins.Age, ins.InitialOpSet, ins.Ops); err != nil {
					return fmt.Errorf("failed to insert instruction: %w", err)
				}
			}
		}
		return nil
	})
}

func (p *Population) ProcessGeneration() error {
	config := p.PopulationConfig
	ranker := NewFitnessRanker(config.FitnessConfig)

	// Curriculum learning: compute effective input cell count for this generation
	effectiveInput := config.EvaluatorConfig.ComputeEffectiveInputCellCount(p.CurrentGeneration)
	effectiveOutput := config.EvaluatorConfig.OutputCellCount
	if effectiveInput < effectiveOutput {
		effectiveOutput = effectiveInput
	}

	log.Printf("Generation %d: effective input cells = %d, GOMAXPROCS = %d",
		p.CurrentGeneration, effectiveInput, runtime.GOMAXPROCS(0))

	genStart := time.Now()

	// Phase 1 — Load + Evaluate & Threshold Select (parallel, no I/O during eval)
	phaseStart := time.Now()
	log.Printf("Phase 1: Evaluate & threshold select")

	allUnits, err := p.LoadUnitsIntoMemory()
	if err != nil {
		return fmt.Errorf("failed to load units: %w", err)
	}
	loadTime := time.Since(phaseStart)

	selector := NewSelector(config.SelectorConfig)
	cpus := runtime.NumCPU()
	var wg sync.WaitGroup

	// Split units across CPUs and evaluate in parallel
	evalStart := time.Now()
	chunkSize := len(allUnits) / cpus
	for i := 0; i < cpus; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if i == cpus-1 {
			end = len(allUnits)
		}
		wg.Add(1)
		go func(chunk []*Unit) {
			defer wg.Done()
			evaluator := NewEvaluator(config.EvaluatorConfig)
			rounds := config.EvaluatorConfig.EvalRounds
			for _, unit := range chunk {
				var eval *Evaluation
				if rounds > 1 {
					eval = evaluator.EvaluateMultiRoundWithCellCounts(unit, rounds, effectiveInput, effectiveOutput)
				} else if effectiveInput != config.EvaluatorConfig.InputCellCount {
					eval = evaluator.EvaluateWithCellCounts(unit, effectiveInput, effectiveOutput)
				} else {
					eval = evaluator.Evaluate(unit)
				}
				reason := selector.Select(unit, eval, p.CurrentGeneration)
				if reason != 0 {
					unit.Die(reason)
					continue
				}
				unit.IncrementAge()
				if !unit.CheckAge() {
					unit.Die(FailedLifespan)
				}
			}
		}(allUnits[start:end])
	}
	wg.Wait()
	evalTime := time.Since(evalStart)

	// Bulk persist all results
	persistStart := time.Now()
	if err := p.persist.PersistEvaluatedBatch(allUnits); err != nil {
		return fmt.Errorf("failed to persist evaluation results: %w", err)
	}
	persistTime := time.Since(persistStart)

	alive := p.GetAliveCount()
	log.Printf("Phase 1 complete: %d/%d alive (load: %v, eval: %v, persist: %v)",
		alive, len(allUnits), loadTime, evalTime, persistTime)

	// Phase 2 — Competitive Cull (uses in-memory evals, no DB query)
	if config.CarryingCapacity > 0 {
		phaseStart = time.Now()
		log.Printf("Phase 2: Competitive cull (capacity: %d, elitism: %d)", config.CarryingCapacity, config.Elitism)
		culler := NewCompetitiveCuller(p.persist, p.ID, config.CarryingCapacity, config.Elitism, p.persist.Config.BatchSize, ranker)
		culled, err := culler.CullFromUnits(allUnits)
		if err != nil {
			return fmt.Errorf("competitive cull failed: %w", err)
		}
		alive = p.GetAliveCount()
		log.Printf("Phase 2 complete: culled %d, %d alive (%v)", culled, alive, time.Since(phaseStart))
	}

	// Phase 3 — Reproduce (uses in-memory units, no DB reload)
	phaseStart = time.Now()
	log.Printf("Phase 3: Reproduce")
	reproducer := NewReproducer(p.persist, p.ID, config.MaxOffspring, p.persist.Config.BatchSize, ranker,
		p.persist.UnitIDs, p.persist.InstructionIDs)
	offspring, err := reproducer.ReproduceFromUnits(allUnits)
	if err != nil {
		return fmt.Errorf("reproduction failed: %w", err)
	}
	alive = p.GetAliveCount()
	log.Printf("Phase 3 complete: %d offspring, %d total alive (%v)", offspring, alive, time.Since(phaseStart))

	// Phase 4 — Bookkeeping
	p.CurrentGeneration++
	if _, err := p.persist.shard0().Exec("UPDATE populations SET current_generation = ? WHERE id = ?",
		p.CurrentGeneration, p.ID); err != nil {
		return fmt.Errorf("failed to save population after generation: %w", err)
	}
	log.Printf("Generation %d total: %v", p.CurrentGeneration-1, time.Since(genStart))

	return nil
}

// evalBatchSize returns the configured eval batch size, defaulting to 10000.
func (p *Population) evalBatchSize() int {
	if p.persist.Config.EvalBatchSize > 0 {
		return int(p.persist.Config.EvalBatchSize)
	}
	return 10000
}

// warmInstructionCaches decompresses packed ops across all cores in parallel.
func warmInstructionCaches(units []*Unit) {
	cpus := runtime.NumCPU()
	if len(units) == 0 {
		return
	}
	chunkSize := len(units) / cpus
	if chunkSize == 0 {
		chunkSize = len(units)
	}
	var wg sync.WaitGroup
	for i := 0; i < cpus; i++ {
		start := i * chunkSize
		if start >= len(units) {
			break
		}
		end := start + chunkSize
		if i == cpus-1 || end > len(units) {
			end = len(units)
		}
		wg.Add(1)
		go func(chunk []*Unit) {
			defer wg.Done()
			for _, u := range chunk {
				for _, ins := range u.Instructions {
					ins.EnsureDecompressed()
				}
			}
		}(units[start:end])
	}
	wg.Wait()
}

// loadBatch loads a batch of units with their instructions from a single shard.
// Returns the loaded units and the cursor ID for the next batch.
func (p *Population) loadBatch(db *sql.DB, shard uint, popID uint, afterID, maxID uint, batchSize int) ([]*Unit, uint, error) {
	units, err := queryUnitsBatch(db, popID, afterID, maxID, batchSize)
	if err != nil {
		return nil, 0, fmt.Errorf("shard %d: failed to query units batch: %w", shard, err)
	}
	if len(units) == 0 {
		return nil, afterID, nil
	}

	unitIDs := make([]uint, len(units))
	unitMap := make(map[uint]*Unit, len(units))
	for i, u := range units {
		unitIDs[i] = u.ID
		u.Instructions = make([]*Instruction, 0, 10)
		unitMap[u.ID] = u
	}

	instructions, err := queryInstructionsForUnits(db, unitIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("shard %d: failed to query instructions: %w", shard, err)
	}
	for _, ins := range instructions {
		if u, ok := unitMap[ins.UnitID]; ok {
			u.Instructions = append(u.Instructions, ins)
		}
	}

	warmInstructionCaches(units)

	nextAfterID := units[len(units)-1].ID
	return units, nextAfterID, nil
}

// ForEachUnitBatch iterates alive units for this population in batches,
// loading instructions and warming caches for each batch. Processes all
// shards in parallel. Within each shard, pipelines DB reads with fn calls
// so the next batch is loading while the current batch is processed.
// The maxIDs parameter caps iteration per shard (use nil to query current max).
// fn must be safe for concurrent calls from multiple goroutines.
func (p *Population) ForEachUnitBatch(batchSize int, maxIDs []uint, fn func(units []*Unit) error) error {
	if batchSize <= 0 {
		batchSize = p.evalBatchSize()
	}

	errs := make([]error, p.persist.NumShards)
	var wg sync.WaitGroup

	for shard := uint(0); shard < p.persist.NumShards; shard++ {
		wg.Add(1)
		go func(shard uint) {
			defer wg.Done()
			db := p.persist.Shards[shard]

			var maxID uint
			if maxIDs != nil {
				maxID = maxIDs[shard]
			} else {
				var err error
				maxID, err = queryMaxUnitID(db, p.ID)
				if err != nil {
					errs[shard] = fmt.Errorf("shard %d: failed to get max unit ID: %w", shard, err)
					return
				}
			}
			if maxID == 0 {
				return
			}

			// Pipeline: pre-fetch next batch while processing current batch
			type batchResult struct {
				units   []*Unit
				afterID uint
				err     error
			}

			// Load first batch synchronously
			units, afterID, err := p.loadBatch(db, shard, p.ID, 0, maxID, batchSize)
			if err != nil {
				errs[shard] = err
				return
			}

			for len(units) > 0 {
				// Start loading next batch in background
				nextCh := make(chan batchResult, 1)
				go func(after uint) {
					u, a, e := p.loadBatch(db, shard, p.ID, after, maxID, batchSize)
					nextCh <- batchResult{units: u, afterID: a, err: e}
				}(afterID)

				// Process current batch
				if err := fn(units); err != nil {
					errs[shard] = err
					<-nextCh // drain
					return
				}

				// Wait for next batch
				next := <-nextCh
				if next.err != nil {
					errs[shard] = next.err
					return
				}
				units = next.units
				afterID = next.afterID
			}
		}(shard)
	}
	wg.Wait()
	return firstError(errs)
}

// evaluateAndSelectBatch evaluates and selects a batch of units in parallel.
func evaluateAndSelectBatch(units []*Unit, config *PopulationConfig, effectiveInput, effectiveOutput, generation uint) {
	cpus := runtime.NumCPU()
	selector := NewSelector(config.SelectorConfig)
	rounds := config.EvaluatorConfig.EvalRounds

	chunkSize := len(units) / cpus
	if chunkSize == 0 {
		chunkSize = 1
	}

	var wg sync.WaitGroup
	for i := 0; i < cpus; i++ {
		start := i * chunkSize
		if start >= len(units) {
			break
		}
		end := start + chunkSize
		if i == cpus-1 || end > len(units) {
			end = len(units)
		}
		wg.Add(1)
		go func(chunk []*Unit) {
			defer wg.Done()
			evaluator := NewEvaluator(config.EvaluatorConfig)
			for _, unit := range chunk {
				var eval *Evaluation
				if rounds > 1 {
					eval = evaluator.EvaluateMultiRoundWithCellCounts(unit, rounds, effectiveInput, effectiveOutput)
				} else if effectiveInput != config.EvaluatorConfig.InputCellCount {
					eval = evaluator.EvaluateWithCellCounts(unit, effectiveInput, effectiveOutput)
				} else {
					eval = evaluator.Evaluate(unit)
				}
				reason := selector.Select(unit, eval, generation)
				if reason != 0 {
					unit.Die(reason)
					continue
				}
				unit.IncrementAge()
				if !unit.CheckAge() {
					unit.Die(FailedLifespan)
				}
			}
		}(units[start:end])
	}
	wg.Wait()
}

// ProcessGenerationStreaming runs a full generation cycle using streaming
// batch processing to keep memory bounded. Each phase streams units from DB
// in batches rather than loading the entire population at once.
func (p *Population) ProcessGenerationStreaming() error {
	config := p.PopulationConfig
	ranker := NewFitnessRanker(config.FitnessConfig)
	batchSize := p.evalBatchSize()

	effectiveInput := config.EvaluatorConfig.ComputeEffectiveInputCellCount(p.CurrentGeneration)
	effectiveOutput := config.EvaluatorConfig.OutputCellCount
	if effectiveInput < effectiveOutput {
		effectiveOutput = effectiveInput
	}

	effFidelity := config.SelectorConfig.EffectiveSetFidelity(p.CurrentGeneration)
	effSortedness := config.SelectorConfig.EffectiveSortedness(p.CurrentGeneration)
	log.Printf("Generation %d: streaming mode, effective input cells = %d, sel_fid=%d sel_sort=%d, eval_batch_size = %d, GOMAXPROCS = %d",
		p.CurrentGeneration, effectiveInput, effFidelity, effSortedness, batchSize, runtime.GOMAXPROCS(0))

	genStart := time.Now()

	// Phase 1 — Streaming Evaluate & Threshold Select
	phaseStart := time.Now()
	log.Printf("Phase 1: Streaming evaluate & threshold select")

	var totalUnits, totalAlive atomic.Uint64
	err := p.ForEachUnitBatch(batchSize, nil, func(units []*Unit) error {
		totalUnits.Add(uint64(len(units)))

		evaluateAndSelectBatch(units, config, effectiveInput, effectiveOutput, p.CurrentGeneration)

		var batchAlive uint64
		for _, u := range units {
			if u.Alive == Alive {
				batchAlive++
			}
		}
		totalAlive.Add(batchAlive)

		// Persist evaluation results + alive/dead status for this batch
		if err := p.persist.PersistEvaluatedBatch(units); err != nil {
			return fmt.Errorf("failed to persist evaluation batch: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("phase 1 failed: %w", err)
	}

	alive := p.GetAliveCount()
	log.Printf("Phase 1 complete: %d/%d alive (%v)", alive, totalUnits.Load(), time.Since(phaseStart))

	// Phase 2 — Competitive Cull (lightweight — queries eval scores only)
	if config.CarryingCapacity > 0 {
		phaseStart = time.Now()
		log.Printf("Phase 2: Competitive cull (capacity: %d, elitism: %d)", config.CarryingCapacity, config.Elitism)
		culler := NewCompetitiveCuller(p.persist, p.ID, config.CarryingCapacity, config.Elitism, p.persist.Config.BatchSize, ranker)
		culled, err := culler.Cull()
		if err != nil {
			return fmt.Errorf("competitive cull failed: %w", err)
		}
		alive = p.GetAliveCount()
		log.Printf("Phase 2 complete: culled %d, %d alive (%v)", culled, alive, time.Since(phaseStart))
	}

	// Phase 2.5 — Prune dead unit data (evaluations, instructions, mutations, tombstones)
	phaseStart = time.Now()
	dEval, dIns, dMut, dTomb, pruneErr := p.PruneDeadUnitData()
	if pruneErr != nil {
		log.Printf("Warning: dead unit data prune failed: %v", pruneErr)
	} else {
		log.Printf("Phase 2.5 complete: pruned evals=%d ins=%d mut=%d tomb=%d (%v)",
			dEval, dIns, dMut, dTomb, time.Since(phaseStart))
	}

	// Phase 3 — Streaming Reproduce
	phaseStart = time.Now()
	log.Printf("Phase 3: Streaming reproduce")
	reproducer := NewReproducer(p.persist, p.ID, config.MaxOffspring, p.persist.Config.BatchSize, ranker,
		p.persist.UnitIDs, p.persist.InstructionIDs)
	offspring, err := reproducer.ReproduceStreaming(batchSize)
	if err != nil {
		return fmt.Errorf("streaming reproduction failed: %w", err)
	}
	alive = p.GetAliveCount()
	log.Printf("Phase 3 complete: %d offspring, %d total alive (%v)", offspring, alive, time.Since(phaseStart))

	// Phase 4 — Bookkeeping
	p.CurrentGeneration++
	if _, err := p.persist.shard0().Exec("UPDATE populations SET current_generation = ? WHERE id = ?",
		p.CurrentGeneration, p.ID); err != nil {
		return fmt.Errorf("failed to save population after generation: %w", err)
	}
	log.Printf("Generation %d total: %v", p.CurrentGeneration-1, time.Since(genStart))

	return nil
}

type pruneShardCounts struct {
	totalUnits, deletedUnits uint
	deletedIns, deletedMut   uint
	deletedEval, deletedTomb uint
}

// PruneResult holds counts from a prune operation.
type PruneResult struct {
	AliveUnits       uint
	AncestorUnits    uint
	TotalUnits       uint
	DeletedUnits     uint
	DeletedInstructions uint
	DeletedMutations    uint
	DeletedEvaluations  uint
	DeletedTombstones   uint
}

// Prune removes dead units that are NOT ancestors of alive units, along with their
// associated instructions, mutations, evaluations, and tombstones. If dryRun is true,
// it reports what would be deleted without actually deleting. After deletion, each
// shard is VACUUMed to reclaim disk space.
func (p *Population) Prune(dryRun bool) (*PruneResult, error) {
	result := &PruneResult{}
	persist := p.persist

	// Step 1: Collect alive unit IDs from all shards in parallel
	log.Printf("Prune: collecting alive unit IDs across %d shards...", persist.NumShards)
	shardAliveIDs := make([][]uint, persist.NumShards)
	errs := make([]error, persist.NumShards)
	var wg sync.WaitGroup

	for i := uint(0); i < persist.NumShards; i++ {
		wg.Add(1)
		go func(shard uint) {
			defer wg.Done()
			ids, err := queryAliveUnitIDs(persist.Shards[shard], p.ID)
			if err != nil {
				errs[shard] = fmt.Errorf("shard %d: %w", shard, err)
				return
			}
			shardAliveIDs[shard] = ids
		}(i)
	}
	wg.Wait()
	if err := firstError(errs); err != nil {
		return nil, fmt.Errorf("failed to collect alive IDs: %w", err)
	}

	keepSet := make(map[uint]bool)
	for _, ids := range shardAliveIDs {
		for _, id := range ids {
			keepSet[id] = true
		}
	}
	result.AliveUnits = uint(len(keepSet))
	log.Printf("Prune: %d alive units found", result.AliveUnits)

	// Step 2: Trace ancestry iteratively — find all ancestors of alive units
	// Each iteration: collect parent_ids not yet in keepSet, group by shard, query parents
	for iteration := 1; ; iteration++ {
		// Collect IDs we need parents for, grouped by shard
		needParents := make([][]uint, persist.NumShards)
		for id := range keepSet {
			shard := id % persist.NumShards
			needParents[shard] = append(needParents[shard], id)
		}

		// Query parent_ids from each shard in parallel
		shardParentMaps := make([]map[uint]uint, persist.NumShards)
		for i := uint(0); i < persist.NumShards; i++ {
			errs[i] = nil
		}
		for i := uint(0); i < persist.NumShards; i++ {
			if len(needParents[i]) == 0 {
				continue
			}
			wg.Add(1)
			go func(shard uint) {
				defer wg.Done()
				m, err := queryUnitParents(persist.Shards[shard], needParents[shard])
				if err != nil {
					errs[shard] = fmt.Errorf("shard %d parents: %w", shard, err)
					return
				}
				shardParentMaps[shard] = m
			}(i)
		}
		wg.Wait()
		if err := firstError(errs); err != nil {
			return nil, fmt.Errorf("failed to trace ancestry: %w", err)
		}

		// Add newly discovered parents to keepSet
		newParents := 0
		for _, m := range shardParentMaps {
			for _, parentID := range m {
				if parentID != 0 && !keepSet[parentID] {
					keepSet[parentID] = true
					newParents++
				}
			}
		}

		log.Printf("Prune: ancestry iteration %d found %d new ancestors (total keep: %d)", iteration, newParents, len(keepSet))
		if newParents == 0 {
			break
		}
	}
	result.AncestorUnits = uint(len(keepSet)) - result.AliveUnits

	// Step 3: Count totals per shard and optionally delete
	log.Printf("Prune: %d total units to keep (%d alive + %d ancestors)", len(keepSet), result.AliveUnits, result.AncestorUnits)

	// Build per-shard keep lists
	shardKeep := make([][]uint, persist.NumShards)
	for id := range keepSet {
		shard := id % persist.NumShards
		shardKeep[shard] = append(shardKeep[shard], id)
	}

	counts := make([]pruneShardCounts, persist.NumShards)

	for i := uint(0); i < persist.NumShards; i++ {
		errs[i] = nil
	}

	for i := uint(0); i < persist.NumShards; i++ {
		wg.Add(1)
		go func(shard uint) {
			defer wg.Done()
			db := persist.Shards[shard]

			// Count total units on this shard
			var total int64
			if err := db.QueryRow("SELECT COUNT(*) FROM units WHERE population_id = ?", p.ID).Scan(&total); err != nil {
				errs[shard] = fmt.Errorf("shard %d count: %w", shard, err)
				return
			}
			counts[shard].totalUnits = uint(total)

			if dryRun {
				// For dry run, count what would be deleted
				keepIDs := shardKeep[shard]
				toDelete := uint(total) - uint(len(keepIDs))
				counts[shard].deletedUnits = toDelete

				// Estimate related records — count records for non-kept units
				// Use temp table approach even for counting
				if err := withTx(db, func(tx *sql.Tx) error {
					if _, err := tx.Exec("CREATE TEMP TABLE prune_keep (id INTEGER PRIMARY KEY)"); err != nil {
						return err
					}
					if err := insertIDsIntoTemp(tx, "prune_keep", keepIDs); err != nil {
						return err
					}

					var cnt int64
					if err := tx.QueryRow("SELECT COUNT(*) FROM instructions WHERE unit_id NOT IN (SELECT id FROM prune_keep) AND unit_id IN (SELECT id FROM units WHERE population_id = ?)", p.ID).Scan(&cnt); err != nil {
						return err
					}
					counts[shard].deletedIns = uint(cnt)

					if err := tx.QueryRow("SELECT COUNT(*) FROM evaluations WHERE unit_id NOT IN (SELECT id FROM prune_keep) AND unit_id IN (SELECT id FROM units WHERE population_id = ?)", p.ID).Scan(&cnt); err != nil {
						return err
					}
					counts[shard].deletedEval = uint(cnt)

					if err := tx.QueryRow("SELECT COUNT(*) FROM tombstones WHERE unit_id NOT IN (SELECT id FROM prune_keep) AND unit_id IN (SELECT id FROM units WHERE population_id = ?)", p.ID).Scan(&cnt); err != nil {
						return err
					}
					counts[shard].deletedTomb = uint(cnt)

					if err := tx.QueryRow(`SELECT COUNT(*) FROM mutations WHERE instruction_id IN
						(SELECT id FROM instructions WHERE unit_id NOT IN (SELECT id FROM prune_keep)
						 AND unit_id IN (SELECT id FROM units WHERE population_id = ?))`, p.ID).Scan(&cnt); err != nil {
						return err
					}
					counts[shard].deletedMut = uint(cnt)

					_, _ = tx.Exec("DROP TABLE IF EXISTS prune_keep")
					return nil
				}); err != nil {
					errs[shard] = fmt.Errorf("shard %d dry-run count: %w", shard, err)
				}
				return
			}

			// Actual deletion
			keepIDs := shardKeep[shard]
			if err := pruneShardData(db, p.ID, keepIDs, &counts[shard]); err != nil {
				errs[shard] = fmt.Errorf("shard %d prune: %w", shard, err)
				return
			}

			// VACUUM to reclaim space
			log.Printf("Prune: VACUUMing shard %d...", shard)
			if _, err := db.Exec("VACUUM"); err != nil {
				errs[shard] = fmt.Errorf("shard %d vacuum: %w", shard, err)
			}
		}(i)
	}
	wg.Wait()
	if err := firstError(errs); err != nil {
		return nil, err
	}

	// Aggregate counts
	for _, c := range counts {
		result.TotalUnits += c.totalUnits
		result.DeletedUnits += c.deletedUnits
		result.DeletedInstructions += c.deletedIns
		result.DeletedMutations += c.deletedMut
		result.DeletedEvaluations += c.deletedEval
		result.DeletedTombstones += c.deletedTomb
	}

	return result, nil
}

// pruneShardData deletes non-kept units and their related data from a single shard.
// Uses a temp table of kept IDs for efficient indexed joins during deletion.
func pruneShardData(db *sql.DB, popID uint, keepIDs []uint, counts *pruneShardCounts) error {
	return withTx(db, func(tx *sql.Tx) error {
		// Create temp table with kept IDs
		if _, err := tx.Exec("CREATE TEMP TABLE prune_keep (id INTEGER PRIMARY KEY)"); err != nil {
			return err
		}
		if err := insertIDsIntoTemp(tx, "prune_keep", keepIDs); err != nil {
			return err
		}

		// Delete mutations for instructions belonging to pruned units
		res, err := tx.Exec(`DELETE FROM mutations WHERE instruction_id IN
			(SELECT id FROM instructions WHERE unit_id NOT IN (SELECT id FROM prune_keep)
			 AND unit_id IN (SELECT id FROM units WHERE population_id = ?))`, popID)
		if err != nil {
			return fmt.Errorf("delete mutations: %w", err)
		}
		n, _ := res.RowsAffected()
		counts.deletedMut = uint(n)

		// Delete instructions for pruned units
		res, err = tx.Exec("DELETE FROM instructions WHERE unit_id NOT IN (SELECT id FROM prune_keep) AND unit_id IN (SELECT id FROM units WHERE population_id = ?)", popID)
		if err != nil {
			return fmt.Errorf("delete instructions: %w", err)
		}
		n, _ = res.RowsAffected()
		counts.deletedIns = uint(n)

		// Delete evaluations for pruned units
		res, err = tx.Exec("DELETE FROM evaluations WHERE unit_id NOT IN (SELECT id FROM prune_keep) AND unit_id IN (SELECT id FROM units WHERE population_id = ?)", popID)
		if err != nil {
			return fmt.Errorf("delete evaluations: %w", err)
		}
		n, _ = res.RowsAffected()
		counts.deletedEval = uint(n)

		// Delete tombstones for pruned units
		res, err = tx.Exec("DELETE FROM tombstones WHERE unit_id NOT IN (SELECT id FROM prune_keep) AND unit_id IN (SELECT id FROM units WHERE population_id = ?)", popID)
		if err != nil {
			return fmt.Errorf("delete tombstones: %w", err)
		}
		n, _ = res.RowsAffected()
		counts.deletedTomb = uint(n)

		// Delete pruned units
		res, err = tx.Exec("DELETE FROM units WHERE population_id = ? AND id NOT IN (SELECT id FROM prune_keep)", popID)
		if err != nil {
			return fmt.Errorf("delete units: %w", err)
		}
		n, _ = res.RowsAffected()
		counts.deletedUnits = uint(n)

		// Drop temp table
		_, _ = tx.Exec("DROP TABLE IF EXISTS prune_keep")
		return nil
	})
}

// insertIDsIntoTemp inserts IDs into a temp table in batches of 900
// to stay under SQLite's 999-variable limit.
func insertIDsIntoTemp(tx *sql.Tx, table string, ids []uint) error {
	const batchSize = 900
	for start := 0; start < len(ids); start += batchSize {
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[start:end]

		var sb strings.Builder
		sb.WriteString("INSERT INTO " + table + " (id) VALUES ")
		args := make([]interface{}, len(chunk))
		for i, id := range chunk {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString("(?)")
			args[i] = id
		}
		if _, err := tx.Exec(sb.String(), args...); err != nil {
			return err
		}
	}
	return nil
}

// PruneDeadUnitData deletes evaluations, instructions, mutations, and tombstones
// for dead units across all shards. Unlike Prune(), this does NOT delete the unit
// rows themselves (preserving parent_id lineage chains) and does NOT trace ancestry
// or VACUUM. It's designed to run periodically during generation processing to keep
// disk usage under control.
func (p *Population) PruneDeadUnitData() (deletedEvals, deletedIns, deletedMut, deletedTomb uint, err error) {
	persist := p.persist
	errs := make([]error, persist.NumShards)
	counts := make([][4]uint, persist.NumShards) // [evals, ins, mut, tomb]

	var wg sync.WaitGroup
	for i := uint(0); i < persist.NumShards; i++ {
		wg.Add(1)
		go func(shard uint) {
			defer wg.Done()
			db := persist.Shards[shard]
			errs[shard] = withTx(db, func(tx *sql.Tx) error {
				// Delete evaluations for dead units
				res, err := tx.Exec(`DELETE FROM evaluations WHERE unit_id IN
					(SELECT id FROM units WHERE population_id = ? AND alive = ?)`, p.ID, Dead)
				if err != nil {
					return fmt.Errorf("delete evaluations: %w", err)
				}
				n, _ := res.RowsAffected()
				counts[shard][0] = uint(n)

				// Delete mutations for instructions of dead units
				res, err = tx.Exec(`DELETE FROM mutations WHERE instruction_id IN
					(SELECT i.id FROM instructions i
					 JOIN units u ON u.id = i.unit_id
					 WHERE u.population_id = ? AND u.alive = ?)`, p.ID, Dead)
				if err != nil {
					return fmt.Errorf("delete mutations: %w", err)
				}
				n, _ = res.RowsAffected()
				counts[shard][2] = uint(n)

				// Delete instructions for dead units
				res, err = tx.Exec(`DELETE FROM instructions WHERE unit_id IN
					(SELECT id FROM units WHERE population_id = ? AND alive = ?)`, p.ID, Dead)
				if err != nil {
					return fmt.Errorf("delete instructions: %w", err)
				}
				n, _ = res.RowsAffected()
				counts[shard][1] = uint(n)

				// Delete tombstones for dead units
				res, err = tx.Exec(`DELETE FROM tombstones WHERE unit_id IN
					(SELECT id FROM units WHERE population_id = ? AND alive = ?)`, p.ID, Dead)
				if err != nil {
					return fmt.Errorf("delete tombstones: %w", err)
				}
				n, _ = res.RowsAffected()
				counts[shard][3] = uint(n)

				return nil
			})
		}(i)
	}
	wg.Wait()

	if err := firstError(errs); err != nil {
		return 0, 0, 0, 0, err
	}

	for _, c := range counts {
		deletedEvals += c[0]
		deletedIns += c[1]
		deletedMut += c[2]
		deletedTomb += c[3]
	}
	return deletedEvals, deletedIns, deletedMut, deletedTomb, nil
}

// createTestSchema creates the database schema for tests. Exported for test files.
func createTestSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS populations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			current_generation INTEGER DEFAULT 0,
			unit_count INTEGER,
			synthesis_pool INTEGER,
			carrying_capacity INTEGER,
			elitism INTEGER,
			max_offspring INTEGER,
			unit_mutation_chance REAL,
			unit_instruction_count INTEGER,
			unit_ins_op_set_count INTEGER,
			unit_lifespan INTEGER,
			eval_machine_max_instruction_execution_count INTEGER,
			eval_machine_memory_cell_count INTEGER,
			eval_input_cell_count INTEGER,
			eval_output_cell_count INTEGER,
			eval_synthesis_input_cell_count INTEGER,
			eval_input_cell_start INTEGER,
			eval_input_cell_step INTEGER,
			eval_eval_rounds INTEGER,
			sel_machine_run INTEGER,
			sel_set_fidelity INTEGER,
			sel_sortedness INTEGER,
			sel_set_fidelity_start INTEGER DEFAULT 0,
			sel_set_fidelity_step INTEGER DEFAULT 0,
			sel_sortedness_start INTEGER DEFAULT 0,
			sel_sortedness_step INTEGER DEFAULT 0,
			sel_instruction_count INTEGER,
			sel_instructions_executed INTEGER,
			fit_sortedness_priority INTEGER,
			fit_set_fidelity_priority INTEGER,
			fit_efficiency_priority INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS units (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			population_id INTEGER,
			parent_id INTEGER,
			age INTEGER DEFAULT 0,
			generation INTEGER DEFAULT 0,
			lifespan INTEGER,
			mutation_chance REAL,
			alive INTEGER DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS instructions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			unit_id INTEGER,
			age INTEGER DEFAULT 0,
			initial_op_set BLOB,
			ops BLOB
		)`,
		`CREATE TABLE IF NOT EXISTS mutations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			instruction_id INTEGER,
			position1 INTEGER,
			position2 INTEGER,
			meta_op INTEGER,
			op INTEGER,
			chance REAL
		)`,
		`CREATE TABLE IF NOT EXISTS evaluations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			unit_id INTEGER,
			machine_run INTEGER,
			set_fidelity INTEGER,
			sortedness INTEGER,
			instruction_count INTEGER,
			instructions_executed INTEGER,
			machine_error TEXT,
			input BLOB,
			output BLOB
		)`,
		`CREATE TABLE IF NOT EXISTS tombstones (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			unit_id INTEGER,
			reason INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_units_pop_alive ON units(population_id, alive)`,
		`CREATE INDEX IF NOT EXISTS idx_instructions_unit_id ON instructions(unit_id)`,
		`CREATE INDEX IF NOT EXISTS idx_evaluations_unit_id ON evaluations(unit_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tombstones_unit_id ON tombstones(unit_id)`,
		`CREATE INDEX IF NOT EXISTS idx_mutations_instruction_id ON mutations(instruction_id)`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("schema creation failed: %w\nStatement: %s", err, stmt)
		}
	}
	return nil
}

// insertPopulation inserts a population into the DB for tests.
func insertPopulation(db *sql.DB, pop *Population) error {
	cols, vals := populationInsertValues(pop)
	query := "INSERT INTO populations (" + strings.Join(cols, ", ") + ") VALUES (" + placeholders(len(cols)) + ")"
	_, err := db.Exec(query, vals...)
	return err
}

// testPersistence creates a 1-shard in-memory Persistence for tests.
func testPersistence(db *sql.DB) *Persistence {
	return &Persistence{
		Config:         &PersistenceConfig{BatchSize: 1000},
		Shards:         []*sql.DB{db},
		NumShards:      1,
		UnitIDs:        NewIDGenerator(0),
		InstructionIDs: NewIDGenerator(0),
		EvalIDs:        NewIDGenerator(0),
		MutationIDs:    NewIDGenerator(0),
		TombstoneIDs:   NewIDGenerator(0),
		PopIDs:         NewIDGenerator(0),
	}
}
