package vm

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/rawbytedev/memory3d/internal/types"
)

var ErrInsufficientGas = errors.New("Insufficient Gas")

type GasAccountant3D struct {
	// Per-thread gas buckets to reduce contention
	buckets   []*GasBucket
	bucketIdx uint64 // atomic counter for bucket selection
	totalGas  uint64 // total gas limit
	mu        sync.RWMutex
}

func (g *GasAccountant3D) Remaining() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	total := int64(0)
	for _, bucket := range g.buckets {
		// Load remaining gas atomically
		total += int64(atomic.LoadUint64(&bucket.remaining))
	}

	if total < 0 {
		return 0
	}
	return int(total)
}

type GasBucket struct {
	remaining     uint64
	used          uint64
	promotions    uint32
	regionChanges uint32
}

func NewAccountant(GasLimit uint64) *GasAccountant3D {
	// Create buckets for concurrent access (default 1 for deterministic behavior)
	// Keep design simple to avoid cross-bucket ordering issues for refunds.
	numBuckets := 1
	buckets := make([]*GasBucket, numBuckets)

	// Distribute gas limit across buckets
	gasPerBucket := GasLimit / uint64(numBuckets)
	remainder := GasLimit % uint64(numBuckets)

	for i := 0; i < numBuckets; i++ {
		gas := gasPerBucket
		if i == 0 {
			gas += remainder // Give remainder to first bucket
		}
		buckets[i] = &GasBucket{
			remaining: gas,
			used:      0,
		}
	}

	return &GasAccountant3D{
		buckets:   buckets,
		bucketIdx: 0,
		totalGas:  GasLimit,
	}
}

func (g *GasAccountant3D) Consume(gas uint64) error {
	bucket := g.getBucket()

	for {
		old := atomic.LoadUint64(&bucket.remaining)
		if old < gas {
			return ErrInsufficientGas
		}
		if atomic.CompareAndSwapUint64(&bucket.remaining, old, old-gas) {
			atomic.AddUint64(&bucket.used, gas)
			return nil
		}
	}
}

func (g *GasAccountant3D) getBucket() *GasBucket {
	// Simple round-robin bucket selection with atomic counter
	// Can be improved with goroutine-local storage in future
	idx := atomic.AddUint64(&g.bucketIdx, 1)
	bucketIdx := idx % uint64(len(g.buckets))
	return g.buckets[bucketIdx]
}

// Vectorized gas calculation using SIMD-like patterns
func (g *GasAccountant3D) calculateGas(addrFrom, addrTo types.Address3D, size uint32) uint64 {
	// Base cost
	gas := uint64(3 + size)

	// Parallel distance calculation
	distances := make([]uint64, 3)
	var wg sync.WaitGroup
	wg.Add(3)

	go func() { distances[0] = absDiff64(addrFrom.X, addrTo.X) * 100; wg.Done() }()
	go func() { distances[1] = uint64(absDiff32(addrFrom.Y, addrTo.Y) * 10); wg.Done() }()
	go func() { distances[2] = uint64(absDiff16(addrFrom.Z, addrTo.Z)); wg.Done() }()

	wg.Wait()

	for _, d := range distances {
		gas += d
	}

	return gas
}

func absDiff64(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

// Refund returns gas to a bucket
func (g *GasAccountant3D) Refund(gas uint64) {
	// Try to find a bucket that has recorded used gas to reduce
	for _, bucket := range g.buckets {
		oldUsed := atomic.LoadUint64(&bucket.used)
		if oldUsed == 0 {
			continue
		}
		// Attempt to subtract from this bucket's used amount
		for {
			cur := atomic.LoadUint64(&bucket.used)
			var newUsed uint64
			if cur <= gas {
				newUsed = 0
			} else {
				newUsed = cur - gas
			}
			if atomic.CompareAndSwapUint64(&bucket.used, cur, newUsed) {
				atomic.AddUint64(&bucket.remaining, gas)
				return
			}
		}
	}

	// If no used bucket found, refund to first bucket
	atomic.AddUint64(&g.buckets[0].remaining, gas)
}

// Used returns total gas used across all buckets
func (g *GasAccountant3D) Used() uint64 {
	var total uint64
	for _, bucket := range g.buckets {
		total += atomic.LoadUint64(&bucket.used)
	}
	return total
}
func absDiff32(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}

func absDiff16(a, b uint16) uint16 {
	if a > b {
		return a - b
	}
	return b - a
}
