package integration

import (
	"crypto/rand"
	"sort"
	"testing"
	"time"

	"github.com/rawbytedev/memory3d/internal/types"
	"github.com/rawbytedev/memory3d/internal/vm"
)

// TestStressFragmentation runs a long, chaotic allocation/free pattern
// to measure fragmentation, allocation latency, and compaction effectiveness.
func TestStressFragmentation(t *testing.T) {
	// Configurations: with and without auto‑compaction
	configs := []struct {
		name          string
		enableCompact bool
	}{
		{"WithoutCompaction", false},
		{"WithCompaction", true},
	}

	for _, cfg := range configs {
		t.Run(cfg.name, func(t *testing.T) {
			vmConfig := vm.VMConfig{
				MemorySize:             100 * 1024, // 100 MB
				GasLimit:               100_000_000,
				EnableProof:            false,
				EnableCompaction:       false, // we control compaction separately
				EnableAutoCompaction:   cfg.enableCompact,
				CompactionInterval:     5 * time.Second,
				FragmentationThreshold: 0.25, // 25% triggers compaction
				MaxMovesPerCycle:       200,
				MaxInstructions:        1_000_000,
				LogLevel:               vm.LogLevelSilent,
			}

			vmInstance, err := vm.NewVM3D(vmConfig)
			if err != nil {
				t.Fatalf("failed to create VM: %v", err)
			}
			defer vmInstance.Shutdown()

			// Run the stress workload
			results := runStressWorkload(vmInstance, 50000, 30*time.Second)

			// Report metrics
			t.Logf("=== %s ===", cfg.name)
			t.Logf("Total allocations : %d", results.totalAllocs)
			t.Logf("Total frees       : %d", results.totalFrees)
			t.Logf("Out of memory     : %d", results.outmem)
			t.Logf("Y‑promotions      : %d", results.yPromotions)
			t.Logf("Final fragmentation: %.2f%%", results.finalFrag*100)
			t.Logf("Final memory usage: %.2f MB", float64(results.finalMem)/1024/1024)
			t.Logf("Alloc latency p50  : %.3f µs", results.latencyP50/1000)
			t.Logf("Alloc latency p95  : %.3f µs", results.latencyP95/1000)
			t.Logf("Alloc latency p99  : %.3f µs", results.latencyP99/1000)
			// Basic sanity: fragmentation should be lower with compaction
			if cfg.enableCompact {
				// We don't assert a strict number because it depends on randomness,
				// but we log a warning if compaction didn't help.
				if results.finalFrag > 0.40 {
					t.Logf("⚠️  Fragmentation still high (%.2f%%) – try increasing workload duration.", results.finalFrag*100)
				}
			}
		})
	}
}

// stressResults holds the collected metrics.
type stressResults struct {
	totalAllocs    int
	totalFrees     int
	yPromotions    uint64
	finalFrag      float64
	finalMem       uint64
	latencyP50     float64 // nanoseconds
	latencyP95     float64
	latencyP99     float64
	latencySamples []float64
	outmem         int
}

// runStressWorkload performs random allocations and frees, measuring everything.
func runStressWorkload(vmInstance *vm.VM3D, maxOps int, maxDuration time.Duration) stressResults {
	var (
		alive       []types.Address3D // currently allocated addresses
		allocs      int
		frees       int
		latencies   []float64
		startTime   = time.Now()
		fragSamples []float64
		outmem      int
	)
	outmem = 0
	// Helper: generate random size between 64B and 64KB
	randomSize := func() uint32 {
		r := randInt(0, 100)
		if r < 10 {
			return uint32(640 + randInt(0, 4480)) // small (<512 B)
		} else if r < 40 {
			return uint32(20480 + randInt(0, 61440)) // medium (2‑8 KB)
		} else {
			return uint32(102400 + randInt(0, 200400)) // large (10‑20 KB) – always multi‑plane
		}
	}

	// Helper: choose a random element to free (if any)
	randomAlive := func() types.Address3D {
		if len(alive) == 0 {
			return types.Address3D{}
		}
		idx := randInt(0, len(alive)-1)
		addr := alive[idx]
		// Remove it from the slice (preserve order not important)
		alive[idx] = alive[len(alive)-1]
		alive = alive[:len(alive)-1]
		return addr
	}

	// Main loop
	ops := 0
	for ops < maxOps && time.Since(startTime) < maxDuration {
		// 80% chance to allocate, 20% chance to free (if we have anything alive)
		if len(alive) == 0 || randInt(0, 100) < 80 {
			var size uint32
			if ops%5 == 0 {
				size = uint32(100000 + randInt(0, 100000)) // definitely > plane
			} else {
				size = randomSize()
			}
			start := time.Now()
			addr, err := vmInstance.AllocateMemory(size, types.RegionTypeHeap)
			if err != nil {
				// Out of memory? Try to free something first.
				if len(alive) > 0 {
					_ = randomAlive()
					outmem++
				}
				continue
			}
			lat := float64(time.Since(start).Nanoseconds())

			latencies = append(latencies, lat)

			// Write some dummy data (just to make it "used")
			data := make([]byte, size)
			rand.Read(data)
			_ = vmInstance.Store3D(addr, data)

			alive = append(alive, addr)
			allocs++
		} else {
			if len(alive) > 0 {
				addr := randomAlive()
				_ = vmInstance.Free3D(addr)
				frees++
			}
		}

		// Every 100 ops, sample fragmentation
		if ops%100 == 0 {
			report := vmInstance.GetCompactionReport()
			avgFrag := 0.0
			count := 0
			for _, r := range report {
				avgFrag += r.Fragmentation
				count++
			}
			if count > 0 {
				//avgFrag /= float64(count)
				fragSamples = append(fragSamples, avgFrag)
			}
		}

		ops++
	}

	// Final metrics
	stats := vmInstance.GetStats()
	report := vmInstance.GetCompactionReport()
	avgFrag := 0.0
	count := 0
	for _, r := range report {
		avgFrag += r.Fragmentation
		count++
	}
	if count > 0 {
		avgFrag /= float64(count)
	}

	// Compute latency percentiles
	sort.Float64s(latencies)
	p50 := percentile(latencies, 0.50)
	p95 := percentile(latencies, 0.95)
	p99 := percentile(latencies, 0.99)

	return stressResults{
		totalAllocs:    allocs,
		totalFrees:     frees,
		yPromotions:    stats.YPromotions,
		finalFrag:      avgFrag,
		finalMem:       stats.MemoryUsage,
		latencyP50:     p50,
		latencyP95:     p95,
		latencyP99:     p99,
		latencySamples: latencies,
		outmem:         outmem,
	}
}

// Helper: random integer in [min, max]
func randInt(min, max int) int {
	if min > max {
		min, max = max, min
	}
	b := make([]byte, 4)
	rand.Read(b)
	v := int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	if v < 0 {
		v = -v
	}
	return min + v%(max-min+1)
}

// Helper: percentile of sorted float64 slice
func percentile(vals []float64, p float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	idx := int(float64(len(vals)-1) * p)
	return vals[idx]
}
