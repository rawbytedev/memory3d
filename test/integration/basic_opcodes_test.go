package integration

import (
	"encoding/binary"
	"testing"

	"github.com/rawbytedev/memory3d/internal/types"
	"github.com/rawbytedev/memory3d/internal/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBasicOpcodeNOP tests the NOP instruction
func TestBasicOpcodeNOP(t *testing.T) {
	vmConfig := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         10000,
		EnableProof:      false,
		EnableCompaction: false,
		MaxInstructions:  1000,
	}
	vmInstance, err := vm.NewVM3D(vmConfig)
	require.NoError(t, err)
	defer vmInstance.Shutdown()

	// Create a program with NOP instruction
	program := []byte{
		0x00, // OP_NOP
	}

	// Execute the program
	err = vmInstance.Execute(program)
	require.NoError(t, err)

	// Verify stats
	stats := vmInstance.GetStats()
	assert.Equal(t, uint64(1), stats.InstructionsExecuted)

	// Verify gas was consumed
	assert.Less(t, vmInstance.GetGasRemaining(), 10000)
}

// TestBasicOpcodeHALT tests the HALT instruction
func TestBasicOpcodeHALT(t *testing.T) {
	vmConfig := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         10000,
		EnableProof:      false,
		EnableCompaction: false,
		MaxInstructions:  1000,
	}
	vmInstance, err := vm.NewVM3D(vmConfig)
	require.NoError(t, err)
	defer vmInstance.Shutdown()

	// Create a program with NOP and HALT
	program := []byte{
		0x00, // OP_NOP
		0xFF, // OP_HALT3D
		0x00, // OP_NOP (should not execute)
	}

	// Execute the program
	err = vmInstance.Execute(program)
	require.NoError(t, err)

	// Verify only 2 instructions executed
	stats := vmInstance.GetStats()
	assert.Equal(t, uint64(2), stats.InstructionsExecuted)
}

// TestBasicOpcodeAdd builds a program with ADD3D instruction
func TestBasicOpcodeAdd(t *testing.T) {
	vmConfig := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         10000,
		EnableProof:      false,
		EnableCompaction: false,
		MaxInstructions:  1000,
	}
	vmInstance, err := vm.NewVM3D(vmConfig)
	require.NoError(t, err)
	defer vmInstance.Shutdown()

	// Set up initial values in registers
	vmInstance.GetRegisters().SetUint64(vm.R0, 100)
	vmInstance.GetRegisters().SetUint64(vm.R1, 200)

	// Create a minimal program with ADD3D
	program := buildAddProgram(vm.R2, vm.R0, vm.R1)

	err = vmInstance.Execute(program)
	require.NoError(t, err)

	// Verify the result
	result := vmInstance.GetRegisters().GetUint64(vm.R2)
	assert.Equal(t, uint64(300), result, "ADD3D should compute 100 + 200 = 300")

	stats := vmInstance.GetStats()
	assert.Equal(t, uint64(1), stats.InstructionsExecuted)
}

// TestBasicOpcodeSub tests SUB3D instruction
func TestBasicOpcodeSub(t *testing.T) {
	vmConfig := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         10000,
		EnableProof:      false,
		EnableCompaction: false,
		MaxInstructions:  1000,
	}
	vmInstance, err := vm.NewVM3D(vmConfig)
	require.NoError(t, err)
	defer vmInstance.Shutdown()

	// Set initial values
	vmInstance.GetRegisters().SetUint64(vm.R0, 500)
	vmInstance.GetRegisters().SetUint64(vm.R1, 200)

	// Create program with SUB3D
	program := buildSubProgram(vm.R2, vm.R0, vm.R1)

	err = vmInstance.Execute(program)
	require.NoError(t, err)

	// Verify the result
	result := vmInstance.GetRegisters().GetUint64(vm.R2)
	assert.Equal(t, uint64(300), result, "SUB3D should compute 500 - 200 = 300")

	stats := vmInstance.GetStats()
	assert.Equal(t, uint64(1), stats.InstructionsExecuted)
}

// TestBasicOpcodeMov tests MOV3D instruction
func TestBasicOpcodeMov(t *testing.T) {
	vmConfig := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         10000,
		EnableProof:      false,
		EnableCompaction: false,
		MaxInstructions:  1000,
	}
	vmInstance, err := vm.NewVM3D(vmConfig)
	require.NoError(t, err)
	defer vmInstance.Shutdown()

	// Set source value
	sourceData := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	vmInstance.GetRegisters().Set(vm.R0, sourceData)

	// Create MOV program
	program := buildMovProgram(vm.R1, vm.R0)

	err = vmInstance.Execute(program)
	require.NoError(t, err)

	// Verify the move
	destData := vmInstance.GetRegisters().Get(vm.R1)
	assert.Equal(t, sourceData, destData, "MOV3D should copy R0 to R1")

	stats := vmInstance.GetStats()
	assert.Equal(t, uint64(1), stats.InstructionsExecuted)
}

// TestBasicOpcodeMSize tests MSIZE3D instruction
func TestBasicOpcodeMSize(t *testing.T) {
	vmConfig := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         10000,
		EnableProof:      false,
		EnableCompaction: false,
		MaxInstructions:  1000,
	}
	vmInstance, err := vm.NewVM3D(vmConfig)
	require.NoError(t, err)
	defer vmInstance.Shutdown()

	// Allocate memory first
	addr, err := vmInstance.AllocateMemory(1024, types.RegionTypeHeap)
	require.NoError(t, err)

	// Create MSIZE program to query allocation size
	program := buildMSizeProgram(addr, vm.R0)

	err = vmInstance.Execute(program)
	require.NoError(t, err)

	// Verify the size
	size := vmInstance.GetRegisters().GetUint64(vm.R0)
	assert.Equal(t, uint64(1024), size, "MSIZE3D should return allocation size")

	stats := vmInstance.GetStats()
	assert.Equal(t, uint64(1), stats.InstructionsExecuted)
}

