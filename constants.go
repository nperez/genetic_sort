package genetic_sort

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// pooledRand uses sync.Pool to give each goroutine its own *rand.Rand,
// eliminating mutex contention in parallel workloads.
type pooledRand struct {
	pool sync.Pool
}

func newPooledRand(seed int64) *pooledRand {
	var counter int64
	return &pooledRand{
		pool: sync.Pool{
			New: func() any {
				s := atomic.AddInt64(&counter, 1) - 1
				return rand.New(rand.NewSource(seed + s))
			},
		},
	}
}

func (pr *pooledRand) Intn(n int) int {
	r := pr.pool.Get().(*rand.Rand)
	v := r.Intn(n)
	pr.pool.Put(r)
	return v
}

func (pr *pooledRand) Float32() float32 {
	r := pr.pool.Get().(*rand.Rand)
	v := r.Float32()
	pr.pool.Put(r)
	return v
}

// rng is the package-level random source. Uses sync.Pool internally
// so concurrent goroutines each get their own *rand.Rand — no contention.
var rng *pooledRand = newPooledRand(time.Now().UnixNano())

// InitRNG seeds the package-level rng. If seed is 0, the current
// time is used (non-deterministic). A non-zero seed gives
// reproducible results.
func InitRNG(seed int64) {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	rng = newPooledRand(seed)
}

const (
	DEBUG                                       = false
	Alive                                       = 1
	Dead                                        = 2
	FailedMachineRun           SelectFailReason = 1
	FailedSetFidelity          SelectFailReason = 2
	FailedSortedness           SelectFailReason = 3
	FailedInstructionCount     SelectFailReason = 4
	FailedInstructionsExecuted SelectFailReason = 5
	FailedLifespan             SelectFailReason = 6
	FailedCompetition          SelectFailReason = 7
	PUSH_OP                    byte             = iota
	POP_OP
	SHIFT_OP
	UNSHIFT_OP
	INSERT_OP
	DELETE_OP
	SWAP_OP
	REPLACE_OP
	META_NO_OP
)
