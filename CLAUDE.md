# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
# Build the library
go build ./...

# Run all tests (library + brainfuck submodule)
go test ./... && go test ./brainfuck/...

# Run a single test
go test -run TestFunctionName ./...

# Run brainfuck submodule tests only
go test ./brainfuck/...

# Build CLI tools
go build ./cmd/addpop/
go build ./cmd/rungen/

# Run addpop (creates a population and synthesizes initial units)
./addpop -config ./config.toml -popconfig ./pop.toml

# Run rungen (processes one generation for an existing population)
./rungen -config ./config.toml -popid 1
```

## Architecture

This is a genetic algorithm that evolves Brainfuck programs to sort arrays of uint8 values.

### Core Loop

1. **Synthesize** (`Population.SynthesizeUnits`): Generate random Units, evaluate them, keep only those passing selection criteria
2. **Evaluate** (`Evaluator.Evaluate`): Run a Unit's compiled Brainfuck program against random input, measuring set fidelity (how many input values appear in output) and sortedness (inversions via merge sort)
3. **Select** (`Selector.Select`): Kill units that fail thresholds for machine_run, set_fidelity, sortedness, instruction_count, or instructions_executed
4. **Reproduce** (`Unit.Mitosis`): Surviving units clone themselves; each instruction has a mutation chance per generation
5. **Persist**: All state stored in SQLite via GORM

### Key Types

- **Unit**: An organism containing `[]*Instruction`. Has lifespan, mutation chance, and alive/dead status. Death is recorded via `Tombstone`.
- **Instruction**: Wraps a Brainfuck program fragment. Ops are stored in a compressed 4-bit-per-symbol binary format (`makeOpsSmall`/`makeOpsBig`). The `Instructions` type alias (`[]*Instruction`) has a `ToProgram()` method that concatenates all instruction ops into a single Brainfuck program string.
- **Mutation**: Meta-operations (push, pop, shift, insert, delete, swap, replace) that modify an Instruction's ops byte slice.
- **Evaluator**: Creates a `brainfuck.Machine`, loads the unit's compiled program and random input, runs it, then scores the output.
- **Selector**: Compares `Evaluation` metrics against `SelectorConfig` thresholds. Returns a `SelectFailReason` (0 = survived).
- **Processor**: Pulls batches of units from a `UnitLoader` channel, evaluates, selects, and persists results.
- **GenerationEngine**: Runs multiple Processors in parallel goroutines (one per CPU).
- **Persistence**: GORM + SQLite. Handles schema migration, batch loading/saving, and provides `UnitLoader`/`UnitPersistor` abstractions. Tables: populations, units, instructions, mutations, evaluations, tombstones.

### brainfuck/ Submodule

A local Go module (`replace` directive in go.mod) implementing an extended Brainfuck interpreter:
- Standard ops: `< > + - [ ]`
- Extensions: `*` (bookmark current memory pointer) and `^` (jump to bookmarked pointer, swapping current and bookmark)
- `#` is a debug no-op
- `PREFAB_OPSETS` in `op.go` define starter instruction fragments (swap, find-zero, move-to-zero) used to seed initial random instructions

### Configuration

Two TOML config files:
- **config.toml** (`ToolConfig`): Database path, SQLite pragmas, batch size
- **pop.toml** (`PopulationConfig`): Unit count, lifespan, mutation chance, instruction count, evaluator/machine settings, selection thresholds

### Constants

`constants.go` holds the `DEBUG` flag (compile-time toggle), alive/dead status values, `SelectFailReason` enum, and mutation meta-op byte constants. `DEBUG` is currently a `const bool` — change it to `true` to enable verbose logging.
