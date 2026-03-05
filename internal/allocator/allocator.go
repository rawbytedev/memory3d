package allocator

import (
	"container/list"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rawbytedev/memory3d/internal/types"
)

// Config configures the 3D allocator behavior
type Config struct {
	PlaneSize        uint16 `json:"plane_size"`
	MaxYPlanes       uint32 `json:"max_y_planes"`
	MaxYPromotions   uint8  `json:"max_y_promotions"`
	WorkerCount      int    `json:"worker_count"`
	EnableConcurrent bool   `json:"enable_concurrent"`
	EnablePooling    bool   `json:"enable_pooling"`
}

// Stats tracks allocator statistics
type Stats struct {
	Allocations   uint64  `json:"allocations"`
	Frees         uint64  `json:"frees"`
	YPromotions   uint64  `json:"y_promotions"`
	Fragmentation float64 `json:"fragmentation"`
	CacheHits     uint64  `json:"cache_hits"`
	CacheMisses   uint64  `json:"cache_misses"`
}

// AllocRequest represents an allocation request for concurrent workers
type AllocRequest struct {
	Size       uint32
	RegionType types.RegionType
	Flags      types.AllocationFlags
	Result     chan AllocResult
}

// FreeRequest represents a free request for concurrent workers
type FreeRequest struct {
	Address types.Address3D
	Result  chan error
}

// AllocResult contains the result of an allocation request
type AllocResult struct {
	Address    types.Address3D
	Allocation *types.Allocation
	Error      error
}

// Allocator3D manages 3D memory allocation across regions, planes, and offsets
type Allocator3D struct {
	regions map[uint64]*types.Region
	config  Config
	stats   Stats

	// Pools
	planePool sync.Pool

	// Concurrency
	regionLRU   *list.List
	lruMutex    sync.RWMutex
	workers     chan *AllocRequest
	freeWorkers chan *FreeRequest

	// Tracking
	nextRegionID uint64
	nextAllocID  uint64
	mu           sync.RWMutex
}

// GetRegion returns the region with the given ID
func (a *Allocator3D) GetRegion(x uint64) *types.Region {
	return a.regions[x]
}

// NewAllocator3D creates and initializes a new 3D allocator
func NewAllocator3D(config Config) *Allocator3D {
	alloc := &Allocator3D{
		regions:      make(map[uint64]*types.Region),
		config:       config,
		regionLRU:    list.New(),
		workers:      make(chan *AllocRequest, 1000),
		freeWorkers:  make(chan *FreeRequest, 1000),
		nextRegionID: 1,
		nextAllocID:  1,
	}

	// Initialize pools
	alloc.planePool = sync.Pool{
		New: func() interface{} {
			return types.NewPlane(0, 0, config.PlaneSize)
		},
	}

	// Start worker goroutines if concurrency enabled
	if config.EnableConcurrent {
		fmt.Println("Allocator3D: starting with concurrency enabled")
		alloc.startWorkers()
	}

	return alloc
}

func (a *Allocator3D) startWorkers() {
	for i := 0; i < a.config.WorkerCount; i++ {
		go a.allocWorker()
		//go a.freeWorker()
	}
}

func (a *Allocator3D) allocWorker() {
	for req := range a.workers {
		result := a.doAllocate(req.Size, req.RegionType, req.Flags)
		req.Result <- result
	}
}

func (a *Allocator3D) doAllocate(size uint32, regionType types.RegionType, flags types.AllocationFlags) AllocResult {
	alloc, err := a.allocateSync(size, regionType, flags)
	if err != nil {
		return AllocResult{
			Error: err,
		}
	}

	return AllocResult{
		Address:    alloc.Address,
		Allocation: alloc,
		Error:      nil,
	}
}

func (a *Allocator3D) Allocate(size uint32, regionType types.RegionType, flags types.AllocationFlags) (*types.Allocation, error) {
	if a.config.EnableConcurrent {
		return a.allocateConcurrent(size, regionType, flags)
	}
	return a.allocateSync(size, regionType, flags)
}

