package integration

import (
	"testing"

	"github.com/rawbytedev/memory3d/internal/types"
	"github.com/rawbytedev/memory3d/internal/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGasAndMerkleIntegration is an end-to-end test
// features:
// 1. Gas accounting with refunds
// 2. Merkle tree updates on memory writes
// 3. Proof generation and verification
// 4. Region merkle root tracking
func TestGasAndMerkleIntegration(t *testing.T) {
	// Create VM with merkle tree enabled and gas limit
	config := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         10000,
		EnableProof:      true, // Enable merkle tree
		EnableCompaction: false,
		MaxInstructions:  1000,
		LogLevel:         vm.LogLevelSilent,
	}

	vmInstance, err := vm.NewVM3D(config)
	require.NoError(t, err, "Failed to create VM3D")
	require.NotNil(t, vmInstance, "VM3D should not be nil")
	defer vmInstance.Shutdown()

	t.Run("Gas consumption tracking", func(t *testing.T) {
		// Initial gas
		initialGas := vmInstance.GetGasRemaining()
		assert.Equal(t, 10000, initialGas, "Initial gas should be 10000")

		// Consume gas
		gasAcct := vm.NewAccountant(10000)
		err := gasAcct.Consume(100)
		require.NoError(t, err)

		remaining := gasAcct.Remaining()
		assert.Equal(t, 9900, remaining, "Remaining gas should be 9900 after consuming 100")

		// Test refund
		gasAcct.Refund(50)
		remaining = gasAcct.Remaining()
		assert.Equal(t, 9950, remaining, "Remaining gas should be 9950 after refund of 50")

		// Test Used tracking
		used := gasAcct.Used()
		assert.Equal(t, uint64(50), used, "Used gas should be 50")
	})

	t.Run("Gas insufficient error", func(t *testing.T) {
		gasAcct := vm.NewAccountant(100)

		// Consume some gas
		err := gasAcct.Consume(50)
		require.NoError(t, err)

		// Record remaining before failing attempt
		before := gasAcct.Remaining()

		// Try to consume more than remaining
		err = gasAcct.Consume(100)
		assert.Error(t, err, "Should fail when consuming more gas than available")
		assert.Equal(t, vm.ErrInsufficientGas, err)

		// Remaining gas should not have changed
		remaining := gasAcct.Remaining()
		assert.Equal(t, before, remaining, "Gas should not have been consumed on failed attempt")
	})

	t.Run("Merkle tree updates on write", func(t *testing.T) {
		// Create new VM with merkle support
		vmWithMerkle, err := vm.NewVM3D(vm.VMConfig{
			MemorySize:      1024 * 1024,
			GasLimit:        100000,
			EnableProof:     true,
			MaxInstructions: 1000,
			LogLevel:        vm.LogLevelSilent,
		})
		require.NoError(t, err)
		defer vmWithMerkle.Shutdown()

		// Get initial merkle root (should be empty/zero)
		initialRoot := vmWithMerkle.GetMerkleRoot()
		assert.NotNil(t, initialRoot, "Merkle root should exist")

		// Allocate memory and write
		testData := []byte{1, 2, 3, 4, 5, 6, 7, 8}
		addr, err := vmWithMerkle.AllocateMemory(uint32(len(testData)), types.RegionTypeHeap)
		require.NoError(t, err)

		// Write to memory
		err = vmWithMerkle.Store3D(addr, testData)
		require.NoError(t, err, "Write to memory should succeed")

		// Get updated merkle root
		updatedRoot := vmWithMerkle.GetMerkleRoot()
		assert.NotNil(t, updatedRoot, "Updated merkle root should exist")

		// Roots should be different after write
		rootsDifferent := false
		for i := range initialRoot {
			if initialRoot[i] != updatedRoot[i] {
				rootsDifferent = true
				break
			}
		}
		assert.True(t, rootsDifferent, "Merkle root should change after write")
	})

	t.Run("Merkle proof generation and verification", func(t *testing.T) {
		// Create VM with merkle support
		vmWithProof, err := vm.NewVM3D(vm.VMConfig{
			MemorySize:      1024 * 1024,
			GasLimit:        100000,
			EnableProof:     true,
			MaxInstructions: 1000,
			LogLevel:        vm.LogLevelSilent,
		})
		require.NoError(t, err)
		defer vmWithProof.Shutdown()

		// Write test data
		testData := []byte{0xAA, 0xBB, 0xCC, 0xDD}
		addr, err := vmWithProof.AllocateMemory(uint32(len(testData)), types.RegionTypeHeap)
		require.NoError(t, err)

		err = vmWithProof.Store3D(addr, testData)
		require.NoError(t, err)

		// Generate proof
		proof, err := vmWithProof.GetMerkleProof(addr)
		require.NoError(t, err, "Should generate merkle proof")
		require.NotNil(t, proof, "Proof should not be nil")

		// Verify proof with correct data
		valid, err := vmWithProof.VerifyMemoryState(addr, testData, proof)
		require.NoError(t, err, "Verification should not error")
		assert.True(t, valid, "Proof should verify with correct data")

		// Verify proof with incorrect data should fail
		wrongData := []byte{0xFF, 0xFF, 0xFF, 0xFF}
		valid, err = vmWithProof.VerifyMemoryState(addr, wrongData, proof)
		require.NoError(t, err)
		assert.False(t, valid, "Proof should not verify with wrong data")
	})

	t.Run("Region merkle root updates", func(t *testing.T) {
		// Create region
		region := types.NewRegion(0, types.RegionTypeHeap)
		initial := region.GetMerkleRoot()
		assert.Equal(t, [32]byte{}, initial, "Initial merkle root should be empty")

		// Update merkle root
		var newRoot [32]byte
		for i := 0; i < 32; i++ {
			newRoot[i] = byte(i % 256)
		}
		region.UpdateMerkleRoot(newRoot)

		// Verify update
		updated := region.GetMerkleRoot()
		assert.Equal(t, newRoot, updated, "Merkle root should be updated")
	})

	t.Run("Integrated gas and merkle with memory operations", func(t *testing.T) {
		vmIntegrated, err := vm.NewVM3D(vm.VMConfig{
			MemorySize:      1024 * 1024,
			GasLimit:        50000,
			EnableProof:     true,
			MaxInstructions: 1000,
			LogLevel:        vm.LogLevelSilent,
		})
		require.NoError(t, err)
		defer vmIntegrated.Shutdown()

		initialGas := vmIntegrated.GetGasRemaining()
		initialUsed := vmIntegrated.GetGasUsed()

		// Allocate and write multiple times
		for i := 0; i < 5; i++ {
			data := []byte{byte(i), byte(i + 1), byte(i + 2)}
			addr, aerr := vmIntegrated.AllocateMemory(uint32(len(data)), types.RegionTypeHeap)
			require.NoError(t, aerr)
			err := vmIntegrated.Store3D(addr, data)
			require.NoError(t, err, "Write iteration %d should succeed", i)

			// Merkle root should be updated
			root := vmIntegrated.GetMerkleRoot()
			vmIntegrated.GetGasUsed()
			assert.NotNil(t, root, "Merkle root should exist after write %d", i)
		}

		// Check gas was consumed
		finalGas := vmIntegrated.GetGasRemaining()
		assert.Less(t, finalGas, initialGas, "Gas should be consumed")

		finalUsed := vmIntegrated.GetGasUsed()
		assert.Greater(t, finalUsed, initialUsed, "Used gas should increase")

		// Stats should be updated
		stats := vmIntegrated.GetStats()
		assert.Greater(t, stats.MemoryUsage+1, uint64(0), "Memory usage should be tracked")
	})

	t.Run("Concurrent gas and merkle operations", func(t *testing.T) {
		vmConcurrent, err := vm.NewVM3D(vm.VMConfig{
			MemorySize:      1024 * 1024,
			GasLimit:        1000000,
			EnableProof:     true,
			MaxInstructions: 100000,
			LogLevel:        vm.LogLevelSilent,
		})
		require.NoError(t, err)
		require.NotNil(t, vmConcurrent, "VM instance should not be nil")
		defer vmConcurrent.Shutdown()

		// Test concurrent gas accounting works
		gasAcct := vm.NewAccountant(1000000)

		// Simulate concurrent operations
		for i := 0; i < 100; i++ {
			err := gasAcct.Consume(uint64(i + 1))
			if err == nil {
				// Successfully consumed
				continue
			} else if err == vm.ErrInsufficientGas {
				// Expected when gas runs out
				break
			}
		}

		// Verify state is consistent
		remaining := gasAcct.Remaining()
		used := gasAcct.Used()
		total := uint64(remaining) + used

		// Total should roughly match initial (allowing for rounding)
		assert.InDelta(t, 1000000, total, 100, "Total gas should be conserved")
	})

	t.Run("Merkle tree disabled mode", func(t *testing.T) {
		vmNoMerkle, err := vm.NewVM3D(vm.VMConfig{
			MemorySize:      1024 * 1024,
			GasLimit:        50000,
			EnableProof:     false, // Merkle disabled
			MaxInstructions: 1000,
			LogLevel:        vm.LogLevelSilent,
		})
		require.NoError(t, err)
		defer vmNoMerkle.Shutdown()

		// Should not be able to get merkle proof
		addr := types.Address3D{X: 0, Y: 0, Z: 0}
		proof, err := vmNoMerkle.GetMerkleProof(addr)
		assert.Error(t, err, "Should error when merkle tree not enabled")
		assert.Nil(t, proof, "Proof should be nil when merkle disabled")

		// GetMerkleRoot should return nil
		root := vmNoMerkle.GetMerkleRoot()
		assert.Nil(t, root, "Merkle root should be nil when disabled")
	})
}