// TestMSizeNonAllocated tests MSIZE3D for non-allocated address
func TestMSizeNonAllocated(t *testing.T) {
	vmConfig := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         10000,
		EnableProof:      false,
		EnableCompaction: false,
		MaxInstructions:  1000,
	}
	vmInstance, err := vm.NewVM3D(vmConfig)
	require.NoError(t, err)
	defer vmInstance.Shutdown()

	// Create MSIZE program for unallocated address
	unallocatedAddr := types.Address3D{X: 100, Y: 100, Z: 100}
	program := buildMSizeProgram(unallocatedAddr, vm.R0)

	err = vmInstance.Execute(program)
	require.NoError(t, err)

	// Verify size is 0
	size := vmInstance.GetRegisters().GetUint64(vm.R0)
	assert.Equal(t, uint64(0), size, "MSIZE3D should return 0 for unallocated address")
}

// TestAddSubChained tests multiple arithmetic operations
func TestAddSubChained(t *testing.T) {
	vmConfig := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         10000,
		EnableProof:      false,
		EnableCompaction: false,
		MaxInstructions:  1000,
	}
	vmInstance, err := vm.NewVM3D(vmConfig)
	require.NoError(t, err)
	defer vmInstance.Shutdown()

	// Set initial values: R0=100, R1=50, R2=10
	vmInstance.GetRegisters().SetUint64(vm.R0, 100)
	vmInstance.GetRegisters().SetUint64(vm.R1, 50)
	vmInstance.GetRegisters().SetUint64(vm.R2, 10)

	// Create program: R3 = R0 + R1 = 150, then R4 = R3 - R2 = 140
	program := buildChainedArithmeticProgram()

	err = vmInstance.Execute(program)
	require.NoError(t, err)

	// Verify results
	r3 := vmInstance.GetRegisters().GetUint64(vm.R3)
	r4 := vmInstance.GetRegisters().GetUint64(vm.R4)

	assert.Equal(t, uint64(150), r3, "R3 should be 100 + 50")
	assert.Equal(t, uint64(140), r4, "R4 should be 150 - 10")

	stats := vmInstance.GetStats()
	assert.Equal(t, uint64(2), stats.InstructionsExecuted)
}

// TestMOVChained tests multiple move operations
func TestMOVChained(t *testing.T) {
	vmConfig := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         10000,
		EnableProof:      false,
		EnableCompaction: false,
		MaxInstructions:  1000,
	}
	vmInstance, err := vm.NewVM3D(vmConfig)
	require.NoError(t, err)
	defer vmInstance.Shutdown()

	// Set initial value
	originalData := []byte{0x42, 0x13, 0x37}
	vmInstance.GetRegisters().Set(vm.R0, originalData)

	// Create program: R1 = R0, then R2 = R1 (chained moves)
	program := buildChainedMovProgram()

	err = vmInstance.Execute(program)
	require.NoError(t, err)

	// Verify all registers contain the same data
	r0Data := vmInstance.GetRegisters().Get(vm.R0)
	r1Data := vmInstance.GetRegisters().Get(vm.R1)
	r2Data := vmInstance.GetRegisters().Get(vm.R2)

	assert.Equal(t, originalData, r0Data)
	assert.Equal(t, originalData, r1Data, "R1 should equal R0")
	assert.Equal(t, originalData, r2Data, "R2 should equal R1")

	stats := vmInstance.GetStats()
	assert.Equal(t, uint64(2), stats.InstructionsExecuted)
}

// Helper functions to build test programs

func buildAddProgram(destReg, src1Reg, src2Reg vm.Register) []byte {
	// [opcode(1)][destReg(1)][srcReg1(1)][srcReg2(1)]
	return []byte{
		byte(vm.OP_ADD3D),
		byte(destReg),
		byte(src1Reg),
		byte(src2Reg),
	}
}

func buildSubProgram(destReg, src1Reg, src2Reg vm.Register) []byte {
	// [opcode(1)][destReg(1)][srcReg1(1)][srcReg2(1)]
	return []byte{
		byte(vm.OP_SUB3D),
		byte(destReg),
		byte(src1Reg),
		byte(src2Reg),
	}
}

func buildMovProgram(destReg, srcReg vm.Register) []byte {
	// [opcode(1)][destReg(1)][srcReg(1)][padding(5)]
	program := make([]byte, 8)
	program[0] = byte(vm.OP_MOV3D)
	program[1] = byte(destReg)
	program[2] = byte(srcReg)
	return program
}

func buildMSizeProgram(addr types.Address3D, destReg vm.Register) []byte {
	// [opcode(1)][x(8)][y(4)][z(2)][destReg(1)]
	program := make([]byte, 16)
	program[0] = byte(vm.OP_MSIZE3D)
	binary.BigEndian.PutUint64(program[1:9], addr.X)
	binary.BigEndian.PutUint32(program[9:13], addr.Y)
	binary.BigEndian.PutUint16(program[13:15], addr.Z)
	program[15] = byte(destReg)
	return program
}

func buildChainedArithmeticProgram() []byte {
	// R3 = R0 + R1
	// R4 = R3 - R2
	addProgram := buildAddProgram(vm.R3, vm.R0, vm.R1)
	subProgram := buildSubProgram(vm.R4, vm.R3, vm.R2)
	return append(addProgram, subProgram...)
}

func buildChainedMovProgram() []byte {
	// R1 = R0
	// R2 = R1
	mov1 := buildMovProgram(vm.R1, vm.R0)
	mov2 := buildMovProgram(vm.R2, vm.R1)
	return append(mov1, mov2...)
}