func (a *Allocator3D) allocateConcurrent(size uint32, regionType types.RegionType, flags types.AllocationFlags) (*types.Allocation, error) {
	// Send allocation request to worker pool
	resultChan := make(chan AllocResult, 1)
	req := &AllocRequest{
		Size:       size,
		RegionType: regionType,
		Flags:      flags,
		Result:     resultChan,
	}

	// Send request to workers
	// Note: This is non-blocking with buffered channel
	// For production use, implement proper queue management and timeout
	select {
	case a.workers <- req:
		// Wait for result
		result := <-resultChan
		return result.Allocation, result.Error
	default:
		// Worker queue full, fallback to sync allocation
		// TODO: Implement proper queue management with timeout
		return a.allocateSync(size, regionType, flags)
	}
}

func (a *Allocator3D) allocateSync(size uint32, regionType types.RegionType, flags types.AllocationFlags) (*types.Allocation, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Try fast path first
	if alloc := a.tryFastAllocation(size, regionType); alloc != nil {
		return alloc, nil
	}

	// Try Y-promotion
	if alloc := a.tryYPromotion(size, regionType, flags); alloc != nil {
		atomic.AddUint64(&a.stats.YPromotions, 1)
		return alloc, nil
	}

	// Create new region if needed
	return a.allocateNewRegion(size, regionType, flags)
}

func (a *Allocator3D) tryFastAllocation(size uint32, regionType types.RegionType) *types.Allocation {
	// Check recently used regions first
	a.lruMutex.RLock()
	var promoteTo uint64
	for e := a.regionLRU.Front(); e != nil; e = e.Next() {
		region := e.Value.(*types.Region)
		if region.Type != regionType {
			continue
		}

		// Check if region has enough free space
		if atomic.LoadUint64(&region.FreeBytes) < uint64(size) {
			continue
		}

		// Try to allocate in region's planes
		region.RLock()
		for _, plane := range region.Planes {
			if uint32(plane.FreeBytes()) >= size {
				if startZ, ok := plane.Allocate(uint16(size)); ok {
					// Create allocation
					allocID := atomic.AddUint64(&a.nextAllocID, 1)
					addr := types.Address3D{X: region.ID, Y: plane.ID, Z: startZ}
					alloc := types.NewAllocation(allocID, addr, size, regionType)

					// Update plane's allocation map
					plane.Allocations[startZ] = alloc

					// Update statistics
					atomic.AddUint64(&region.AllocCount, 1)
					atomic.AddUint64(&region.FreeBytes, ^uint64(size-1))

					region.RUnlock()
					promoteTo = region.ID
					a.lruMutex.RUnlock()
					a.promoteRegionLRU(promoteTo)
					return alloc
				}
			}
		}
		region.RUnlock()
	}
	a.lruMutex.RUnlock()

	return nil
}

