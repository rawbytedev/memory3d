package types

import (
    "sync/atomic"
    "time"
)

// Allocation represents a memory allocation
type Allocation struct {
    ID          uint64        `json:"id"`
    Address     Address3D     `json:"address"`
    Size        uint32        `json:"size"`
    RegionType  RegionType    `json:"region_type"`
    Flags       AllocationFlags `json:"flags"`
    
    // Metadata
    CreatedAt   int64         `json:"created_at"`
    LastUsed    int64         `json:"last_used"`
    AccessCount uint64        `json:"access_count"`
    GasUsed     uint64        `json:"gas_used"`
    
    // For Y-promotion allocations
    Fragments   []Fragment    `json:"fragments,omitempty"`
    
    // Linked list for fragmented allocations
    Next        *Allocation   `json:"-"`
}

// AllocationFlags bitmask
type AllocationFlags byte

const (
    FlagContiguous AllocationFlags = 1 << iota
    FlagShared
    FlagPinned      // Cannot be moved during compaction
    FlagExecutable
    FlagAtomic      // Atomic operations allowed
)

// Fragment represents part of a Y-promoted allocation
type Fragment struct {
    PlaneID     uint32    `json:"plane_id"`
    StartZ      uint16    `json:"start_z"`
    Size        uint16    `json:"size"`
    IsFirst     bool      `json:"is_first"`
}

// NewAllocation creates a new allocation
func NewAllocation(id uint64, addr Address3D, size uint32, regionType RegionType) *Allocation {
    now := time.Now().UnixNano()
    return &Allocation{
        ID:         id,
        Address:    addr,
        Size:       size,
        RegionType: regionType,
        CreatedAt:  now,
        LastUsed:   now,
    }
}

// AddFragment adds a fragment to Y-promoted allocation
func (a *Allocation) AddFragment(planeID uint32, startZ, size uint16, isFirst bool) {
    fragment := Fragment{
        PlaneID: planeID,
        StartZ:  startZ,
        Size:    size,
        IsFirst: isFirst,
    }
    a.Fragments = append(a.Fragments, fragment)
}

// IsYPromoted returns true if allocation uses Y-promotion
func (a *Allocation) IsYPromoted() bool {
    return len(a.Fragments) > 1
}

// UpdateAccess updates access statistics
func (a *Allocation) UpdateAccess(gasUsed uint64) {
    atomic.StoreInt64(&a.LastUsed, time.Now().UnixNano())
    atomic.AddUint64(&a.AccessCount, 1)
    atomic.AddUint64(&a.GasUsed, gasUsed)
}

// GetHotnessScore calculates allocation hotness
func (a *Allocation) GetHotnessScore() float64 {
    now := time.Now().UnixNano()
    timeSinceAccess := float64(now-a.LastUsed) / 1e9 // Seconds
    
    // Exponential decay: older accesses matter less
    decayFactor := 0.5
    agePenalty := 1.0 / (1.0 + timeSinceAccess*decayFactor)
    
    // Access frequency component
    freqComponent := float64(atomic.LoadUint64(&a.AccessCount)) / 1000.0
    
    return agePenalty * freqComponent
}
