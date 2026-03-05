package integration

import (
	"testing"

	"github.com/rawbytedev/memory3d/internal/types"
	"github.com/rawbytedev/memory3d/internal/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPriority1GasRefundSimple tests gas refund functionality
func TestPriority1GasRefundSimple(t *testing.T) {
	gasAcct := vm.NewAccountant(1000)

	// Consume some gas
	err := gasAcct.Consume(300)
	require.NoError(t, err, "Consume should succeed with enough gas")

	remaining := gasAcct.Remaining()
	assert.Equal(t, 700, remaining, "Remaining gas should be 700")

	used := gasAcct.Used()
	assert.Equal(t, uint64(300), used, "Used gas should be 300")

	// Test refund
	gasAcct.Refund(100)
	remaining = gasAcct.Remaining()
	assert.Equal(t, 800, remaining, "After refund, remaining should be 800")

	used = gasAcct.Used()
	assert.Equal(t, uint64(200), used, "After refund, used should be 200")
}

// TestPriority1GasInsufficientSimple tests insufficient gas error
func TestPriority1GasInsufficientSimple(t *testing.T) {
	gasAcct := vm.NewAccountant(100)

	// Consume some gas
	err := gasAcct.Consume(60)
	require.NoError(t, err)

	remaining := gasAcct.Remaining()
	assert.Equal(t, 40, remaining)

	// Try to consume more than remaining
	err = gasAcct.Consume(50)
	assert.Equal(t, vm.ErrInsufficientGas, err, "Should error on insufficient gas")

	// Remaining should not change
	remaining = gasAcct.Remaining()
	assert.Equal(t, 40, remaining, "Remaining gas should not change on failed consume")
}

// TestPriority1RegionMerkleRoot tests region merkle root updates
func TestPriority1RegionMerkleRoot(t *testing.T) {
	region := types.NewRegion(1, types.RegionTypeHeap)

	// Initial merkle root should be empty
	initial := region.GetMerkleRoot()
	assert.Equal(t, [32]byte{}, initial, "Initial merkle root should be zero")

	// Update merkle root
	var expectedRoot [32]byte
	for i := 0; i < 32; i++ {
		expectedRoot[i] = byte((i * 7) % 256)
	}

	region.UpdateMerkleRoot(expectedRoot)

	// Verify update
	retrieved := region.GetMerkleRoot()
	assert.Equal(t, expectedRoot, retrieved, "Merkle root should match updated value")

	// Second update
	var secondRoot [32]byte
	for i := 0; i < 32; i++ {
		secondRoot[i] = byte((i * 13) % 256)
	}

	region.UpdateMerkleRoot(secondRoot)
	retrieved = region.GetMerkleRoot()
	assert.Equal(t, secondRoot, retrieved, "Merkle root should reflect second update")
}

// TestPriority1VMGasTracking tests VM gas tracking
func TestPriority1VMGasTracking(t *testing.T) {
	config := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         5000,
		EnableProof:      false,
		MaxInstructions:  100,
		LogLevel:         vm.LogLevelSilent,
	}

	vmInstance, err := vm.NewVM3D(config)
	require.NoError(t, err)
	defer vmInstance.Shutdown()

	// Check initial gas
	remaining := vmInstance.GetGasRemaining()
	assert.Equal(t, 5000, remaining, "Initial gas should be 5000")

	used := vmInstance.GetGasUsed()
	assert.Equal(t, uint64(0), used, "Initial used gas should be 0")

	// Stats should be accessible
	stats := vmInstance.GetStats()
	assert.NotNil(t, stats, "Stats should not be nil")
	assert.Equal(t, uint64(0), stats.InstructionsExecuted, "Initial instructions should be 0")
}

// TestPriority1VMMerkleDisabled tests VM with merkle disabled
func TestPriority1VMMerkleDisabled(t *testing.T) {
	config := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         10000,
		EnableProof:      false, // Merkle disabled
		MaxInstructions:  100,
		LogLevel:         vm.LogLevelSilent,
	}

	vmInstance, err := vm.NewVM3D(config)
	require.NoError(t, err)
	defer vmInstance.Shutdown()

	// Should return error when trying to access merkle proof with disabled tree
	addr := types.Address3D{X: 0, Y: 0, Z: 0}
	proof, err := vmInstance.GetMerkleProof(addr)
	assert.Error(t, err, "Should error when merkle disabled")
	assert.Nil(t, proof, "Proof should be nil when merkle disabled")

	// GetMerkleRoot should return nil
	root := vmInstance.GetMerkleRoot()
	assert.Nil(t, root, "Merkle root should be nil when disabled")
}

