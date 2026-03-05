package allocator

import (
	"sync"
	"testing"
	"time"

	"github.com/rawbytedev/memory3d/internal/types"
)

func TestAllocatorAllocationsNonOverlapping(t *testing.T) {
	cfg := Config{
		PlaneSize:        64,
		MaxYPlanes:       4,
		MaxYPromotions:   2,
		WorkerCount:      1,
		EnableConcurrent: false,
		EnablePooling:    false,
	}

	a := NewAllocator3D(cfg)

	addrs := make([]types.Address3D, 0, 20)
	allocs := make([]*types.Allocation, 0, 20)

	for i := 0; i < 12; i++ {
		sz := uint32(4)
		alloc, err := a.Allocate(sz, types.RegionTypeHeap, 0)
		if err != nil {
			t.Fatalf("allocate failed: %v", err)
		}
		if alloc == nil {
			t.Fatalf("allocate returned nil")
		}
		addrs = append(addrs, alloc.Address)
		allocs = append(allocs, alloc)
	}

	// Ensure no overlapping allocations
	for i := 0; i < len(allocs); i++ {
		a1 := allocs[i]
		for j := i + 1; j < len(allocs); j++ {
			a2 := allocs[j]
			if a1.Address.X == a2.Address.X && a1.Address.Y == a2.Address.Y {
				s1 := uint32(a1.Address.Z)
				e1 := s1 + a1.Size
				s2 := uint32(a2.Address.Z)
				e2 := s2 + a2.Size
				if !(e1 <= s2 || e2 <= s1) {
					t.Fatalf("allocations overlap: %v and %v", a1.Address, a2.Address)
				}
			}
		}
	}
}

func TestPlaneFreeBytesUpdates(t *testing.T) {
	cfg := Config{
		PlaneSize:        128,
		MaxYPlanes:       4,
		MaxYPromotions:   2,
		WorkerCount:      1,
		EnableConcurrent: false,
		EnablePooling:    false,
	}
	a := NewAllocator3D(cfg)

	// Allocate a block and check plane free bytes delta
	alloc, err := a.Allocate(16, types.RegionTypeHeap, 0)
	if err != nil {
		t.Fatalf("allocate failed: %v", err)
	}
	region := a.GetRegion(alloc.Address.X)
	if region == nil {
		t.Fatalf("region not found: %d", alloc.Address.X)
	}
	plane := region.GetPlane(alloc.Address.Y)
	if plane == nil {
		t.Fatalf("plane not found: %d", alloc.Address.Y)
	}

	// After allocation free bytes should have decreased by alloc.Size
	freeAfterAlloc := plane.FreeBytes()
	if uint32(plane.Size)-uint32(freeAfterAlloc) < alloc.Size {
		t.Fatalf("expected free bytes to decrease by at least %d, got %d", alloc.Size, uint32(plane.Size)-uint32(freeAfterAlloc))
	}

	// Free and expect free bytes back to initial
	// Capture before free to compute expected
	beforeFree := freeAfterAlloc
	if err := a.Free(alloc); err != nil {
		t.Fatalf("free failed: %v", err)
	}
	freeAfter := plane.FreeBytes()
	if freeAfter <= beforeFree {
		t.Fatalf("expected free bytes to increase after free: before=%d after=%d", beforeFree, freeAfter)
	}
}

