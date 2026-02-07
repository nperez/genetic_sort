package genetic_sort

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	_ "github.com/glebarez/go-sqlite"
	bf "nickandperla.net/brainfuck"
)

type PersistenceConfig struct {
	Name          string   `toml:"name"`
	Path          string   `toml:"path"`
	ShardCount    uint     `toml:"shard_count"`
	SQLitePragmas []string `toml:"pragmas"`
	SQLiteOptions []string `toml:"options"`
	BatchSize     uint     `toml:"batch_size"`
	EvalBatchSize uint     `toml:"eval_batch_size"`
	Seed          int64    `toml:"seed"`
}

type IDGenerator struct {
	next atomic.Uint64
}

func NewIDGenerator(start uint64) *IDGenerator {
	g := &IDGenerator{}
	g.next.Store(start)
	return g
}

func (g *IDGenerator) Next() uint {
	return uint(g.next.Add(1))
}

type Persistence struct {
	Config         *PersistenceConfig
	Shards         []*sql.DB
	NumShards      uint
	UnitIDs        *IDGenerator
	InstructionIDs *IDGenerator
	EvalIDs        *IDGenerator
	MutationIDs    *IDGenerator
	TombstoneIDs   *IDGenerator
	PopIDs         *IDGenerator
}

func (p *Persistence) shardFor(unitID uint) *sql.DB {
	return p.Shards[unitID%p.NumShards]
}

func (p *Persistence) shard0() *sql.DB {
	return p.Shards[0]
}

func NewPersistence(config *PersistenceConfig) (*Persistence, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if len(config.Path) == 0 {
		return nil, fmt.Errorf("Path to database must be defined")
	}

	if len(config.Name) == 0 {
		return nil, fmt.Errorf("Name of database must be defined")
	}

	numShards := config.ShardCount
	if numShards == 0 {
		numShards = 1
	}

	shards := make([]*sql.DB, numShards)
	for i := uint(0); i < numShards; i++ {
		dsn := buildDSN(config, i, numShards)
		if DEBUG {
			log.Printf("Opening shard %d: %s", i, dsn)
		}
		db, err := sql.Open("sqlite", dsn)
		if err != nil {
			// Close any already-opened shards
			for j := uint(0); j < i; j++ {
				shards[j].Close()
			}
			return nil, fmt.Errorf("failed to open shard %d: %w", i, err)
		}
		db.SetMaxOpenConns(runtime.NumCPU() + 1)
		shards[i] = db
	}

	p := &Persistence{Config: config, Shards: shards, NumShards: numShards}
	if err := p.createSchema(); err != nil {
		p.Shutdown()
		return nil, err
	}
	if err := p.initIDGenerators(); err != nil {
		p.Shutdown()
		return nil, err
	}

	return p, nil
}

func buildDSN(config *PersistenceConfig, shardIdx, numShards uint) string {
	var dsn strings.Builder
	dsn.WriteString("file:")

	if numShards <= 1 {
		dsn.WriteString(filepath.Join(config.Path, config.Name))
	} else {
		ext := filepath.Ext(config.Name)
		base := strings.TrimSuffix(config.Name, ext)
		dsn.WriteString(filepath.Join(config.Path, fmt.Sprintf("%s_shard%d.db", base, shardIdx)))
	}

	var query bool

	if len(config.SQLitePragmas) > 0 {
		dsn.WriteRune('?')
		query = true
		for i, prag := range config.SQLitePragmas {
			if i > 0 {
				dsn.WriteRune('&')
			}
			dsn.WriteString(fmt.Sprintf("_pragma=%s", prag))
		}
	}

	if len(config.SQLiteOptions) > 0 {
		if !query {
			dsn.WriteRune('?')
		} else {
			dsn.WriteRune('&')
		}
		for i, opt := range config.SQLiteOptions {
			if i > 0 {
				dsn.WriteRune('&')
			}
			dsn.WriteString(opt)
		}
	}

	return dsn.String()
}

