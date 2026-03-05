package types

import (
	"math/bits"
)

// Bitmap for tracking free/used memory in a plane
type Bitmap struct {
    bits []uint64
    size uint16
}

// NewBitmap creates a new bitmap for given size (in bytes)
func NewBitmap(size uint16) *Bitmap {
    // Each bit represents 1 byte, so we need size bits
    // Use int arithmetic to avoid uint16 overflow when adding 63
    numUint64 := int((int(size) + 63) / 64)
    return &Bitmap{
        bits: make([]uint64, numUint64),
        size: size,
    }
}

// Set sets a bit at position pos
func (b *Bitmap) Set(pos uint16) {
	if pos >= b.size || len(b.bits) == 0 {
		return
	}
    idx := int(pos / 64)
    if idx >= len(b.bits) {
        return
    }
    bit := pos % 64
    b.bits[idx] |= uint64(1) << bit
}

// Clear clears a bit at position pos
func (b *Bitmap) Clear(pos uint16) {
	if pos >= b.size || len(b.bits) == 0 {
		return
	}
    idx := int(pos / 64)
    if idx >= len(b.bits) {
        return
    }
    bit := pos % 64
    b.bits[idx] &^= uint64(1) << bit
}

// IsSet checks if a bit is set
func (b *Bitmap) IsSet(pos uint16) bool {
	if pos >= b.size || len(b.bits) == 0 {
		return false
	}
    idx := int(pos / 64)
    if idx >= len(b.bits) {
        return false
    }
    bit := pos % 64
    return (b.bits[idx] & (uint64(1) << bit)) != 0
}

// FindContiguous finds contiguous free space of given size
func (b *Bitmap) FindContiguous(size uint16) (uint16, bool) {
    if size == 0 || size > b.size {
        return 0, false
    }
    
    // Simple linear search for now (can be optimized)
    for i := uint16(0); i <= b.size-size; i++ {
        found := true
        for j := uint16(0); j < size; j++ {
            if b.IsSet(i + j) {
                found = false
                break
            }
        }
        if found {
            return i, true
        }
    }
    return 0, false
}

// SetRange sets a range of bits
func (b *Bitmap) SetRange(start, end uint16) {
    for i := start; i < end; i++ {
        b.Set(i)
    }
}

// ClearRange clears a range of bits
func (b *Bitmap) ClearRange(start, end uint16) {
    for i := start; i < end; i++ {
        b.Clear(i)
    }
}

// FreeBytes returns number of free bytes
func (b *Bitmap) FreeBytes() uint16 {
    totalSet := 0
    for _, word := range b.bits {
        totalSet += bits.OnesCount64(word)
    }
    return b.size - uint16(totalSet)
}

// UsedBytes returns number of used bytes
func (b *Bitmap) UsedBytes() uint16 {
    totalSet := 0
    for _, word := range b.bits {
        totalSet += bits.OnesCount64(word)
    }
    return uint16(totalSet)
}

// Fragmentation returns fragmentation percentage (0-1)
func (b *Bitmap) Fragmentation() float64 {
    // Count holes (free sequences between used bits)
    holes := 0
    inHole := false
    
    for i := uint16(0); i < b.size; i++ {
        if b.IsSet(i) {
            inHole = false
        } else {
            if !inHole {
                holes++
                inHole = true
            }
        }
    }
    
    freeBytes := b.FreeBytes()
    if freeBytes == 0 {
        return 0.0
    }
    
    // More holes = more fragmentation
    return float64(holes-1) / float64(freeBytes)
}

// BitsLen returns the internal bits slice length for debugging and testing.
// Should only be used in tests and diagnostics.
func (b *Bitmap) BitsLen() int {
	return len(b.bits)
}

// FirstWord returns the first uint64 word in the bitmap for debugging and testing.
// Should only be used in tests and diagnostics.
func (b *Bitmap) FirstWord() uint64 {
    if len(b.bits) == 0 {
        return 0
    }
    return b.bits[0]
}
