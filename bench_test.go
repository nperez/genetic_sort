package genetic_sort

import (
	"runtime"
	"sync"
	"testing"

	bf "nickandperla.net/brainfuck"
)

// BenchmarkParallelEval measures actual CPU core utilization during parallel
// evaluation. Run with: go test -run=^$ -bench=BenchmarkParallelEval -benchtime=1x -v
// Then compare user vs real time from `time go test ...`
func BenchmarkParallelEval(b *testing.B) {
	rng = newPooledRand(42)

	config := &UnitConfig{
		MutationChance:   0.25,
		InstructionCount: 10,
		InstructionConfig: &InstructionConfig{
			OpSetCount: 10,
		},
		Lifespan: 200,
	}

	evalConfig := &EvaluatorConfig{
		InputCellCount:  2,
		OutputCellCount: 2,
		MachineConfig: &bf.MachineConfig{
			MaxInstructionExecutionCount: 10000,
			MemoryCellCount:             30,
		},
	}

	// Create 100k units
	n := 100000
	units := make([]*Unit, n)
	for i := 0; i < n; i++ {
		units[i] = NewUnitFromConfig(config)
	}

	cpus := runtime.NumCPU()
	b.Logf("Units: %d, CPUs: %d, GOMAXPROCS: %d", n, cpus, runtime.GOMAXPROCS(0))

	b.ResetTimer()

	for iter := 0; iter < b.N; iter++ {
		// Parallel eval
		var wg sync.WaitGroup
		chunkSize := n / cpus
		for i := 0; i < cpus; i++ {
			start := i * chunkSize
			end := start + chunkSize
			if i == cpus-1 {
				end = n
			}
			wg.Add(1)
			go func(chunk []*Unit) {
				defer wg.Done()
				evaluator := NewEvaluator(evalConfig)
				for _, unit := range chunk {
					unit.Evaluations = nil
					evaluator.EvaluateWithCellCounts(unit, 2, 2)
				}
			}(units[start:end])
		}
		wg.Wait()

		// Parallel mitosis
		type ch struct{ units []*Unit }
		chunks := make([]ch, cpus)
		for i := 0; i < cpus; i++ {
			start := i * chunkSize
			end := start + chunkSize
			if i == cpus-1 {
				end = n
			}
			wg.Add(1)
			go func(idx int, chunk []*Unit) {
				defer wg.Done()
				var local []*Unit
				for _, unit := range chunk {
					local = append(local, unit.Mitosis(nil, nil))
				}
				chunks[idx].units = local
			}(i, units[start:end])
		}
		wg.Wait()
	}
}