func (p *Persistence) createSchema() error {
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

	for _, db := range p.Shards {
		for _, stmt := range stmts {
			if _, err := db.Exec(stmt); err != nil {
				return fmt.Errorf("schema creation failed: %w\nStatement: %s", err, stmt)
			}
		}
	}
	return nil
}

func (p *Persistence) initIDGenerators() error {
	tables := []struct {
		table string
		gen   **IDGenerator
	}{
		{"units", &p.UnitIDs},
		{"instructions", &p.InstructionIDs},
		{"evaluations", &p.EvalIDs},
		{"mutations", &p.MutationIDs},
		{"tombstones", &p.TombstoneIDs},
		{"populations", &p.PopIDs},
	}
	for _, spec := range tables {
		globalMax := uint64(0)
		for _, db := range p.Shards {
			var maxID sql.NullInt64
			if err := db.QueryRow("SELECT MAX(id) FROM " + spec.table).Scan(&maxID); err != nil {
				return fmt.Errorf("failed to query max ID from %s: %w", spec.table, err)
			}
			if maxID.Valid && uint64(maxID.Int64) > globalMax {
				globalMax = uint64(maxID.Int64)
			}
		}
		*spec.gen = NewIDGenerator(globalMax)
	}
	return nil
}

func (p *Persistence) Shutdown() {
	for _, db := range p.Shards {
		db.Close()
	}
}

func (p *Persistence) Create(config *PopulationConfig) (*Population, error) {
	if config == nil {
		return nil, fmt.Errorf("PopulationConfig cannot be nil")
	}

	pop := NewPopulationFromConfig(config)
	pop.ID = p.PopIDs.Next()

	cols, vals := populationInsertValues(pop)
	query := "INSERT INTO populations (" + strings.Join(cols, ", ") + ") VALUES (" + placeholders(len(cols)) + ")"
	if _, err := p.shard0().Exec(query, vals...); err != nil {
		return nil, fmt.Errorf("failed to insert population: %w", err)
	}

	pop.persist = p
	return pop, nil
}

func (p *Persistence) LoadShallow(id uint) (*Population, error) {
	pop := &Population{}
	row := p.shard0().QueryRow(`SELECT id, current_generation,
		unit_count, synthesis_pool, carrying_capacity, elitism, max_offspring,
		unit_mutation_chance, unit_instruction_count, unit_ins_op_set_count, unit_lifespan,
		eval_machine_max_instruction_execution_count, eval_machine_memory_cell_count,
		eval_input_cell_count, eval_output_cell_count, eval_synthesis_input_cell_count,
		eval_input_cell_start, eval_input_cell_step, eval_eval_rounds,
		sel_machine_run, sel_set_fidelity, sel_sortedness,
		sel_set_fidelity_start, sel_set_fidelity_step, sel_sortedness_start, sel_sortedness_step,
		sel_instruction_count, sel_instructions_executed,
		fit_sortedness_priority, fit_set_fidelity_priority, fit_efficiency_priority
		FROM populations WHERE id = ?`, id)

	if err := scanPopulation(row, pop); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("Population id [%d] not found", id)
		}
		return nil, fmt.Errorf("Failed to find population [%d]: %w", id, err)
	}

	pop.persist = p
	return pop, nil
}