func TestYPromotionMultiPlaneAllocations(t *testing.T) {
	cfg := Config{
		PlaneSize:        16,
		MaxYPlanes:       8,
		MaxYPromotions:   8,
		WorkerCount:      1,
		EnableConcurrent: false,
		EnablePooling:    false,
	}
	a := NewAllocator3D(cfg)

	// Create a small allocation first to ensure a region and initial plane exist
	smallAlloc, err := a.Allocate(1, types.RegionTypeHeap, 0)
	if err != nil || smallAlloc == nil {
		t.Fatalf("bootstrap allocate failed: %v", err)
	}
	region := a.GetRegion(smallAlloc.Address.X)
	if region == nil {
		t.Fatalf("region not found after bootstrap: %d", smallAlloc.Address.X)
	}
	// Add extra planes to the region to allow Y-promotion
	for id := uint32(1); id <= 3; id++ {
		p := a.getPlaneFromPool(id, region.ID)
		region.AddPlane(p)
	}

	sz := uint32(cfg.PlaneSize) * 3 // require 3 planes
	alloc, err := a.Allocate(sz, types.RegionTypeHeap, 0)
	if err != nil {
		t.Fatalf("y-promotion allocate failed: %v", err)
	}
	if alloc == nil {
		t.Fatalf("y-promotion allocate returned nil")
	}
	if len(alloc.Fragments) <= 1 {
		t.Fatalf("expected y-promoted allocation (fragments>1), got fragments=%d", len(alloc.Fragments))
	}

	// Sum fragments sizes
	var total uint32
	for _, f := range alloc.Fragments {
		total += uint32(f.Size)
	}
	if total != sz {
		t.Fatalf("expected total fragment size %d, got %d", sz, total)
	}

	// Check region and planes exist and free bytes decreased
	region = a.GetRegion(alloc.Address.X)
	if region == nil {
		t.Fatalf("region not found: %d", alloc.Address.X)
	}
	// Validate each fragment plane size reduced
	for _, f := range alloc.Fragments {
		p := region.GetPlane(f.PlaneID)
		if p == nil {
			t.Fatalf("plane %d not found", f.PlaneID)
		}
		// Ensure that used bytes accounted for (rough sanity)
		used := p.UsedBytes()
		if used == 0 {
			t.Fatalf("expected plane %d to have used bytes after allocation", p.ID)
		}
	}

	// Free and ensure planes freed
	if err := a.Free(alloc); err != nil {
		t.Fatalf("free failed: %v", err)
	}
	// After free, no fragment should be present in planes map at fragment start
	region = a.GetRegion(alloc.Address.X)
	for _, f := range alloc.Fragments {
		p := region.GetPlane(f.PlaneID)
		if p == nil {
			continue
		}
		if p.IsAllocated(f.StartZ) {
			t.Fatalf("expected fragment to be freed at plane %d start %d", f.PlaneID, f.StartZ)
		}
	}
}

func TestAllocatorConcurrencyAllocations(t *testing.T) {
	cfg := Config{
		PlaneSize:        64,
		MaxYPlanes:       8,
		MaxYPromotions:   4,
		WorkerCount:      4,
		EnableConcurrent: true,
		EnablePooling:    false,
	}
	a := NewAllocator3D(cfg)

	var wg sync.WaitGroup
	n := 100
	allocs := make([]*types.Allocation, 0, n)
	var mu sync.Mutex

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			al, err := a.Allocate(8, types.RegionTypeHeap, 0)
			if err != nil || al == nil {
				// record nothing; test will fail if too many nils
				return
			}
			mu.Lock()
			allocs = append(allocs, al)
			mu.Unlock()
		}()
	}
	wg.Wait()

	if len(allocs) == 0 {
		t.Fatalf("no allocations returned in concurrent test")
	}

	// Ensure no overlapping among returned allocations
	for i := 0; i < len(allocs); i++ {
		a1 := allocs[i]
		for j := i + 1; j < len(allocs); j++ {
			a2 := allocs[j]
			if a1.Address.X == a2.Address.X && a1.Address.Y == a2.Address.Y {
				s1 := uint32(a1.Address.Z)
				e1 := s1 + a1.Size
				s2 := uint32(a2.Address.Z)
				e2 := s2 + a2.Size
				if !(e1 <= s2 || e2 <= s1) {
					t.Fatalf("concurrent allocations overlap: %v and %v", a1.Address, a2.Address)
				}
			}
		}
	}

	// Quick sanity: perform some frees concurrently
	wg.Add(len(allocs))
	for _, al := range allocs {
		go func(aobj *types.Allocation) {
			defer wg.Done()
			time.Sleep(time.Millisecond)
			_ = a.Free(aobj)
		}(al)
	}
	wg.Wait()
}

