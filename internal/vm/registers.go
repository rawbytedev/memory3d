package vm

import (
	"fmt"

	"github.com/rawbytedev/memory3d/internal/types"
)

type Register byte

const (
	R0 Register = iota
	R1
	R2
	R3
	R4
	R5
	R6
	R7
	R8
	R9
	R10
	R11
	R12
	R13
	R14
	R15
	// Special purpose registers
	SP // Stack pointer
	BP // Base pointer
	PC // Program counter
	GP // Gas pointer
	MP // Memory pointer
)

type Registers struct {
	General [16][]byte
	Special [5]uint64

	// Cache for frequently used addresses
	addrCache map[types.Address3D][]byte
}

func NewRegisters() *Registers {
	return &Registers{
		General:   [16][]byte{},
		Special:   [5]uint64{},
		addrCache: make(map[types.Address3D][]byte),
	}
}

func (r *Registers) Get(reg Register) []byte {
	if reg <= R15 {
		return r.General[reg]
	}
	return nil
}

func (r *Registers) Set(reg Register, value []byte) {
	if reg <= R15 {
		r.General[reg] = value
	}
}

func (r *Registers) GetUint64(reg Register) uint64 {
	if reg <= R15 {
		if len(r.General[reg]) >= 8 {
			return bytesToUint64(r.General[reg][:8])
		}
	}
	return 0
}

func (r *Registers) SetUint64(reg Register, value uint64) {
	if reg <= R15 {
		r.General[reg] = uint64ToBytes(value)
	}
}

func (r *Registers) GetAddress(reg Register) (types.Address3D, error) {
	data := r.Get(reg)
	if len(data) != 14 {
		return types.Address3D{}, fmt.Errorf("invalid address length")
	}
	return types.FromBytes(data)
}

func (r *Registers) SetAddress(reg Register, addr types.Address3D) {
	r.Set(reg, addr.Bytes())
}
