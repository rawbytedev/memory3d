package integration

import (
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/rawbytedev/memory3d/internal/types"
	"github.com/rawbytedev/memory3d/internal/vm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFullStackRealWorldScenario simulates a real-world use case with multiple
// operations: memory allocation, arithmetic, data storage, and size queries
// This test ensures the VM can handle practical workloads
func TestFullStackRealWorldScenario(t *testing.T) {
	t.Run("Complex_workflow_with_memory_and_arithmetic", func(t *testing.T) {
		// Create VM with realistic configuration
		vmConfig := vm.VMConfig{
			MemorySize:       1024 * 1024,      // 1MB
			GasLimit:         500000,           // 500K gas
			EnableProof:      true,             // Enable proofs for security
			EnableCompaction: true,             // Enable memory compaction
			MaxInstructions:  5000,             // 5K instructions
		}
		vmInstance, err := vm.NewVM3D(vmConfig)
		require.NoError(t, err)
		defer vmInstance.Shutdown()

		gasAtStart := vmInstance.GetGasRemaining()

		// Scenario: Calculate and store computation results
		// 1. Allocate result storage (1KB)
		resultAddr, err := vmInstance.AllocateMemory(1024, types.RegionTypeHeap)
		require.NoError(t, err)
		require.NotNil(t, resultAddr)

		// 2. Store initial data in memory
		initialData := []byte("Configuration data for processing")
		err = vmInstance.Store3D(resultAddr, initialData)
		require.NoError(t, err)

		// 3. Verify stored data
		retrievedData, err := vmInstance.Load3D(resultAddr, uint32(len(initialData)))
		require.NoError(t, err)
		assert.Equal(t, initialData, retrievedData, "Stored and retrieved data should match")

		// 4. Build computation program (arithmetic operations)
		program := buildComplexComputationProgram()

		// Set initial values for computation
		vmInstance.GetRegisters().SetUint64(vm.R0, 100)
		vmInstance.GetRegisters().SetUint64(vm.R1, 50)

		err = vmInstance.Execute(program)
		require.NoError(t, err)

		// 5. Verify arithmetic results
		r2Result := vmInstance.GetRegisters().GetUint64(vm.R2)
		assert.Equal(t, uint64(150), r2Result, "R2 should be 100 + 50 = 150")

		r3Result := vmInstance.GetRegisters().GetUint64(vm.R3)
		assert.Equal(t, uint64(100), r3Result, "R3 should be 150 - 50 = 100")

		// 6. Check statistics
		stats := vmInstance.GetStats()
		assert.Greater(t, stats.InstructionsExecuted, uint64(0), "Should execute instructions")
		assert.Greater(t, vmInstance.GetGasUsed(), uint64(0), "Should consume some gas")

		gasAtEnd := vmInstance.GetGasRemaining()
		assert.Less(t, gasAtEnd, gasAtStart, "Gas should be consumed")

		t.Logf("Instructions executed: %d", stats.InstructionsExecuted)
		t.Logf("Gas used: %d", vmInstance.GetGasUsed())
		t.Logf("Memory usage: %d bytes", stats.MemoryUsage)
	})

	t.Run("Multi_allocation_workflow", func(t *testing.T) {
		vmConfig := vm.VMConfig{
			MemorySize:      1024 * 1024,
			GasLimit:        1000000,
			EnableProof:     true,
			EnableCompaction: true,
			MaxInstructions: 5000,
		}
		vmInstance, err := vm.NewVM3D(vmConfig)
		require.NoError(t, err)
		defer vmInstance.Shutdown()

		// Allocate multiple blocks
		blockSize := uint32(256)
		blockCount := 5
		addresses := make([]types.Address3D, blockCount)

		for i := 0; i < blockCount; i++ {
			addr, err := vmInstance.AllocateMemory(blockSize, types.RegionTypeHeap)
			require.NoError(t, err)
			addresses[i] = addr
		}

		// Write data to each block
		for i, addr := range addresses {
			data := fmt.Appendf(nil, "Block %d data content here", i)
			err := vmInstance.Store3D(addr, data)
			require.NoError(t, err)
		}

		// Read and verify all blocks
		for i, addr := range addresses {
			data, err := vmInstance.Load3D(addr, blockSize)
			require.NoError(t, err)
			assert.NotEmpty(t, data)
			assert.Contains(t, string(data), fmt.Sprintf("Block %d", i))
		}

		// Verify statistics
		stats := vmInstance.GetStats()
		assert.Greater(t, vmInstance.GetGasUsed(), uint64(0))
		assert.Greater(t, vmInstance.GetGasUsed(), uint64(0))
		t.Logf("Allocations: %d, Gas used: %d", stats.Allocations, vmInstance.GetGasUsed())
	})

	t.Run("Mixed_operations_sequence", func(t *testing.T) {
		vmConfig := vm.VMConfig{
			MemorySize:      1024 * 1024,
			GasLimit:        500000,
			EnableProof:     false,
			EnableCompaction: true,
			MaxInstructions: 10000,
		}
		vmInstance, err := vm.NewVM3D(vmConfig)
		require.NoError(t, err)
		defer vmInstance.Shutdown()

		// Phase 1: Memory operations
		addr1, _ := vmInstance.AllocateMemory(512, types.RegionTypeHeap)
		data1 := []byte("First data block")
		vmInstance.Store3D(addr1, data1)

		addr2, _ := vmInstance.AllocateMemory(512, types.RegionTypeHeap)
		data2 := []byte("Second data block")
		vmInstance.Store3D(addr2, data2)

		// Phase 2: Execute arithmetic operations
		// Set values for computation
		vmInstance.GetRegisters().SetUint64(vm.R0, 10)
		vmInstance.GetRegisters().SetUint64(vm.R1, 20)
		vmInstance.GetRegisters().SetUint64(vm.R2, 100)

		program := buildSequentialArithmeticProgram()
		err = vmInstance.Execute(program)
		require.NoError(t, err)

		// Phase 3: Verify data persistence through arithmetic operations
		read1, _ := vmInstance.Load3D(addr1, uint32(len(data1)))
		read2, _ := vmInstance.Load3D(addr2, uint32(len(data2)))

		assert.Equal(t, data1, read1)
		assert.Equal(t, data2, read2)

		// Phase 4: Verify computation results
		r0 := vmInstance.GetRegisters().GetUint64(vm.R0)
		r1 := vmInstance.GetRegisters().GetUint64(vm.R1)
		r2 := vmInstance.GetRegisters().GetUint64(vm.R2)

		// Expected: R0=10, R1=20, R2=30 (after ADD)
		assert.Equal(t, uint64(10), r0)
		assert.Equal(t, uint64(20), r1)
		assert.Equal(t, uint64(30), r2)

		stats := vmInstance.GetStats()
		assert.Greater(t, stats.InstructionsExecuted, uint64(0))
		assert.Greater(t, vmInstance.GetGasUsed(), uint64(0))
	})

	t.Run("Memory_query_and_verification", func(t *testing.T) {
		vmConfig := vm.VMConfig{
			MemorySize:      1024 * 1024,
			GasLimit:        50000,
			EnableProof:     false,
			EnableCompaction: false,
			MaxInstructions: 100,
		}
		vmInstance, err := vm.NewVM3D(vmConfig)
		require.NoError(t, err)
		defer vmInstance.Shutdown()

		// Allocate blocks of different sizes
		sizes := []uint32{256, 512, 1024}
		addresses := make([]types.Address3D, len(sizes))

		for i, size := range sizes {
			addr, err := vmInstance.AllocateMemory(size, types.RegionTypeHeap)
			require.NoError(t, err)
			addresses[i] = addr
		}

		// Query sizes using MSIZE3D
		for i, addr := range addresses {
			// Build MSIZE3D program
			program := make([]byte, 16)
			program[0] = 0x15 // MSIZE3D opcode
			binary.BigEndian.PutUint64(program[1:9], addr.X)
			binary.BigEndian.PutUint32(program[9:13], addr.Y)
			binary.BigEndian.PutUint16(program[13:15], addr.Z)
			program[15] = byte(vm.R0) // Result destination

			err := vmInstance.Execute(program)
			require.NoError(t, err)

			// Check result
			queriedSize := vmInstance.GetRegisters().GetUint64(vm.R0)
			assert.Equal(t, uint64(sizes[i]), queriedSize,
				fmt.Sprintf("Size query for block %d should return %d", i, sizes[i]))
		}
	})

	t.Run("Register_data_flow", func(t *testing.T) {
		vmConfig := vm.VMConfig{
			MemorySize:      1024 * 1024,
			GasLimit:        50000,
			EnableProof:     false,
			EnableCompaction: false,
			MaxInstructions: 100,
		}
		vmInstance, err := vm.NewVM3D(vmConfig)
		require.NoError(t, err)
		defer vmInstance.Shutdown()

		// Initialize registers with test data
		testData1 := []byte{0x01, 0x02, 0x03, 0x04}
		testData2 := []byte{0x05, 0x06, 0x07, 0x08}
		testData3 := []byte{0x09, 0x0A, 0x0B, 0x0C}

		vmInstance.GetRegisters().Set(vm.R0, testData1)
		vmInstance.GetRegisters().Set(vm.R1, testData2)
		vmInstance.GetRegisters().Set(vm.R2, testData3)

		// Build program: R3 = R0, then R4 = R1, then R5 = R3
		// This creates a chain: R0 -> R3 -> R5
		program := buildRegisterChainProgram()
		err = vmInstance.Execute(program)
		require.NoError(t, err)

		// Verify data flow
		r3Data := vmInstance.GetRegisters().Get(vm.R3)
		r4Data := vmInstance.GetRegisters().Get(vm.R4)
		r5Data := vmInstance.GetRegisters().Get(vm.R5)

		assert.Equal(t, testData1, r3Data, "R3 should equal R0")
		assert.Equal(t, testData2, r4Data, "R4 should equal R1")
		assert.Equal(t, testData1, r5Data, "R5 should equal R3 (which is R0)")
	})

	t.Run("Gas_consumption_tracking", func(t *testing.T) {
		gasLimit := uint64(500000)
		vmConfig := vm.VMConfig{
			MemorySize:      1024 * 1024,
			GasLimit:        gasLimit,
			EnableProof:     false,
			EnableCompaction: false,
			MaxInstructions: 100,
		}
		vmInstance, err := vm.NewVM3D(vmConfig)
		require.NoError(t, err)
		defer vmInstance.Shutdown()

		// Allocate memory (costs gas)
		vmInstance.AllocateMemory(256, types.RegionTypeHeap)

		// Execute program with operations
		program := buildSimpleArithmeticProgram()
		err = vmInstance.Execute(program)
		require.NoError(t, err)

		totalGasUsed := vmInstance.GetGasUsed()
		assert.Greater(t, totalGasUsed, uint64(0), "Should have used some gas")
	})

	t.Run("Program_halt_correctness", func(t *testing.T) {
		vmConfig := vm.VMConfig{
			MemorySize:      1024 * 1024,
			GasLimit:        10000,
			EnableProof:     false,
			EnableCompaction: false,
			MaxInstructions: 100,
		}
		vmInstance, err := vm.NewVM3D(vmConfig)
		require.NoError(t, err)
		defer vmInstance.Shutdown()

		// Build program: NOP, NOP, HALT, NOP (last NOP should not execute)
		program := []byte{
			0x00,       // OP_NOP
			0x00,       // OP_NOP
			0xFF,       // OP_HALT3D
			0x00,       // OP_NOP (should not execute)
			0x00,       // OP_NOP (should not execute)
		}

		err = vmInstance.Execute(program)
		require.NoError(t, err)

		// Should have executed exactly 3 instructions
		stats := vmInstance.GetStats()
		assert.Equal(t, uint64(3), stats.InstructionsExecuted,
			"HALT should stop execution, only 3 instructions should run")
	})
}

// Helper functions

func buildComplexComputationProgram() []byte {
	// Build a program that performs:
	// R0 = 100, R1 = 50
	// R2 = R0 + R1 = 150
	// R3 = R2 - R1 = 100
	// Then HALT

	// Initialize registers manually before execution
	// This program just does arithmetic
	program := []byte{
		0x31,       // ADD3D: R2 = R0 + R1
		byte(vm.R2), // dest
		byte(vm.R0), // src1
		byte(vm.R1), // src2
		0x32,       // SUB3D: R3 = R2 - R1
		byte(vm.R3),
		byte(vm.R2),
		byte(vm.R1),
		0xFF, // HALT3D
	}
	return program
}

func buildSequentialArithmeticProgram() []byte {
	// Register initialization should be done before Execute
	// This builds: R2 = R0 + R1
	program := []byte{
		0x31,       // ADD3D
		byte(vm.R2), // destination
		byte(vm.R0), // source 1
		byte(vm.R1), // source 2
		0xFF,       // HALT
	}
	return program
}

func buildRegisterChainProgram() []byte {
	// R3 = R0, R4 = R1, R5 = R3
	// MOV3D format: [opcode(1)][destReg(1)][srcReg(1)][padding(5)]
	
	// MOV R3 = R0
	mov1 := make([]byte, 8)
	mov1[0] = 0x30 // MOV3D opcode
	mov1[1] = byte(vm.R3)
	mov1[2] = byte(vm.R0)
	
	// MOV R4 = R1
	mov2 := make([]byte, 8)
	mov2[0] = 0x30 // MOV3D opcode
	mov2[1] = byte(vm.R4)
	mov2[2] = byte(vm.R1)
	
	// MOV R5 = R3
	mov3 := make([]byte, 8)
	mov3[0] = 0x30 // MOV3D opcode
	mov3[1] = byte(vm.R5)
	mov3[2] = byte(vm.R3)
	
	halt := []byte{0xFF}

	program := append(mov1, mov2...)
	program = append(program, mov3...)
	program = append(program, halt...)
	return program
}

func buildSimpleArithmeticProgram() []byte {
	// Simple: R2 = R0 + R1
	program := []byte{
		0x31,       // ADD3D
		byte(vm.R2), // destination
		byte(vm.R0), // source 1
		byte(vm.R1), // source 2
		0xFF,       // HALT
	}
	return program
}

// TestFullStackIntegratedWorkflow tests a realistic multi-step workflow
func TestFullStackIntegratedWorkflow(t *testing.T) {
	t.Run("Complete_application_simulation", func(t *testing.T) {
		// Simulate a realistic application:
		// 1. Initialize VM
		// 2. Allocate multiple memory regions
		// 3. Perform computations
		// 4. Store results
		// 5. Query and verify
		// 6. Cleanup

		vmConfig := vm.VMConfig{
			MemorySize:      1024 * 1024 * 10, // 10MB
			GasLimit:        5000000,          // 5M gas
			EnableProof:     true,
			EnableCompaction: true,
			MaxInstructions: 10000,
		}

		vmInstance, err := vm.NewVM3D(vmConfig)
		require.NoError(t, err)
		defer vmInstance.Shutdown()

		// Simulate an application workflow
		// Step 1: Allocate input buffer
		inputAddr, err := vmInstance.AllocateMemory(4096, types.RegionTypeHeap)
		require.NoError(t, err)

		inputData := []byte("This is input data for the application to process")
		err = vmInstance.Store3D(inputAddr, inputData)
		require.NoError(t, err)

		// Step 2: Allocate output buffer
		outputAddr, err := vmInstance.AllocateMemory(4096, types.RegionTypeHeap)
		require.NoError(t, err)

		// Step 3: Allocate computation state buffer
		_, err = vmInstance.AllocateMemory(2048, types.RegionTypeHeap)
		require.NoError(t, err)

		// Step 4: Perform computation
		vmInstance.GetRegisters().SetUint64(vm.R0, 1000)
		vmInstance.GetRegisters().SetUint64(vm.R1, 500)
		vmInstance.GetRegisters().SetUint64(vm.R2, 100)

		computeProgram := buildComplexComputationProgram()
		err = vmInstance.Execute(computeProgram)
		require.NoError(t, err)

		// Step 5: Store results
		result := vmInstance.GetRegisters().GetUint64(vm.R3)
		resultBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(resultBytes, result)
		err = vmInstance.Store3D(outputAddr, resultBytes)
		require.NoError(t, err)

		// Step 6: Read and verify
		storedResult, err := vmInstance.Load3D(outputAddr, 8)
		require.NoError(t, err)

		storedValue := binary.BigEndian.Uint64(storedResult)
		assert.Equal(t, result, storedValue)

		// Step 7: Check overall statistics
		stats := vmInstance.GetStats()
		assert.Greater(t, vmInstance.GetGasUsed(), uint64(0))
		assert.Greater(t, stats.InstructionsExecuted, uint64(0))

		t.Logf("Application simulation completed successfully")
		t.Logf("- Input: %s", string(inputData))
		t.Logf("- Result computed: %d", result)
		t.Logf("- Gas used: %d / %d", vmInstance.GetGasUsed(), vmConfig.GasLimit)
		t.Logf("- Allocations: %d", stats.Allocations)
	})
}

// TestRealWorldDataProcessing tests realistic data processing patterns
func TestRealWorldDataProcessing(t *testing.T) {
	t.Run("Batch_data_processing", func(t *testing.T) {
		vmConfig := vm.VMConfig{
			MemorySize:      1024 * 1024 * 50,
			GasLimit:        10000000,
			EnableProof:     true,
			EnableCompaction: true,
			MaxInstructions: 50000,
		}

		vmInstance, err := vm.NewVM3D(vmConfig)
		require.NoError(t, err)
		defer vmInstance.Shutdown()

		// Simulate batch processing of records
		recordSize := uint32(512)
		recordCount := 10

		records := make([]types.Address3D, recordCount)

		// Allocate and populate records
		for i := 0; i < recordCount; i++ {
			addr, err := vmInstance.AllocateMemory(recordSize, types.RegionTypeHeap)
			require.NoError(t, err)
			records[i] = addr

			// Store record data
			recordData := []byte(fmt.Sprintf("Record %d with some structured data", i))
			err = vmInstance.Store3D(addr, recordData)
			require.NoError(t, err)
		}

		// Process each record
		for i, addr := range records {
			// Retrieve record
			data, err := vmInstance.Load3D(addr, recordSize)
			require.NoError(t, err)

			// Verify record content
			assert.Contains(t, string(data), fmt.Sprintf("Record %d", i))
		}

		// Verify final state
		assert.Greater(t, vmInstance.GetGasUsed(), uint64(0))
		assert.Greater(t, vmInstance.GetGasUsed(), uint64(0))

		gasPerRecord := float64(vmInstance.GetGasUsed()) / float64(recordCount)
		t.Logf("Batch processing completed: %d records processed", recordCount)
		t.Logf("Gas efficiency: %.2f gas/record", gasPerRecord)
	})
}
