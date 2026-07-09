package integration

import (
	"testing"

	"github.com/rawbytedev/memory3d/internal/allocator"
	"github.com/rawbytedev/memory3d/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYPromotionExplicit(t *testing.T) {
	cfg := allocator.Config{
		PlaneSize:        1024, // 1 KB per plane
		MaxYPlanes:       8,
		MaxYPromotions:   4,
		WorkerCount:      1,
		EnableConcurrent: false,
		EnablePooling:    false,
	}
	alloc := allocator.NewAllocator3D(cfg)
	defer alloc.Shutdown()

	const smallSize = 128
	var smallAllocs []*types.Allocation

	// 1. Fill the first plane (8×128 = 1024 bytes).
	for i := 0; i < 8; i++ {
		al, err := alloc.Allocate(smallSize, types.RegionTypeHeap, 0)
		require.NoError(t, err)
		smallAllocs = append(smallAllocs, al)
	}

	// Get the region that was created.
	regionID := smallAllocs[0].Address.X
	region := alloc.GetRegion(regionID)
	require.NotNil(t, region)

	// 2. Manually add a second plane to the same region.
	region.Lock()
	plane := types.NewPlane(1, regionID, cfg.PlaneSize)
	region.AddPlane(plane)
	region.Unlock()

	// 3. Fill the second plane.
	for i := 0; i < 8; i++ {
		al, err := alloc.Allocate(smallSize, types.RegionTypeHeap, 0)
		require.NoError(t, err)
		smallAllocs = append(smallAllocs, al)
	}

	// 4. Free ALL small allocations – both planes become completely empty.
	for _, al := range smallAllocs {
		err := alloc.Free(al)
		require.NoError(t, err)
	}

	// 5. Verify both planes exist and are fully free.
	region = alloc.GetRegion(regionID)
	require.NotNil(t, region)
	region.RLock()
	freeBytes := region.FreeBytes
	region.RUnlock()
	assert.Equal(t, uint64(2048), freeBytes, "Region should have 2048 free bytes")

	plane0 := region.GetPlane(0)
	require.NotNil(t, plane0)
	assert.Equal(t, uint16(1024), plane0.FreeBytes())

	plane1 := region.GetPlane(1)
	require.NotNil(t, plane1)
	assert.Equal(t, uint16(1024), plane1.FreeBytes())

	// 6. Allocate 2048 bytes – this requires two planes, so Y‑promotion is forced.
	largeSize := uint32(2048)
	largeAlloc, err := alloc.Allocate(largeSize, types.RegionTypeHeap, 0)
	require.NoError(t, err, "Y‑promotion allocation should succeed")
	require.NotNil(t, largeAlloc)

	// 7. Verify it is Y‑promoted.
	assert.True(t, largeAlloc.IsYPromoted(), "Allocation should be Y‑promoted")
	assert.GreaterOrEqual(t, len(largeAlloc.Fragments), 2,
		"Should use at least 2 planes")

	// 8. Verify total size matches.
	total := uint32(0)
	for _, frag := range largeAlloc.Fragments {
		total += uint32(frag.Size)
	}
	assert.Equal(t, largeSize, total, "Fragment sizes sum to requested size")

	t.Logf("Y‑promotion succeeded with %d fragments", len(largeAlloc.Fragments))
}