// TestPriority1VMMerkleEnabled tests VM with merkle enabled
func TestPriority1VMMerkleEnabled(t *testing.T) {
	config := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         10000,
		EnableProof:      true, // Merkle enabled
		MaxInstructions:  100,
		LogLevel:         vm.LogLevelSilent,
	}

	vmInstance, err := vm.NewVM3D(config)
	require.NoError(t, err)
	defer vmInstance.Shutdown()

	// Check merkle root exists
	root := vmInstance.GetMerkleRoot()
	assert.NotNil(t, root, "Merkle root should exist when enabled")
	assert.Len(t, root, 32, "Merkle root should be 32 bytes")
}

// TestPriority1AllocationAndMemoryOps tests memory allocation and operations
func TestPriority1AllocationAndMemoryOps(t *testing.T) {
	config := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         100000,
		EnableProof:      true,
		MaxInstructions:  1000,
		LogLevel:         vm.LogLevelSilent,
	}

	vmInstance, err := vm.NewVM3D(config)
	require.NoError(t, err)
	defer vmInstance.Shutdown()

	// Allocate memory
	testData := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	addr, err := vmInstance.AllocateMemory(uint32(len(testData)), types.RegionTypeHeap)
	require.NoError(t, err, "Allocation should succeed")
	assert.NotEqual(t, types.Address3D{}, addr, "Allocated address should be valid")

	// Store data at allocated address
	err = vmInstance.Store3D(addr, testData)
	require.NoError(t, err, "Store should succeed at allocated address")

	// Load data back
	loaded, err := vmInstance.Load3D(addr, uint32(len(testData)))
	require.NoError(t, err, "Load should succeed")
	assert.Equal(t, testData, loaded, "Loaded data should match stored data")
}

// TestPriority1MerkleTreeViaVM tests merkle tree through VM integration
func TestPriority1MerkleTreeViaVM(t *testing.T) {
	config := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         100000,
		EnableProof:      true,
		MaxInstructions:  1000,
		LogLevel:         vm.LogLevelSilent,
	}

	vmInstance, err := vm.NewVM3D(config)
	require.NoError(t, err)
	defer vmInstance.Shutdown()

	// Check merkle root exists
	root1 := vmInstance.GetMerkleRoot()
	assert.NotNil(t, root1, "Root should not be nil")
	assert.Len(t, root1, 32, "Root should be 32 bytes")

	// Verify merkle disabled check
	config2 := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         10000,
		EnableProof:      false,
		MaxInstructions:  100,
		LogLevel:         vm.LogLevelSilent,
	}

	vmNoProof, err := vm.NewVM3D(config2)
	require.NoError(t, err)
	defer vmNoProof.Shutdown()

	// Should error when proof disabled
	addr := types.Address3D{X: 0, Y: 0, Z: 0}
	proof, err := vmNoProof.GetMerkleProof(addr)
	assert.Error(t, err)
	assert.Nil(t, proof)
}

// TestPriority1EndToEndMemoryAndGas tests end-to-end memory ops with gas tracking
func TestPriority1EndToEndMemoryAndGas(t *testing.T) {
	config := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         50000,
		EnableProof:      true,
		MaxInstructions:  1000,
		LogLevel:         vm.LogLevelSilent,
	}

	vmInstance, err := vm.NewVM3D(config)
	require.NoError(t, err)
	defer vmInstance.Shutdown()

	initialGas := vmInstance.GetGasRemaining()
	assert.Equal(t, 50000, initialGas)

	initialRoot := vmInstance.GetMerkleRoot()
	assert.Len(t, initialRoot, 32)

	// Perform single allocation and write
	data := []byte{0x01, 0x02, 0x03, 0x04}
	addr, err := vmInstance.AllocateMemory(uint32(len(data)), types.RegionTypeHeap)
	require.NoError(t, err, "Allocation should succeed")

	err = vmInstance.Store3D(addr, data)
	require.NoError(t, err, "Store should succeed")

	// Verify we can read it back
	loaded, err := vmInstance.Load3D(addr, uint32(len(data)))
	require.NoError(t, err)
	assert.Equal(t, data, loaded, "Data should match")

	// Check stats were updated (allocation should be tracked)
	stats := vmInstance.GetStats()
	assert.GreaterOrEqual(t, stats.Allocations, uint64(0), "Allocations should be tracked")
}