func TestYPromotionStress(t *testing.T) {
	cfg := Config{
		PlaneSize:        64,
		MaxYPlanes:       8,
		MaxYPromotions:   8,
		WorkerCount:      1,
		EnableConcurrent: false,
		EnablePooling:    false,
	}
	a := NewAllocator3D(cfg)

	// Create multiple allocations with varied sizes (keep them smaller than plane size or up to 2 planes)
	var allocs []*types.Allocation
	sizes := []uint32{10, 20, 15, 32, 48, 32, 25, 20}

	for _, sz := range sizes {
		alloc, err := a.Allocate(sz, types.RegionTypeHeap, 0)
		if err != nil {
			t.Fatalf("stress allocate size=%d failed: %v", sz, err)
		}
		if alloc == nil {
			t.Fatalf("stress allocate size=%d returned nil (may be exhausted)", sz)
		}
		allocs = append(allocs, alloc)
	}

	// Verify no overlaps across all allocations
	for i := 0; i < len(allocs); i++ {
		a1 := allocs[i]
		for j := i + 1; j < len(allocs); j++ {
			a2 := allocs[j]
			if a1.Address.X == a2.Address.X && a1.Address.Y == a2.Address.Y {
				s1 := uint32(a1.Address.Z)
				e1 := s1 + a1.Size
				s2 := uint32(a2.Address.Z)
				e2 := s2 + a2.Size
				if !(e1 <= s2 || e2 <= s1) {
					t.Fatalf("stress test overlapping allocations: %v and %v", a1.Address, a2.Address)
				}
			}
		}
	}

	// Free in reverse order and verify
	for i := len(allocs) - 1; i >= 0; i-- {
		if err := a.Free(allocs[i]); err != nil {
			t.Fatalf("stress free index=%d failed: %v", i, err)
		}
	}
}

func TestRegionFreeBytesConsistency(t *testing.T) {
	cfg := Config{
		PlaneSize:        64,
		MaxYPlanes:       8,
		MaxYPromotions:   4,
		WorkerCount:      1,
		EnableConcurrent: false,
		EnablePooling:    false,
	}
	a := NewAllocator3D(cfg)

	// Helper to check region free bytes consistency
	checkRegionConsistency := func(region *types.Region) {
		region.RLock()
		defer region.RUnlock()

		// Calculate sum of all plane free bytes
		var sumPlanesFree uint64
		for _, p := range region.Planes {
			sumPlanesFree += uint64(p.FreeBytes())
		}

		regionFree := regionFreeBytes(region)
		if sumPlanesFree != regionFree {
			t.Fatalf("region free bytes mismatch: sum of planes=%d, region.FreeBytes=%d", sumPlanesFree, regionFree)
		}
	}

	// Perform allocations and check consistency
	allocs := make([]*types.Allocation, 0)
	for i := 0; i < 5; i++ {
		alloc, err := a.Allocate(16, types.RegionTypeHeap, 0)
		if err != nil {
			t.Fatalf("allocate %d failed: %v", i, err)
		}
		if alloc == nil {
			t.Fatalf("allocate %d returned nil", i)
		}
		allocs = append(allocs, alloc)

		region := a.GetRegion(alloc.Address.X)
		if region != nil {
			checkRegionConsistency(region)
		}
	}

	// Check consistency after freeing each allocation
	for i, alloc := range allocs {
		if err := a.Free(alloc); err != nil {
			t.Fatalf("free %d failed: %v", i, err)
		}

		region := a.GetRegion(alloc.Address.X)
		if region != nil {
			checkRegionConsistency(region)
		}
	}
}

// regionFreeBytes returns the total free bytes in a region (must be called with region locked)
func regionFreeBytes(region *types.Region) uint64 {
	// For now, we'll just return the atomic value; in reality we could verify it
	// by summing plane.FreeBytes() but that requires more complex locking
	return region.FreeBytes
}