func (p *Persistence) SaveUnits(units []*Unit) error {
	// Pre-assign IDs so writeSharded can partition correctly by u.ID % NumShards
	for _, u := range units {
		if u.ID == 0 {
			u.ID = p.UnitIDs.Next()
		}
		for _, ins := range u.Instructions {
			if ins.ID == 0 {
				ins.ID = p.InstructionIDs.Next()
			}
			ins.UnitID = u.ID
			for _, mut := range ins.Mutations {
				if mut.ID == 0 {
					mut.ID = p.MutationIDs.Next()
				}
				mut.InstructionID = ins.ID
			}
		}
	}

	return p.writeSharded(units, func(tx *sql.Tx, batch []*Unit) error {
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
				for _, mut := range ins.Mutations {
					if _, err := tx.Exec(`INSERT INTO mutations (id, instruction_id, position1, position2, meta_op, op, chance)
						VALUES (?, ?, ?, ?, ?, ?, ?)`,
						mut.ID, mut.InstructionID, nullableUint(mut.Position1), nullableUint(mut.Position2),
						mut.MetaOP, mut.Op, mut.Chance); err != nil {
						return fmt.Errorf("failed to insert mutation: %w", err)
					}
				}
			}
		}
		return nil
	})
}

// PersistEvaluatedBatch does targeted persistence for units that have been
// evaluated: batch UPDATE alive/age on units, batch INSERT evaluations and
// tombstones. Does NOT touch instructions or mutations.
func (p *Persistence) PersistEvaluatedBatch(units []*Unit) error {
	return p.writeSharded(units, func(tx *sql.Tx, batch []*Unit) error {
		for _, u := range batch {
			if u.Alive == Dead {
				if _, err := tx.Exec("UPDATE units SET alive = ? WHERE id = ?", Dead, u.ID); err != nil {
					return err
				}
			} else {
				if _, err := tx.Exec("UPDATE units SET age = age + 1 WHERE id = ?", u.ID); err != nil {
					return err
				}
			}

			for _, e := range u.Evaluations {
				if e.ID == 0 {
					e.ID = p.EvalIDs.Next()
					var machineRun int
					if e.MachineRun {
						machineRun = 1
					}
					if _, err := tx.Exec(`INSERT INTO evaluations (id, unit_id, machine_run, set_fidelity, sortedness,
						instruction_count, instructions_executed, machine_error)
						VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
						e.ID, e.UnitID, machineRun, e.SetFidelity, e.Sortedness,
						e.InstructionCount, e.InstructionsExecuted,
						nullableString(e.MachineError)); err != nil {
						return err
					}
				}
			}

			if u.Tombstone != nil {
				t := u.Tombstone
				if t.ID == 0 {
					t.ID = p.TombstoneIDs.Next()
				}
				if _, err := tx.Exec("INSERT INTO tombstones (id, unit_id, reason) VALUES (?, ?, ?)",
					t.ID, t.UnitID, t.Reason); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// writeSharded partitions units by shard (u.ID % NumShards) and runs fn in
// parallel transactions, one per shard.
func (p *Persistence) writeSharded(units []*Unit, fn func(tx *sql.Tx, batch []*Unit) error) error {
	if p.NumShards == 1 {
		return withTx(p.Shards[0], func(tx *sql.Tx) error { return fn(tx, units) })
	}

	buckets := make([][]*Unit, p.NumShards)
	for _, u := range units {
		s := u.ID % p.NumShards
		buckets[s] = append(buckets[s], u)
	}

	errs := make([]error, p.NumShards)
	var wg sync.WaitGroup
	for i := uint(0); i < p.NumShards; i++ {
		if len(buckets[i]) == 0 {
			continue
		}
		wg.Add(1)
		go func(shard uint) {
			defer wg.Done()
			errs[shard] = withTx(p.Shards[shard], func(tx *sql.Tx) error {
				return fn(tx, buckets[shard])
			})
		}(i)
	}
	wg.Wait()
	return firstError(errs)
}

// writeShardedByID partitions unit IDs by shard and runs fn in parallel
// transactions, one per shard.
func (p *Persistence) writeShardedByID(ids []uint, fn func(tx *sql.Tx, ids []uint) error) error {
	if p.NumShards == 1 {
		return withTx(p.Shards[0], func(tx *sql.Tx) error { return fn(tx, ids) })
	}

	buckets := make([][]uint, p.NumShards)
	for _, id := range ids {
		s := id % p.NumShards
		buckets[s] = append(buckets[s], id)
	}

	errs := make([]error, p.NumShards)
	var wg sync.WaitGroup
	for i := uint(0); i < p.NumShards; i++ {
		if len(buckets[i]) == 0 {
			continue
		}
		wg.Add(1)
		go func(shard uint) {
			defer wg.Done()
			errs[shard] = withTx(p.Shards[shard], func(tx *sql.Tx) error {
				return fn(tx, buckets[shard])
			})
		}(i)
	}
	wg.Wait()
	return firstError(errs)
}

func firstError(errs []error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

type UnitPersistor func(*[]*Unit)

type UnitLoader func(id, total uint) <-chan []*Unit

// Helper: run fn inside a transaction with automatic rollback on error.
func withTx(db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// placeholders generates "?,?,?" for n parameters.
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}

// nullableUint converts *uint to a value suitable for sql driver (nil or int64).
func nullableUint(p *uint) interface{} {
	if p == nil {
		return nil
	}
	return int64(*p)
}

// nullableString converts *string to a value suitable for sql driver.
func nullableString(p *string) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

// bulkInsertUnits inserts units and their instructions using multi-row INSERT
// statements for dramatically better throughput. Batches rows to avoid hitting
// SQLite limits (max 500 variables per statement to stay safe).
func bulkInsertUnits(tx *sql.Tx, units []*Unit) error {
	// Bulk insert units in chunks
	const unitCols = 8 // id, population_id, parent_id, age, generation, lifespan, mutation_chance, alive
	const maxRowsPerStmt = 60 // 60 * 8 = 480 variables, under SQLite's limit

	for start := 0; start < len(units); start += maxRowsPerStmt {
		end := start + maxRowsPerStmt
		if end > len(units) {
			end = len(units)
		}
		chunk := units[start:end]

		var sb strings.Builder
		sb.WriteString("INSERT INTO units (id, population_id, parent_id, age, generation, lifespan, mutation_chance, alive) VALUES ")
		args := make([]interface{}, 0, len(chunk)*unitCols)
		for i, u := range chunk {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString("(?,?,?,?,?,?,?,?)")
			args = append(args, u.ID, u.PopulationID, nullableUint(u.ParentID), u.Age, u.Generation, u.Lifespan, u.MutationChance, u.Alive)
		}
		if _, err := tx.Exec(sb.String(), args...); err != nil {
			return fmt.Errorf("bulk insert units failed: %w", err)
		}
	}

	// Collect all instructions across all units, then bulk insert
	const insCols = 5 // id, unit_id, age, initial_op_set, ops
	const maxInsPerStmt = 100 // 100 * 5 = 500 variables

	var allIns []*Instruction
	for _, u := range units {
		allIns = append(allIns, u.Instructions...)
	}

	for start := 0; start < len(allIns); start += maxInsPerStmt {
		end := start + maxInsPerStmt
		if end > len(allIns) {
			end = len(allIns)
		}
		chunk := allIns[start:end]

		var sb strings.Builder
		sb.WriteString("INSERT INTO instructions (id, unit_id, age, initial_op_set, ops) VALUES ")
		args := make([]interface{}, 0, len(chunk)*insCols)
		for i, ins := range chunk {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString("(?,?,?,?,?)")
			args = append(args, ins.ID, ins.UnitID, ins.Age, ins.InitialOpSet, ins.Ops)
		}
		if _, err := tx.Exec(sb.String(), args...); err != nil {
			return fmt.Errorf("bulk insert instructions failed: %w", err)
		}
	}

	return nil
}

// populationInsertValues returns column names and values for INSERT.
func populationInsertValues(pop *Population) ([]string, []interface{}) {
	c := pop.PopulationConfig
	uc := c.UnitConfig
	ic := uc.InstructionConfig
	ec := c.EvaluatorConfig
	mc := ec.MachineConfig
	sc := c.SelectorConfig
	fc := c.FitnessConfig

	var selMachineRun int
	if sc.MachineRun {
		selMachineRun = 1
	}

	cols := []string{
		"id", "current_generation",
		"unit_count", "synthesis_pool", "carrying_capacity", "elitism", "max_offspring",
		"unit_mutation_chance", "unit_instruction_count", "unit_ins_op_set_count", "unit_lifespan",
		"eval_machine_max_instruction_execution_count", "eval_machine_memory_cell_count",
		"eval_input_cell_count", "eval_output_cell_count", "eval_synthesis_input_cell_count",
		"eval_input_cell_start", "eval_input_cell_step", "eval_eval_rounds",
		"sel_machine_run", "sel_set_fidelity", "sel_sortedness",
		"sel_set_fidelity_start", "sel_set_fidelity_step", "sel_sortedness_start", "sel_sortedness_step",
		"sel_instruction_count", "sel_instructions_executed",
		"fit_sortedness_priority", "fit_set_fidelity_priority", "fit_efficiency_priority",
	}
	vals := []interface{}{
		pop.ID, pop.CurrentGeneration,
		c.UnitCount, c.SynthesisPool, c.CarryingCapacity, c.Elitism, c.MaxOffspring,
		uc.MutationChance, uc.InstructionCount, ic.OpSetCount, uc.Lifespan,
		mc.MaxInstructionExecutionCount, mc.MemoryCellCount,
		ec.InputCellCount, ec.OutputCellCount, ec.SynthesisInputCellCount,
		ec.InputCellStart, ec.InputCellStep, ec.EvalRounds,
		selMachineRun, sc.SetFidelity, sc.Sortedness,
		sc.SetFidelityStart, sc.SetFidelityStep, sc.SortednessStart, sc.SortednessStep,
		sc.InstructionCount, sc.InstructionsExecuted,
		fc.SortednessPriority, fc.SetFidelityPriority, fc.EfficiencyPriority,
	}
	return cols, vals
}

// queryUnitsBatch returns up to `limit` alive units on this shard with id > afterID
// and id <= maxID, ordered by id. Used for cursor-paginated streaming.
func queryUnitsBatch(db *sql.DB, popID uint, afterID, maxID uint, limit int) ([]*Unit, error) {
	rows, err := db.Query(`SELECT id, population_id, parent_id, age, generation, lifespan, mutation_chance, alive
		FROM units WHERE population_id = ? AND alive = ? AND id > ? AND id <= ?
		ORDER BY id LIMIT ?`, popID, Alive, afterID, maxID, limit)
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

// queryInstructionsForUnits loads instructions for a specific set of unit IDs.
// Chunks the IN clause into groups of 900 to stay under SQLite's 999-variable limit.
func queryInstructionsForUnits(db *sql.DB, unitIDs []uint) ([]*Instruction, error) {
	if len(unitIDs) == 0 {
		return nil, nil
	}

	const chunkSize = 900
	var allInstructions []*Instruction

	for start := 0; start < len(unitIDs); start += chunkSize {
		end := start + chunkSize
		if end > len(unitIDs) {
			end = len(unitIDs)
		}
		chunk := unitIDs[start:end]

		args := make([]interface{}, len(chunk))
		for i, id := range chunk {
			args[i] = id
		}

		query := "SELECT id, unit_id, age, initial_op_set, ops FROM instructions WHERE unit_id IN (" + placeholders(len(chunk)) + ")"
		rows, err := db.Query(query, args...)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			ins := &Instruction{}
			if err := rows.Scan(&ins.ID, &ins.UnitID, &ins.Age, &ins.InitialOpSet, &ins.Ops); err != nil {
				rows.Close()
				return nil, err
			}
			allInstructions = append(allInstructions, ins)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}

	return allInstructions, nil
}

// queryMaxUnitID returns the maximum unit id on the given shard for the population
// (alive or dead). Returns 0 if no units exist.
func queryMaxUnitID(db *sql.DB, popID uint) (uint, error) {
	var maxID sql.NullInt64
	if err := db.QueryRow("SELECT MAX(id) FROM units WHERE population_id = ?", popID).Scan(&maxID); err != nil {
		return 0, err
	}
	if !maxID.Valid {
		return 0, nil
	}
	return uint(maxID.Int64), nil
}

// queryAliveUnitIDs returns just the IDs of alive units on the given shard for popID.
func queryAliveUnitIDs(db *sql.DB, popID uint) ([]uint, error) {
	rows, err := db.Query("SELECT id FROM units WHERE population_id = ? AND alive = ?", popID, Alive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uint
	for rows.Next() {
		var id uint
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// queryUnitParents returns a map of unitID → parentID for the given unit IDs on this shard.
// Units with NULL parent_id are omitted from the result. Chunks the IN clause to stay
// under SQLite's 999-variable limit.
func queryUnitParents(db *sql.DB, unitIDs []uint) (map[uint]uint, error) {
	if len(unitIDs) == 0 {
		return nil, nil
	}

	const chunkSize = 900
	result := make(map[uint]uint, len(unitIDs))

	for start := 0; start < len(unitIDs); start += chunkSize {
		end := start + chunkSize
		if end > len(unitIDs) {
			end = len(unitIDs)
		}
		chunk := unitIDs[start:end]

		args := make([]interface{}, len(chunk))
		for i, id := range chunk {
			args[i] = id
		}

		query := "SELECT id, parent_id FROM units WHERE id IN (" + placeholders(len(chunk)) + ") AND parent_id IS NOT NULL"
		rows, err := db.Query(query, args...)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var id uint
			var parentID uint
			if err := rows.Scan(&id, &parentID); err != nil {
				rows.Close()
				return nil, err
			}
			result[id] = parentID
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}

	return result, nil
}

// scanPopulation scans a row into a Population, reconstructing nested config structs.
func scanPopulation(row *sql.Row, pop *Population) error {
	var (
		selMachineRun                              int
		unitCount, synthesisPool, carryingCapacity uint
		elitism, maxOffspring                      uint
		machineMaxExec, machineCellCount           uint
	)
	uc := &UnitConfig{InstructionConfig: &InstructionConfig{}}
	ec := &EvaluatorConfig{}
	sc := &SelectorConfig{}
	fc := &FitnessConfig{}

	err := row.Scan(
		&pop.ID, &pop.CurrentGeneration,
		&unitCount, &synthesisPool, &carryingCapacity, &elitism, &maxOffspring,
		&uc.MutationChance, &uc.InstructionCount, &uc.InstructionConfig.OpSetCount, &uc.Lifespan,
		&machineMaxExec, &machineCellCount,
		&ec.InputCellCount, &ec.OutputCellCount, &ec.SynthesisInputCellCount,
		&ec.InputCellStart, &ec.InputCellStep, &ec.EvalRounds,
		&selMachineRun, &sc.SetFidelity, &sc.Sortedness,
		&sc.SetFidelityStart, &sc.SetFidelityStep, &sc.SortednessStart, &sc.SortednessStep,
		&sc.InstructionCount, &sc.InstructionsExecuted,
		&fc.SortednessPriority, &fc.SetFidelityPriority, &fc.EfficiencyPriority,
	)
	if err != nil {
		return err
	}

	sc.MachineRun = selMachineRun != 0

	ec.MachineConfig = &bf.MachineConfig{
		MaxInstructionExecutionCount: machineMaxExec,
		MemoryCellCount:             machineCellCount,
	}

	pop.PopulationConfig = &PopulationConfig{
		UnitCount:        unitCount,
		SynthesisPool:    synthesisPool,
		CarryingCapacity: carryingCapacity,
		Elitism:          elitism,
		MaxOffspring:     maxOffspring,
		UnitConfig:       uc,
		EvaluatorConfig:  ec,
		SelectorConfig:   sc,
		FitnessConfig:    fc,
	}

	return nil
}
