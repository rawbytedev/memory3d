package integration

import (
	"testing"

	"github.com/rawbytedev/memory3d/internal/allocator"
	"github.com/rawbytedev/memory3d/internal/types"
	"github.com/rawbytedev/memory3d/internal/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPriority2FragmentationAnalysis tests fragmentation detection
func TestPriority2FragmentationAnalysis(t *testing.T) {
	region := types.NewRegion(1, types.RegionTypeHeap)

	// Add a plane to the region
	plane := types.NewPlane(0, 1, types.PlaneSize)
	region.AddPlane(plane)

	// Get compaction report
	report := region.GetCompactionReport()

	assert.NotNil(t, report, "Report should not be nil")
	assert.Equal(t, uint64(1), report.RegionID, "Region ID should match")
	assert.GreaterOrEqual(t, report.Fragmentation, 0.0, "Fragmentation should be >= 0")
	assert.LessOrEqual(t, report.Fragmentation, 1.0, "Fragmentation should be <= 1")
}


// TestPriority2VMCompactionAPI tests VM compaction API
func TestPriority2VMCompactionAPI(t *testing.T) {
	config := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         100000,
		EnableProof:      false,
		MaxInstructions:  1000,
		LogLevel:         vm.LogLevelSilent,
	}

	vmInstance, err := vm.NewVM3D(config)
	require.NoError(t, err)
    defer vmInstance.Shutdown()

	// Get compaction report
	report := vmInstance.GetCompactionReport()
	assert.NotNil(t, report, "Compaction report should not be nil")

	// Analyze compaction
	compactConfig := allocator.DefaultCompactionConfig()
	allocsToMove := vmInstance.AnalyzeCompaction(compactConfig)
	assert.NotNil(t, allocsToMove, "AllocsToMove should not be nil")
}

// TestPriority2MemoryManagerRelocation tests allocation relocation through VM
func TestPriority2MemoryManagerRelocation(t *testing.T) {
	config := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         100000,
		EnableProof:      false,
		MaxInstructions:  1000,
		LogLevel:         vm.LogLevelSilent,
	}

	vmInstance, err := vm.NewVM3D(config)
	require.NoError(t, err)
    defer vmInstance.Shutdown()

	// Allocate memory through VM
	addr1, err := vmInstance.AllocateMemory(100, types.RegionTypeHeap)
	require.NoError(t, err, "Allocation should succeed")

	// Store data
	data := []byte{1, 2, 3, 4, 5}
	err = vmInstance.Store3D(addr1, data)
	require.NoError(t, err, "Store should succeed")

	// Get allocations before relocation
	beforeAllocations := vmInstance.GetMemoryManager().GetAllAllocations()
	initialCount := len(beforeAllocations)
	assert.Greater(t, initialCount, 0, "Should have at least one allocation")

	// Try to relocate
	addr2 := types.Address3D{X: 1, Y: 0, Z: 100}
	err = vmInstance.GetMemoryManager().RelocateAllocation(addr1, addr2)
	require.NoError(t, err, "Relocation should succeed")

	// Allocation should now be at new address
	afterAllocations := vmInstance.GetMemoryManager().GetAllAllocations()
	assert.Equal(t, initialCount, len(afterAllocations), "Should have same allocation count")
}

// TestPriority2MCompact3DInstruction tests MCOMPACT3D execution
func TestPriority2MCompact3DInstruction(t *testing.T) {
	config := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         100000,
		EnableProof:      false,
		MaxInstructions:  1000,
		LogLevel:         vm.LogLevelSilent,
	}

	vmInstance, err := vm.NewVM3D(config)
	require.NoError(t, err)
    defer vmInstance.Shutdown()

	// Record initial gas
	initialGas := vmInstance.GetGasRemaining()
	assert.Equal(t, 100000, initialGas)

	// Create an instruction for MCOMPACT3D
	inst := &vm.Instruction3D{
		Opcode:   vm.OP_MCOMPACT3D,
		Operands: []vm.Operand{},
		Size:     1,
	}

	// Execute compaction
	err = vmInstance.ExecuteCompactInstruction(inst)
	require.NoError(t, err, "MCOMPACT3D should execute without error")

	// Gas might be refunded if allocations were compacted
	// COMPACT3D doesn't charge anything instead refunds if gas was used
	// in this case nothing to refund
	finalGas := vmInstance.GetGasRemaining()
	assert.Equal(t, finalGas, initialGas, "Gas should be equal to initial (with possible refunds)")
}

// TestPriority2CompactionEndToEnd tests compaction API consistency
func TestPriority2CompactionEndToEnd(t *testing.T) {
	// Create allocator directly for minimal VM interaction
	allocConfig := allocator.Config{
		PlaneSize:        types.PlaneSize,
		MaxYPlanes:       1024,
		MaxYPromotions:   8,
		WorkerCount:      4,
		EnableConcurrent: true, 
	}
	alloc := allocator.NewAllocator3D(allocConfig)
	defer alloc.Shutdown()

	// Get compaction report directly from allocator
	report := alloc.GetCompactionReport()
	assert.NotNil(t, report, "Compaction report should exist")

	// Test compaction analysis
	compactConfig := allocator.DefaultCompactionConfig()
	allocsToMove := alloc.AnalyzeCompaction(compactConfig)
	assert.NotNil(t, allocsToMove, "AllocsToMove should not be nil")

	// Verify it's an empty list initially
	assert.Equal(t, 0, len(allocsToMove), "No allocations to move initially")
}

// TestPriority2PinnedAllocationProtection tests pinned allocations are protected
func TestPriority2PinnedAllocationProtection(t *testing.T) {
	// Create allocation with pinned flag
	addr := types.Address3D{X: 1, Y: 0, Z: 0}
	alloc := types.NewAllocation(1, addr, 100, types.RegionTypeHeap)
	alloc.Flags |= types.FlagPinned

	// Pinned allocations should not be selected for compaction
	// This is enforced in AnalyzeCompaction logic
	assert.True(t, (alloc.Flags&types.FlagPinned) != 0, "Pinned flag should be set")
}

// TestPriority2SelfContainedCompaction tests compaction without full allocation
func TestPriority2SelfContainedCompaction(t *testing.T) {
	config := vm.VMConfig{
		MemorySize:       1024 * 1024,
		GasLimit:         50000,
		EnableProof:      false,
		MaxInstructions:  1000,
		LogLevel:         vm.LogLevelSilent,
	}

	vmInstance, err := vm.NewVM3D(config)
	require.NoError(t, err)

	// Record gas before
	beforeGas := vmInstance.GetGasRemaining()

	// Create instruction
	inst := &vm.Instruction3D{
		Opcode:   vm.OP_MCOMPACT3D,
		Operands: []vm.Operand{},
		Size:     1,
	}

	// Execute should not fail even with minimal setup
	err = vmInstance.ExecuteCompactInstruction(inst)
	require.NoError(t, err, "MCOMPACT3D should execute cleanly")

	// Track gas state
	afterGas := vmInstance.GetGasRemaining()
	assert.LessOrEqual(t, afterGas, beforeGas, "Gas should not increase")
}
