package vm

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rawbytedev/memory3d/internal/allocator"
	"github.com/rawbytedev/memory3d/internal/types"
)

// RegionStats tracks statistics for a memory region
type RegionStats struct {
	TotalAllocations uint64
	TotalFrees       uint64
	CurrentUsage     uint64
	MaxUsage         uint64
	AccessCount      uint64
}

// AccessEntry records a memory access for auditing and analysis
type AccessEntry struct {
	Address types.Address3D
	Time    int64
	Type    AccessType
	Size    uint32
}

// AccessType categorizes memory operations
type AccessType byte

const (
	AccessRead AccessType = iota
	AccessWrite
	AccessAlloc
	AccessFree
)

// MemoryManager manages memory operations, access tracking, and allocation binding
type MemoryManager struct {
	allocator   *allocator.Allocator3D
	allocations map[types.Address3D]*types.Allocation
	regions     map[uint64]*RegionStats
	accessLog   []AccessEntry
	mu          sync.RWMutex
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager() *MemoryManager {
	return &MemoryManager{
		allocations: make(map[types.Address3D]*types.Allocation),
		regions:     make(map[uint64]*RegionStats),
		accessLog:   make([]AccessEntry, 0, 1000),
	}
}

// UpdateAccess records memory access for an address
func (mm *MemoryManager) UpdateAccess(addr types.Address3D) {
	mm.mu.RLock()
	alloc := mm.allocations[addr]
	mm.mu.RUnlock()

	if alloc != nil {
		alloc.UpdateAccess(0)
	}
}

// getAllocation returns the allocation containing the given address.
// It checks exact match first, then checks Y-promoted fragments.
func (mm *MemoryManager) getAllocation(addr types.Address3D) *types.Allocation {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	// First check exact match
	if alloc, exists := mm.allocations[addr]; exists {
		return alloc
	}

	// For Y-promoted allocations, check fragments
	for _, alloc := range mm.allocations {
		if alloc.IsYPromoted() {
			for _, frag := range alloc.Fragments {
				if addr.Y == frag.PlaneID &&
					addr.Z >= frag.StartZ &&
					addr.Z < frag.StartZ+frag.Size {
					return alloc
				}
			}
		}
	}

	return nil
}

// Load3D reads data from a 3D memory address
func (mm *MemoryManager) Load3D(addr types.Address3D, size uint32) ([]byte, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	// Find allocation containing this address
	alloc := mm.getAllocationLocked(addr)
	if alloc == nil {
		return nil, fmt.Errorf("address not allocated: %v", addr)
	}

	// Check if read is within allocation bounds (compute relative offset within allocation)
	relStart := addr.Z - alloc.Address.Z
	endOffset := relStart + uint16(size)
	if endOffset > uint16(alloc.Size) {
		return nil, fmt.Errorf("read out of bounds")
	}

	// Get region and plane
	region := mm.allocator.GetRegion(addr.X)
	if region == nil {
		return nil, fmt.Errorf("region not found: %d", addr.X)
	}

	plane := region.GetPlane(addr.Y)
	if plane == nil {
		return nil, fmt.Errorf("plane not found: %d", addr.Y)
	}

	// Calculate actual offset in plane data
	planeStart := uint32(addr.Z)
	planeEnd := planeStart + size

	if planeEnd > uint32(len(plane.Data)) {
		return nil, fmt.Errorf("plane read out of bounds")
	}

	// Copy data
	data := make([]byte, size)
	copy(data, plane.Data[planeStart:planeEnd])

	// Log access
	mm.logAccess(addr, AccessRead, size)

	// Update region stats
	if stats, exists := mm.regions[addr.X]; exists {
		atomic.AddUint64(&stats.AccessCount, 1)
	}

	return data, nil
}

func (mm *MemoryManager) Store3D(addr types.Address3D, data []byte) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	size := uint32(len(data))

	// Find allocation containing this address
	alloc := mm.getAllocationLocked(addr)
	if alloc == nil {
		return fmt.Errorf("address not allocated: %v", addr)
	}

	// Check if write is within allocation bounds (compute relative offset within allocation)
	relStart := addr.Z - alloc.Address.Z
	endOffset := relStart + uint16(size)
	if endOffset > uint16(alloc.Size) {
		return fmt.Errorf("write out of bounds")
	}

	// Get region and plane
	region := mm.allocator.GetRegion(addr.X)
	if region == nil {
		return fmt.Errorf("region not found: %d", addr.X)
	}

	plane := region.GetPlane(addr.Y)
	if plane == nil {
		return fmt.Errorf("plane not found: %d", addr.Y)
	}

	// Calculate actual offset in plane data
	planeStart := uint32(addr.Z)
	planeEnd := planeStart + size

	if planeEnd > uint32(len(plane.Data)) {
		return fmt.Errorf("plane write out of bounds")
	}

	// Copy data
	copy(plane.Data[planeStart:planeEnd], data)

	// Log access
	mm.logAccess(addr, AccessWrite, size)

	// Update allocation access time
	alloc.UpdateAccess(0) // Gas will be tracked separately

	return nil
}