func (a *Allocator3D) tryYPromotion(size uint32, regionType types.RegionType, flags types.AllocationFlags) *types.Allocation {
	// Find region of correct type
	var targetRegion *types.Region
	for _, region := range a.regions {
		if region.Type == regionType {
			targetRegion = region
			break
		}
	}

	if targetRegion == nil {
		return nil
	}

	// Calculate planes needed
	planesNeeded := (size + uint32(a.config.PlaneSize) - 1) / uint32(a.config.PlaneSize)
	if planesNeeded > uint32(a.config.MaxYPromotions) {
		return nil // Too large for Y-promotion
	}

	// Find planes with enough free space
	candidatePlanes := make([]*types.Plane, 0, planesNeeded)
	remaining := size

	targetRegion.RLock()
	for _, plane := range targetRegion.Planes {
		if remaining == 0 {
			break
		}

		freeBytes := plane.FreeBytes()
		if freeBytes > 0 {
			candidatePlanes = append(candidatePlanes, plane)
			if uint32(freeBytes) >= remaining {
				remaining = 0
			} else {
				remaining -= uint32(freeBytes)
			}
		}
	}
	targetRegion.RUnlock()

	if remaining > 0 || len(candidatePlanes) < int(planesNeeded) {
		return nil
	}

	// Create Y-promoted allocation
	allocID := atomic.AddUint64(&a.nextAllocID, 1)
	alloc := &types.Allocation{
		ID:         allocID,
		Size:       size,
		RegionType: regionType,
		Flags:      flags,
		CreatedAt:  time.Now().UnixNano(),
		LastUsed:   time.Now().UnixNano(),
	}

	// Allocate in each plane
	remaining = size
	targetRegion.Lock()
	defer targetRegion.Unlock()

	for i, plane := range candidatePlanes {
		allocSize := uint16(min(uint32(plane.FreeBytes()), remaining))
		if allocSize == 0 {
			continue
		}

		startZ, ok := plane.Allocate(allocSize)
		if !ok {
			// Rollback previous allocations
			for j := 0; j < i; j++ {
				_ = candidatePlanes[j]
				// Need to track what we allocated to rollback
				// Simplified for now
			}
			return nil
		}

		// Add fragment
		isFirst := i == 0
		alloc.AddFragment(plane.ID, startZ, allocSize, isFirst)

		// Store allocation in first plane
		if isFirst {
			plane.Allocations[startZ] = alloc
			alloc.Address = types.Address3D{X: targetRegion.ID, Y: plane.ID, Z: startZ}
		}

		remaining -= uint32(allocSize)
	}

	if remaining > 0 {
		return nil
	}

	// Update statistics
	atomic.AddUint64(&targetRegion.AllocCount, 1)
	atomic.AddUint64(&targetRegion.FreeBytes, ^uint64(size-1))

	a.promoteRegionLRU(targetRegion.ID)
	return alloc
}

func (a *Allocator3D) allocateNewRegion(size uint32, regionType types.RegionType, flags types.AllocationFlags) (*types.Allocation, error) {
	// Create new region
	regionID := atomic.AddUint64(&a.nextRegionID, 1)
	region := types.NewRegion(regionID, regionType)

	// Create initial plane
	plane := a.getPlaneFromPool(0, regionID)
	region.AddPlane(plane)

	a.regions[regionID] = region

	// Allocate in the new plane
	if uint32(plane.FreeBytes()) >= size {
		startZ, ok := plane.Allocate(uint16(size))
		if !ok {
			return nil, fmt.Errorf("failed to allocate in new plane")
		}

		// Create allocation
		allocID := atomic.AddUint64(&a.nextAllocID, 1)
		addr := types.Address3D{X: regionID, Y: 0, Z: startZ}
		alloc := types.NewAllocation(allocID, addr, size, regionType)
		alloc.Flags = flags

		plane.Allocations[startZ] = alloc

		// Update statistics
		atomic.AddUint64(&region.AllocCount, 1)
		atomic.AddUint64(&region.FreeBytes, ^uint64(size-1))

		a.promoteRegionLRU(regionID)
		return alloc, nil
	}

	// If single plane not enough, use Y-promotion
	return a.tryYPromotion(size, regionType, flags), nil
}

func (a *Allocator3D) Free(alloc *types.Allocation) error {
	if a.config.EnableConcurrent {
		return a.freeConcurrent(alloc)
	}
	return a.freeSync(alloc)
}

func (a *Allocator3D) freeConcurrent(alloc *types.Allocation) error {
	// For now, delegate to sync free
	// TODO: Implement concurrent free worker if needed
	return a.freeSync(alloc)
}

func (a *Allocator3D) freeSync(alloc *types.Allocation) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	region, exists := a.regions[alloc.Address.X]
	if !exists {
		return fmt.Errorf("region %d not found", alloc.Address.X)
	}

	region.Lock()
	defer region.Unlock()

	if alloc.IsYPromoted() {
		// Free all fragments
		for _, fragment := range alloc.Fragments {
			// Inline plane lookup to avoid deadlock (we already have write lock)
			var plane *types.Plane
			for _, p := range region.Planes {
				if p.ID == fragment.PlaneID {
					plane = p
					break
				}
			}
			if plane != nil {
				plane.Free(fragment.StartZ)
			}
		}
	} else {
		// Single plane allocation
		var plane *types.Plane
		for _, p := range region.Planes {
			if p.ID == alloc.Address.Y {
				plane = p
				break
			}
		}
		if plane == nil {
			return fmt.Errorf("plane %d not found", alloc.Address.Y)
		}
		if !plane.Free(alloc.Address.Z) {
			return fmt.Errorf("failed to free allocation at %v", alloc.Address)
		}
	}

	// Update statistics
	atomic.AddUint64(&region.AllocCount, ^uint64(0))
	atomic.AddUint64(&region.FreeBytes, uint64(alloc.Size))
	atomic.AddUint64(&a.stats.Frees, 1)

	return nil
}

