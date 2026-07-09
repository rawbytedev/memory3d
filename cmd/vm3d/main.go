package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/rawbytedev/memory3d/internal/types"
	"github.com/rawbytedev/memory3d/internal/vm"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	f, err := os.Create("mem.prof")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	runtime.MemProfileRate = 1

	vmConfig := vm.VMConfig{ // needs to be turned into pointer
		MemorySize:       1024 * 1024, // 1MB
		GasLimit:         500000,      // 500K gas
		EnableProof:      true,        // Enable proofs for security
		EnableCompaction: true,        // Enable memory compaction
		MaxInstructions:  5000,        // 5K instructions
	}
	vmInstance, err := vm.NewVM3D(vmConfig)
	if err != nil {
		return
	}
	defer vmInstance.Shutdown()

	gasAtStart := vmInstance.GetGasRemaining()

	// Scenario: Calculate and store computation results
	// 1. Allocate result storage (1KB)
	resultAddr, err := vmInstance.AllocateMemory(1024, types.RegionTypeHeap)
	if err != nil {
		return
	}

	// 2. Store initial data in memory
	initialData := []byte("Configuration data for processing")
	err = vmInstance.Store3D(resultAddr, initialData)
	if err != nil {
		return
	}

	// 3. Verify stored data
	retrievedData, err := vmInstance.Load3D(resultAddr, uint32(len(initialData)))
	if err != nil {
		return
	}
	if !bytes.Equal(initialData, retrievedData) {
		fmt.Printf("Stored and retrieved data should match\n")
	}

	// 4. Build computation program (arithmetic operations)
	program := buildComplexComputationProgram()

	// Set initial values for computation
	vmInstance.GetRegisters().SetUint64(vm.R0, 100)
	vmInstance.GetRegisters().SetUint64(vm.R1, 50)

	err = vmInstance.Execute(program)
	if err != nil {
		return
	}

	// 5. Verify arithmetic results
	r2Result := vmInstance.GetRegisters().GetUint64(vm.R2)
	if uint64(150) != r2Result {
		fmt.Print("R2 should be 100 + 50 = 150")
	}

	r3Result := vmInstance.GetRegisters().GetUint64(vm.R3)
	if uint64(100) != r3Result {
		fmt.Print("R3 should be 150 - 50 = 100")
	}
	// 6. Check statistics
	stats := vmInstance.GetStats()
	if stats.InstructionsExecuted < uint64(0) {
		fmt.Print("Should execute instructions")
	}
	if vmInstance.GetGasUsed() < uint64(0) {
		fmt.Print("Should consume some gas")
	}

	gasAtEnd := vmInstance.GetGasRemaining()
	if gasAtEnd >= gasAtStart {
		print("Gas should be consumed")
	}
	fmt.Printf("Instructions executed: %d", stats.InstructionsExecuted)
	fmt.Printf("Gas used: %d", vmInstance.GetGasUsed())
	fmt.Printf("Memory usage: %d bytes", stats.MemoryUsage)

	pprof.WriteHeapProfile(f)
	time.Sleep(5 * time.Minute)
}

func buildComplexComputationProgram() []byte {
	// Build a program that performs:
	// R0 = 100, R1 = 50
	// R2 = R0 + R1 = 150
	// R3 = R2 - R1 = 100
	// Then HALT

	// Initialize registers manually before execution
	// This program just does arithmetic
	program := []byte{
		0x31,        // ADD3D: R2 = R0 + R1
		byte(vm.R2), // dest
		byte(vm.R0), // src1
		byte(vm.R1), // src2
		0x32,        // SUB3D: R3 = R2 - R1
		byte(vm.R3),
		byte(vm.R2),
		byte(vm.R1),
		0xFF, // HALT3D
	}
	return program
}
