package types

import (
	"sync"
	"sync/atomic"
	"time"
)

// RegionType enum
type RegionType byte

const (
	RegionTypeInvalid RegionType = iota
	RegionTypeVMControl
	RegionTypeCode
	RegionTypeState
	RegionTypeHeap
	RegionTypeStack
	RegionTypeShared
	RegionTypeTransient
)

// Permissions for memory regions
type Permissions struct {
	Readable   bool `json:"readable"`
	Writable   bool `json:"writable"`
	Executable bool `json:"executable"`
	Shared     bool `json:"shared"`
	Atomic     bool `json:"atomic"`
}

// Region represents a memory region (X dimension)
type Region struct {
	ID          uint64      `json:"id"`
	Type        RegionType  `json:"type"`
	Permissions Permissions `json:"permissions"`
	Planes      []*Plane    `json:"-"`
	MerkleRoot  [32]byte    `json:"merkle_root"`

	// Statistics (atomic for lock-free reads)
	AllocCount  uint64 `json:"alloc_count"`
	FreeBytes   uint64 `json:"free_bytes"`
	TotalBytes  uint64 `json:"total_bytes"`
	AccessCount uint64 `json:"access_count"`
	LastAccess  int64  `json:"last_access"` // Unix nano

	// Hotness tracking for compaction
	HotnessScore float64 `json:"hotness_score"`

	// Mutex for plane operations
	mu sync.RWMutex `json:"-"`
}

// NewRegion creates a new region
func NewRegion(id uint64, regionType RegionType) *Region {
	perms := GetDefaultPermissions(regionType)
	return &Region{
		ID:          id,
		Type:        regionType,
		Permissions: perms,
		Planes:      make([]*Plane, 0, 16),
		LastAccess:  time.Now().UnixNano(),
	}
}

// GetDefaultPermissions returns default permissions for region type
func GetDefaultPermissions(regionType RegionType) Permissions {
	switch regionType {
	case RegionTypeVMControl:
		return Permissions{Readable: true, Writable: true}
	case RegionTypeCode:
		return Permissions{Readable: true, Executable: true}
	case RegionTypeState:
		return Permissions{Readable: true, Writable: true}
	case RegionTypeHeap:
		return Permissions{Readable: true, Writable: true}
	case RegionTypeStack:
		return Permissions{Readable: true, Writable: true}
	case RegionTypeShared:
		return Permissions{Readable: true, Writable: true, Shared: true}
	case RegionTypeTransient:
		return Permissions{Readable: true, Writable: true}
	case RegionTypeInvalid:
		fallthrough
	default:
		return Permissions{}
	}
}
func (r *Region) RLock() {
	r.mu.RLock()
}
func (r *Region) RUnlock() {
	r.mu.RUnlock()
}
func (r *Region) Lock() {
	r.mu.Lock()
}
func (r *Region) Unlock() {
	r.mu.Unlock()
}

// AddPlane adds a new plane to the region
func (r *Region) AddPlane(plane *Plane) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Planes = append(r.Planes, plane)
	atomic.AddUint64(&r.TotalBytes, uint64(plane.Size))
	atomic.AddUint64(&r.FreeBytes, uint64(plane.FreeBytes()))
}

// RemovePlane removes a plane from the region
func (r *Region) RemovePlane(planeID uint32) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, plane := range r.Planes {
		if plane.ID == planeID {
			// Update statistics
			atomic.AddUint64(&r.TotalBytes, ^uint64(plane.Size-1))
			atomic.AddUint64(&r.FreeBytes, ^uint64(plane.FreeBytes()-1))

			// Remove from slice
			r.Planes = append(r.Planes[:i], r.Planes[i+1:]...)
			return true
		}
	}
	return false
}

// GetPlane returns a plane by ID
func (r *Region) GetPlane(planeID uint32) *Plane {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, plane := range r.Planes {
		if plane.ID == planeID {
			return plane
		}
	}
	return nil
}

// UpdateAccess updates access statistics
func (r *Region) UpdateAccess() {
	atomic.StoreInt64(&r.LastAccess, time.Now().UnixNano())
	atomic.AddUint64(&r.AccessCount, 1)
}

// Fragmentation returns fragmentation percentage
func (r *Region) Fragmentation() float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.Planes) == 0 {
		return 0.0
	}

	totalFrag := 0.0
	for _, plane := range r.Planes {
		totalFrag += plane.Fragmentation()
	}
	return totalFrag / float64(len(r.Planes))
}


// UpdateMerkleRoot updates the merkle root for this region
func (r *Region) UpdateMerkleRoot(root [32]byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.MerkleRoot = root
}

// GetMerkleRoot returns the current merkle root
func (r *Region) GetMerkleRoot() [32]byte {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.MerkleRoot
}

// CompactionReport represents fragmentation data for compaction decisions
type CompactionReport struct {
	RegionID           uint64
	TotalBytes         uint64
	UsedBytes          uint64
	FreeBytes          uint64
	Fragmentation      float64
	AllocationCount    uint32
	RecommendCompact   bool
	EstimatedGasSavings uint64
}

// GetCompactionReport generates a report for compaction analysis
func (r *Region) GetCompactionReport() CompactionReport {
	r.mu.RLock()
	defer r.mu.RUnlock()

	total := atomic.LoadUint64(&r.TotalBytes)
	used := atomic.LoadUint64(&r.TotalBytes) - atomic.LoadUint64(&r.FreeBytes)
	free := atomic.LoadUint64(&r.FreeBytes)
	frag := r.Fragmentation()

	// Recommend compaction if fragmentation > 30%
	recommendCompact := frag > 0.30 && free > 0

	// Estimate gas savings from Y-promotion reduction (simplified)
	var estimatedSavings uint64
	if recommendCompact && free > 1024 {
		// Rough estimate: 50 gas per KB freed
		estimatedSavings = (free / 1024) * 50
	}

	return CompactionReport{
		RegionID:            r.ID,
		TotalBytes:          total,
		UsedBytes:           used,
		FreeBytes:           free,
		Fragmentation:       frag,
		AllocationCount:     uint32(atomic.LoadUint64(&r.AllocCount)),
		RecommendCompact:    recommendCompact,
		EstimatedGasSavings: estimatedSavings,
	}
}