func (a *Allocator3D) promoteRegionLRU(regionID uint64) {
	a.lruMutex.Lock()
	defer a.lruMutex.Unlock()

	// Remove if already in list
	for e := a.regionLRU.Front(); e != nil; e = e.Next() {
		if e.Value.(*types.Region).ID == regionID {
			a.regionLRU.Remove(e)
			break
		}
	}

	// Add to front
	if region, exists := a.regions[regionID]; exists {
		a.regionLRU.PushFront(region)

		// Trim if too large
		if a.regionLRU.Len() > 100 {
			a.regionLRU.Remove(a.regionLRU.Back())
		}
	}
}

func (a *Allocator3D) getPlaneFromPool(id uint32, regionID uint64) *types.Plane {
	if a.config.EnablePooling {
		plane := a.planePool.Get().(*types.Plane)
		plane.ID = id
		plane.RegionID = regionID
		return plane
	}
	return types.NewPlane(id, regionID, a.config.PlaneSize)
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

// GetCompactionReport returns fragmentation info for all regions
func (a *Allocator3D) GetCompactionReport() map[uint64]types.CompactionReport {
	a.mu.RLock()
	defer a.mu.RUnlock()

	report := make(map[uint64]types.CompactionReport)
	for _, region := range a.regions {
		compReport := region.GetCompactionReport()
		report[region.ID] = compReport
	}
	return report
}

// CompactionConfig stores compaction strategy parameters
type CompactionConfig struct {
	FragmentationThreshold float64 // Trigger compaction if fragmentation exceeds this
	MaxAllocationsToMove   int     // Maximum allocations to move in one compaction pass
	GasRefundPercentage    float64 // What % of saved gas to refund back to user
}

// DefaultCompactionConfig returns conservative compaction settings
func DefaultCompactionConfig() CompactionConfig {
	return CompactionConfig{
		FragmentationThreshold: 0.30,
		MaxAllocationsToMove:   10,
		GasRefundPercentage:    0.50,
	}
}

// AnalyzeCompaction determines if and what to compact
// Returns list of allocation IDs that should be moved to achieve consolidation
func (a *Allocator3D) AnalyzeCompaction(config CompactionConfig) []uint64 {
	a.mu.RLock()
	defer a.mu.RUnlock()

	allocsToMove := make([]uint64, 0)

	// Scan all regions for fragmentation
	for _, region := range a.regions {
		report := region.GetCompactionReport()

		// Skip if not fragmented enough
		if !report.RecommendCompact || report.Fragmentation < config.FragmentationThreshold {
			continue
		}

		// Find coldest allocations in this region (lowest AccessCount)
		region.RLock()
		for _, plane := range region.Planes {
			for _, alloc := range plane.Allocations {
				// Skip pinned and Y-promoted allocations
				if (alloc.Flags & types.FlagPinned) != 0 || alloc.IsYPromoted() {
					continue
				}

				// Prefer allocations with low access count
				allocsToMove = append(allocsToMove, alloc.ID)

				// Stop when we have enough allocations to move
				if len(allocsToMove) >= config.MaxAllocationsToMove {
					region.RUnlock()
					return allocsToMove
				}
			}
		}
		region.RUnlock()
	}

	return allocsToMove
}

// Shutdown cleanly closes worker channels, allowing worker goroutines to exit
// This MUST be called when allocator is no longer needed (especially if workers were started)
func (a *Allocator3D) Shutdown() {
	close(a.workers)
	close(a.freeWorkers)
}

// IsConcurrent returns true when the allocator was configured to run worker goroutines.
func (a *Allocator3D) IsConcurrent() bool {
	return a.config.EnableConcurrent
}