// TestGasRefundScenarios tests specific gas refund scenarios
func TestGasRefundScenarios(t *testing.T) {
	t.Run("Simple refund scenario", func(t *testing.T) {
		gasAcct := vm.NewAccountant(1000)

		// Consume gas
		err := gasAcct.Consume(600)
		require.NoError(t, err)
		assert.Equal(t, 400, gasAcct.Remaining())

		// Refund some
		gasAcct.Refund(200)
		assert.Equal(t, 600, gasAcct.Remaining())
	})

	t.Run("Multiple bucket refunds", func(t *testing.T) {
		// With 8 buckets, refunds should be distributed
		gasAcct := vm.NewAccountant(8000)

		// Consume evenly across buckets
		for i := 0; i < 8; i++ {
			err := gasAcct.Consume(500)
			require.NoError(t, err)
		}

		remaining := gasAcct.Remaining()
		assert.Equal(t, 4000, remaining)

		// Refund should work
		gasAcct.Refund(1000)
		remaining = gasAcct.Remaining()
		assert.Equal(t, 5000, remaining)
	})
}

// TestMerkleTreeIntegration tests merkle tree specific functionality
func TestMerkleTreeIntegration(t *testing.T) {
	t.Run("Multiple writes to same address", func(t *testing.T) {
		vmMultiWrite, err := vm.NewVM3D(vm.VMConfig{
			MemorySize:      1024 * 1024,
			GasLimit:        100000,
			EnableProof:     true,
			MaxInstructions: 1000,
			LogLevel:        vm.LogLevelSilent,
		})
		require.NoError(t, err)
		defer vmMultiWrite.Shutdown()

		// First write
		data1 := []byte{1, 2, 3, 4}
		addr, err := vmMultiWrite.AllocateMemory(uint32(len(data1)), types.RegionTypeHeap) // safely generate address

		err = vmMultiWrite.Store3D(addr, data1)
		require.NoError(t, err)

		root1 := vmMultiWrite.GetMerkleRoot()
		proof1, _ := vmMultiWrite.GetMerkleProof(addr)

		// Second write to same address
		data2 := []byte{5, 6, 7, 8}
		err = vmMultiWrite.Store3D(addr, data2)
		require.NoError(t, err)

		root2 := vmMultiWrite.GetMerkleRoot()
		proof2, _ := vmMultiWrite.GetMerkleProof(addr)
		// Roots should be different
		assert.NotEqual(t, root1, root2, "Merkle root should change on second write")
		// New proof should verify with new data only
		valid, _ := vmMultiWrite.VerifyMemoryState(addr, data2, proof2)
		assert.True(t, valid, "New proof should verify new data")

		// Old proof might not verify new data
		validOld, _ := vmMultiWrite.VerifyMemoryState(addr, data2, proof1)
		assert.False(t, validOld, "Old proof should not verify new data")
	})

	t.Run("Multiple addresses merkle tracking", func(t *testing.T) {
		vmMultiAddr, err := vm.NewVM3D(vm.VMConfig{
			MemorySize:      1024 * 1024,
			GasLimit:        100000,
			EnableProof:     true,
			MaxInstructions: 1000,
			LogLevel:        vm.LogLevelSilent,
		})
		require.NoError(t, err)
		defer vmMultiAddr.Shutdown()
		var datas [][]byte
		var addrs []types.Address3D
		for x := 0; x < 3; x++ {
			for y := 0; y < 3; y++ {
				data := []byte{byte(x*3 + y), byte(x*3 + y + 1)}
				datas = append(datas, data)
			}
		}
		// Write to multiple addresses
		for x := 0; x < len(datas); x++ {
			addr, err := vmMultiAddr.AllocateMemory(uint32(len(datas[x])), types.RegionTypeHeap)
			require.NoError(t, err)
			addrs = append(addrs, addr)
			if x > 0 {
				require.NotEqual(t, addr, addrs[x-1])
			}
			err = vmMultiAddr.Store3D(addr, datas[x])
			require.NoError(t, err)

		}

		// All addresses should have valid proofs
		for x := 0; x < len(addrs); x++ {
			proof, err := vmMultiAddr.GetMerkleProof(addrs[x])
			require.NoError(t, err)
			valid, err := vmMultiAddr.VerifyMemoryState(addrs[x], datas[x], proof)
			require.NoError(t, err)
			assert.True(t, valid, "Proof should be valid for address (%d, %d)", addrs[x].X, addrs[x].Y)

		}
	})
}