func (mm *MemoryManager) getAllocationLocked(addr types.Address3D) *types.Allocation {
	// First check exact match
	if alloc, exists := mm.allocations[addr]; exists {
		return alloc
	}

	// For Y-promoted allocations, check fragments
	for _, alloc := range mm.allocations {
		if alloc.IsYPromoted() {
			for _, frag := range alloc.Fragments {
				if addr.Y == frag.PlaneID &&
					addr.Z >= frag.StartZ &&
					addr.Z < frag.StartZ+frag.Size {
					return alloc
				}
			}
		}
	}

	return nil
}

func (mm *MemoryManager) trackAllocation(alloc *types.Allocation) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.allocations[alloc.Address] = alloc

	// Update region stats
	stats, exists := mm.regions[alloc.Address.X]
	if !exists {
		stats = &RegionStats{}
		mm.regions[alloc.Address.X] = stats
	}

	atomic.AddUint64(&stats.TotalAllocations, 1)
	atomic.AddUint64(&stats.CurrentUsage, uint64(alloc.Size))

	if stats.CurrentUsage > stats.MaxUsage {
		stats.MaxUsage = stats.CurrentUsage
	}

	// Log access
	mm.logAccess(alloc.Address, AccessAlloc, alloc.Size)
}

func (mm *MemoryManager) untrackAllocation(addr types.Address3D) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	alloc, exists := mm.allocations[addr]
	if !exists {
		return
	}

	// Update region stats
	if stats, exists := mm.regions[addr.X]; exists {
		atomic.AddUint64(&stats.TotalFrees, 1)
		atomic.AddUint64(&stats.CurrentUsage, ^uint64(alloc.Size-1))
	}

	delete(mm.allocations, addr)

	// Log access
	mm.logAccess(addr, AccessFree, alloc.Size)
}

func (mm *MemoryManager) logAccess(addr types.Address3D, accessType AccessType, size uint32) {
	entry := AccessEntry{
		Address: addr,
		Time:    time.Now().UnixNano(),
		Type:    accessType,
		Size:    size,
	}

	mm.accessLog = append(mm.accessLog, entry)

	// Keep log bounded
	if len(mm.accessLog) > 10000 {
		mm.accessLog = mm.accessLog[1000:] // Remove oldest 1000 entries
	}
}

func (mm *MemoryManager) GetFreeMemory(regionX uint64) uint64 {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	stats, exists := mm.regions[regionX]
	if !exists {
		return 0
	}
	fmt.Printf("%d", stats.CurrentUsage)
	// This is simplified - in reality would query allocator
	return 1024 * 1024 * 64 // Example: 64MB free
}

func (mm *MemoryManager) GetFragmentation(regionX uint64) float64 {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	// This would query the allocator for actual fragmentation
	// For now, return a placeholder
	return 0.25 // 25% fragmentation
}

// RelocateAllocation moves an allocation from oldAddr to newAddr
// Used by compaction to consolidate memory
func (mm *MemoryManager) RelocateAllocation(oldAddr, newAddr types.Address3D) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// Find allocation at old address
	alloc, exists := mm.allocations[oldAddr]
	if !exists {
		return fmt.Errorf("allocation not found at %v", oldAddr)
	}

	// Ensure new address is available
	if _, exists := mm.allocations[newAddr]; exists {
		return fmt.Errorf("destination %v already occupied", newAddr)
	}

	// Update allocation address
	alloc.Address = newAddr

	// Remove old mapping, add new mapping
	delete(mm.allocations, oldAddr)
	mm.allocations[newAddr] = alloc

	// Log relocation
	mm.logAccess(newAddr, AccessAlloc, alloc.Size)

	return nil
}

// GetAllAllocations returns a snapshot of all allocations (for compaction analysis)
func (mm *MemoryManager) GetAllAllocations() []*types.Allocation {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	allocs := make([]*types.Allocation, 0, len(mm.allocations))
	for _, alloc := range mm.allocations {
		allocs = append(allocs, alloc)
	}
	return allocs
}
