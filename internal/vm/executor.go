package vm

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/rawbytedev/memory3d/internal/types"
)

func (vm *VM3D) executeMLoad3D(inst *Instruction3D) error {
	if len(inst.Operands) != 3 {
		return fmt.Errorf("MLOAD3D requires 3 operands")
	}

	addr, ok := inst.Operands[0].Value.(types.Address3D)
	if !ok {
		return fmt.Errorf("invalid address operand")
	}

	size, ok := inst.Operands[1].Value.(uint32)
	if !ok {
		return fmt.Errorf("invalid size operand")
	}

	reg, ok := inst.Operands[2].Value.(Register)
	if !ok {
		return fmt.Errorf("invalid register operand")
	}

	// Check cache first
	if data, hit := vm.cache.Get(addr); hit {
		if uint32(len(data)) >= size {
			vm.registers.Set(reg, data[:size])
			return nil
		}
	}

	// Load from memory
	data, err := vm.memory.Load3D(addr, size)
	if err != nil {
		return fmt.Errorf("MLOAD3D failed: %w", err)
	}

	// Cache the result
	vm.cache.Put(addr, data)

	// Store in register
	vm.registers.Set(reg, data)

	// Update access statistics
	vm.memory.UpdateAccess(addr)

	return nil
}

func (vm *VM3D) executeMStore3D(inst *Instruction3D) error {
	if len(inst.Operands) != 3 {
		return fmt.Errorf("MSTORE3D requires 3 operands")
	}

	addr, ok := inst.Operands[0].Value.(types.Address3D)
	if !ok {
		return fmt.Errorf("invalid address operand")
	}

	reg, ok := inst.Operands[1].Value.(Register)
	if !ok {
		return fmt.Errorf("invalid register operand")
	}

	size, ok := inst.Operands[2].Value.(uint32)
	if !ok {
		return fmt.Errorf("invalid size operand")
	}

	// Get data from register
	data := vm.registers.Get(reg)
	if uint32(len(data)) < size {
		return fmt.Errorf("register data too small")
	}

	// Store to memory
	if err := vm.memory.Store3D(addr, data[:size]); err != nil {
		return fmt.Errorf("MSTORE3D failed: %w", err)
	}

	// Update cache
	vm.cache.Put(addr, data[:size])

	// Update Merkle tree if enabled
	if vm.merkleTree != nil {
		vm.merkleTree.Update(addr, data[:size])
	}

	// Update access statistics
	vm.memory.UpdateAccess(addr)

	return nil
}

func (vm *VM3D) executeMAlloc3D(inst *Instruction3D) error {
	if len(inst.Operands) != 3 {
		return fmt.Errorf("MALLOC3D requires 3 operands")
	}

	size, ok := inst.Operands[0].Value.(uint32)
	if !ok {
		return fmt.Errorf("invalid size operand")
	}

	regionType, ok := inst.Operands[1].Value.(types.RegionType)
	if !ok {
		return fmt.Errorf("invalid region type operand")
	}

	flags, ok := inst.Operands[2].Value.(types.AllocationFlags)
	if !ok {
		flags = 0
	}

	// Allocate memory
	alloc, err := vm.allocator.Allocate(size, regionType, flags)
	if err != nil {
		return fmt.Errorf("MALLOC3D failed: %w", err)
	}

	// Store address in R0 (convention)
	vm.registers.SetAddress(R0, alloc.Address)

	// Track allocation
	vm.memory.trackAllocation(alloc)

	// Update statistics
	vm.stats.Allocations++
	if alloc.IsYPromoted() {
		vm.stats.YPromotions++
	}

	return nil
}

func (vm *VM3D) executeMFree3D(inst *Instruction3D) error {
	if len(inst.Operands) != 1 {
		return fmt.Errorf("MFREE3D requires 1 operand")
	}

	addr, ok := inst.Operands[0].Value.(types.Address3D)
	if !ok {
		return fmt.Errorf("invalid address operand")
	}

	// Find allocation
	alloc := vm.memory.getAllocation(addr)
	if alloc == nil {
		return fmt.Errorf("allocation not found")
	}

	// Free memory
	if err := vm.allocator.Free(alloc); err != nil {
		return fmt.Errorf("MFREE3D failed: %w", err)
	}

	// Remove from tracking
	vm.memory.untrackAllocation(addr)

	// Clear from cache
	vm.cache.Invalidate(addr)

	return nil
}

func (vm *VM3D) executeMCopy3D(inst *Instruction3D) error {
	if len(inst.Operands) != 3 {
		return fmt.Errorf("MCOPY3D requires 3 operands")
	}

	srcAddr, ok := inst.Operands[0].Value.(types.Address3D)
	if !ok {
		return fmt.Errorf("invalid source address")
	}

	dstAddr, ok := inst.Operands[1].Value.(types.Address3D)
	if !ok {
		return fmt.Errorf("invalid destination address")
	}

	size, ok := inst.Operands[2].Value.(uint32)
	if !ok {
		return fmt.Errorf("invalid size operand")
	}

	// Load source data
	data, err := vm.memory.Load3D(srcAddr, size)
	if err != nil {
		return fmt.Errorf("MCOPY3D failed to load source: %w", err)
	}

	// Store to destination
	if err := vm.memory.Store3D(dstAddr, data); err != nil {
		return fmt.Errorf("MCOPY3D failed to store destination: %w", err)
	}

	// Update caches
	vm.cache.Put(srcAddr, data)
	vm.cache.Put(dstAddr, data)

	return nil
}

