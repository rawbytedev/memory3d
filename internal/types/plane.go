package types

import (
	"sync"
)

// Plane represents a memory plane (Y dimension)
type Plane struct {
	ID          uint32
	RegionID    uint64
	Data        []byte
	FreeMap     *Bitmap
	Allocations map[uint16]*Allocation // keyed by start Z
	Size        uint16
	AccessCount uint64
	mu          sync.RWMutex
}

// NewPlane creates a new plane
func NewPlane(id uint32, regionID uint64, size uint16) *Plane {
	// Ensure minimum plane size (64KB - 1)
	if size == 0 {
		size = 65535
	}

	return &Plane{
		ID:          id,
		RegionID:    regionID,
		Data:        make([]byte, size),
		FreeMap:     NewBitmap(size),
		Allocations: make(map[uint16]*Allocation),
		Size:        size,
	}
}

// Allocate allocates space in the plane
func (p *Plane) Allocate(size uint16) (uint16, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	start, found := p.FreeMap.FindContiguous(size)
	if !found {
		return 0, false
	}

	p.FreeMap.SetRange(start, start+size)
	return start, true
}

// Free frees allocated space
func (p *Plane) Free(start uint16) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	alloc, exists := p.Allocations[start]
	if !exists {
		return false
	}

	p.FreeMap.ClearRange(start, start+uint16(alloc.Size))
	delete(p.Allocations, start)
	return true
}

// IsAllocated checks if position is allocated
func (p *Plane) IsAllocated(pos uint16) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Check if pos falls within any allocation
	for start, alloc := range p.Allocations {
		if pos >= start && pos < start+uint16(alloc.Size) {
			return true
		}
	}
	return false
}

// GetContainingAllocation returns allocation containing position
func (p *Plane) GetContainingAllocation(pos uint16) *Allocation {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for start, alloc := range p.Allocations {
		if pos >= start && pos < start+uint16(alloc.Size) {
			return alloc
		}
	}
	return nil
}

// Fragmentation returns fragmentation percentage
func (p *Plane) Fragmentation() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.FreeMap.Fragmentation()
}

// FreeBytes returns free bytes count
func (p *Plane) FreeBytes() uint16 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.FreeMap.FreeBytes()
}

// UsedBytes returns used bytes count
func (p *Plane) UsedBytes() uint16 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.FreeMap.UsedBytes()
}
