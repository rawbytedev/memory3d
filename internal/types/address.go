package types

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// Address3D represents a 3D memory address
type Address3D struct {
	X uint64 `json:"x"` // Region (64 bits)
	Y uint32 `json:"y"` // Plane within region (32 bits)
	Z uint16 `json:"z"` // Offset within plane (16 bits)
}

// NewAddress creates a new 3D address
func NewAddress(x uint64, y uint32, z uint16) Address3D {
	return Address3D{X: x, Y: y, Z: z}
}

// IsValid checks if the address is within bounds
func (a Address3D) IsValid() bool {
	return a.X < MaxXRegions && a.Y < MaxYPlanes && a.Z <= MaxZOffset
}

// Bytes serializes the address to bytes
func (a Address3D) Bytes() []byte {
	buf := make([]byte, 14) // 8 + 4 + 2
	binary.BigEndian.PutUint64(buf[0:8], a.X)
	binary.BigEndian.PutUint32(buf[8:12], a.Y)
	binary.BigEndian.PutUint16(buf[12:14], a.Z)
	return buf
}

// FromBytes deserializes bytes to address
func FromBytes(data []byte) (Address3D, error) {
	if len(data) != 14 {
		return Address3D{}, fmt.Errorf("invalid address length: %d", len(data))
	}
	return Address3D{
		X: binary.BigEndian.Uint64(data[0:8]),
		Y: binary.BigEndian.Uint32(data[8:12]),
		Z: binary.BigEndian.Uint16(data[12:14]),
	}, nil
}

// String returns string representation
func (a Address3D) String() string {
	return fmt.Sprintf("[X:%d,Y:%d,Z:%d]", a.X, a.Y, a.Z)
}

// ParseAddress parses string representation
func ParseAddress(s string) (Address3D, error) {
	var x, y, z uint64
	_, err := fmt.Sscanf(strings.Trim(s, "[]"), "X:%d,Y:%d,Z:%d", &x, &y, &z)
	if err != nil {
		return Address3D{}, err
	}
	return Address3D{X: uint64(x), Y: uint32(y), Z: uint16(z)}, nil
}

// AddZ adds offset to Z dimension
func (a Address3D) AddZ(offset uint16) Address3D {
	return Address3D{X: a.X, Y: a.Y, Z: a.Z + offset}
}

// NextY moves to next plane
func (a Address3D) NextY() Address3D {
	return Address3D{X: a.X, Y: a.Y + 1, Z: 0}
}

// Compare returns -1, 0, 1 for comparison
func (a Address3D) Compare(b Address3D) int {
	if a.X != b.X {
		if a.X < b.X {
			return -1
		}
		return 1
	}
	if a.Y != b.Y {
		if a.Y < b.Y {
			return -1
		}
		return 1
	}
	if a.Z != b.Z {
		if a.Z < b.Z {
			return -1
		}
		return 1
	}
	return 0
}

// ManhattanDistance calculates distance between addresses
func (a Address3D) ManhattanDistance(b Address3D) uint64 {
	dx := absDiff64(a.X, b.X)
	dy := absDiff32(a.Y, b.Y)
	dz := absDiff16(a.Z, b.Z)
	return uint64(dx)*100 + uint64(dy)*10 + uint64(dz)
}

func absDiff64(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
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