func (vm *VM3D) executeMQuery3D(inst *Instruction3D) error {
	if len(inst.Operands) != 3 {
		return fmt.Errorf("MQUERY3D requires 3 operands")
	}

	queryType, ok := inst.Operands[0].Value.(byte)
	if !ok {
		return fmt.Errorf("invalid query type")
	}

	addr, ok := inst.Operands[1].Value.(types.Address3D)
	if !ok {
		return fmt.Errorf("invalid address")
	}

	reg, ok := inst.Operands[2].Value.(Register)
	if !ok {
		return fmt.Errorf("invalid register")
	}

	var result []byte

	switch QueryType(queryType) {
	case QueryFreeMemory:
		free := vm.memory.GetFreeMemory(addr.X)
		result = uint64ToBytes(free)

	case QueryFragmentation:
		frag := vm.memory.GetFragmentation(addr.X)
		// Convert float to bytes
		result = float64ToBytes(frag)

	case QueryAllocationInfo:
		alloc := vm.memory.getAllocation(addr)
		if alloc != nil {
			result = allocInfoToBytes(alloc)
		} else {
			result = make([]byte, 32) // Empty result
		}

	default:
		return fmt.Errorf("unknown query type: %d", queryType)
	}

	vm.registers.Set(reg, result)
	return nil
}

func allocInfoToBytes(alloc *types.Allocation) []byte {
	b := make([]byte, 32) // Fixed size for allocation info

	// Encode allocation metadata into bytes
	// Format: [id(8)][size(4)][flags(1)][isYPromoted(1)][padding(18)]
	binary.BigEndian.PutUint64(b[0:8], alloc.ID)
	binary.BigEndian.PutUint32(b[8:12], alloc.Size)
	b[12] = byte(alloc.Flags)

	if alloc.IsYPromoted() {
		b[13] = 1
	} else {
		b[13] = 0
	}

	// Rest is padding (zeros)
	return b
}


type QueryType byte

const (
	QueryFreeMemory QueryType = iota
	QueryFragmentation
	QueryAllocationInfo
	QueryGasRemaining
	QueryMemoryUsage
)

func (vm *VM3D) executeNOP() error {
	// NOP (No Operation): Does nothing, just consumes 1 gas
	return nil
}

func (vm *VM3D) executeAdd3D(inst *Instruction3D) error {
	if len(inst.Operands) != 3 {
		return fmt.Errorf("ADD3D requires 3 operands")
	}

	destReg, ok := inst.Operands[0].Value.(Register)
	if !ok {
		return fmt.Errorf("invalid destination register operand")
	}

	srcReg1, ok := inst.Operands[1].Value.(Register)
	if !ok {
		return fmt.Errorf("invalid source register 1 operand")
	}

	srcReg2, ok := inst.Operands[2].Value.(Register)
	if !ok {
		return fmt.Errorf("invalid source register 2 operand")
	}

	// Get values from registers
	val1 := vm.registers.GetUint64(srcReg1)
	val2 := vm.registers.GetUint64(srcReg2)

	// Perform addition
	result := val1 + val2

	// Store result in destination register
	vm.registers.SetUint64(destReg, result)

	return nil
}

func (vm *VM3D) executeSub3D(inst *Instruction3D) error {
	if len(inst.Operands) != 3 {
		return fmt.Errorf("SUB3D requires 3 operands")
	}

	destReg, ok := inst.Operands[0].Value.(Register)
	if !ok {
		return fmt.Errorf("invalid destination register operand")
	}

	srcReg1, ok := inst.Operands[1].Value.(Register)
	if !ok {
		return fmt.Errorf("invalid source register 1 operand")
	}

	srcReg2, ok := inst.Operands[2].Value.(Register)
	if !ok {
		return fmt.Errorf("invalid source register 2 operand")
	}

	// Get values from registers
	val1 := vm.registers.GetUint64(srcReg1)
	val2 := vm.registers.GetUint64(srcReg2)

	// Perform subtraction
	result := val1 - val2

	// Store result in destination register
	vm.registers.SetUint64(destReg, result)

	return nil
}

func (vm *VM3D) executeMov3D(inst *Instruction3D) error {
	if len(inst.Operands) != 2 {
		return fmt.Errorf("MOV3D requires 2 operands")
	}

	destReg, ok := inst.Operands[0].Value.(Register)
	if !ok {
		return fmt.Errorf("invalid destination register operand")
	}

	srcReg, ok := inst.Operands[1].Value.(Register)
	if !ok {
		return fmt.Errorf("invalid source register operand")
	}

	// Get value from source register
	data := vm.registers.Get(srcReg)

	// Copy to destination register
	vm.registers.Set(destReg, data)

	return nil
}

func (vm *VM3D) executeMSize3D(inst *Instruction3D) error {
	if len(inst.Operands) != 2 {
		return fmt.Errorf("MSIZE3D requires 2 operands")
	}

	addr, ok := inst.Operands[0].Value.(types.Address3D)
	if !ok {
		return fmt.Errorf("invalid address operand")
	}

	destReg, ok := inst.Operands[1].Value.(Register)
	if !ok {
		return fmt.Errorf("invalid destination register operand")
	}

	// Try to get allocation at address
	alloc := vm.memory.getAllocation(addr)
	if alloc == nil {
		// Address not allocated, return size 0
		vm.registers.SetUint64(destReg, 0)
		return nil
	}

	// Return the allocation size
	vm.registers.SetUint64(destReg, uint64(alloc.Size))

	return nil
}

// Helper functions
func uint64ToBytes(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func bytesToUint64(b []byte) uint64 {
	if len(b) < 8 {
		return 0
	}
	return binary.BigEndian.Uint64(b[:8])
}

func float64ToBytes(f float64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, math.Float64bits(f))
	return b
}
