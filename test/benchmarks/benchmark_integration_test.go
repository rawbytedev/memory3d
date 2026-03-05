package benchmarks

import (
	"testing"

	"github.com/rawbytedev/memory3d/internal/types"
	"github.com/rawbytedev/memory3d/internal/vm"
	"github.com/rawbytedev/memory3d/test/helper"
)

// BenchmarkIntegrationAllocateAndStore benchmarks allocate + write
func BenchmarkIntegrationAllocateAndStore(b *testing.B) {
	config := vm.VMConfig{
		MemorySize:      1024 * 1024 * 10,
		GasLimit:        10000000,
		EnableProof:     false,
		MaxInstructions: 100000,
		LogLevel:        vm.LogLevelSilent,
	}

	vmInstance, _ := vm.NewVM3D(config)
	defer vmInstance.Shutdown()

	data := helper.GenerateData(256)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		addr, _ := vmInstance.AllocateMemory(256, types.RegionTypeHeap)
		vmInstance.Store3D(addr, data)
	}
}

// BenchmarkIntegrationAllocateStoreLoad benchmarks allocate + write + read
func BenchmarkIntegrationAllocateStoreLoad(b *testing.B) {
	config := vm.VMConfig{
		MemorySize:      1024 * 1024 * 10,
		GasLimit:        10000000,
		EnableProof:     false,
		MaxInstructions: 100000,
		LogLevel:        vm.LogLevelSilent,
	}

	vmInstance, _ := vm.NewVM3D(config)
	defer vmInstance.Shutdown()

	data := make([]byte, 256)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		addr, _ := vmInstance.AllocateMemory(256, types.RegionTypeHeap)
		vmInstance.Store3D(addr, data)
		vmInstance.Load3D(addr, 256)
	}
}

// BenchmarkIntegrationWithMerkle benchmarks full workflow with merkle enabled
func BenchmarkIntegrationWithMerkle(b *testing.B) {
	config := vm.VMConfig{
		MemorySize:      1024 * 1024 * 10,
		GasLimit:        10000000,
		EnableProof:     true,
		MaxInstructions: 100000,
		LogLevel:        vm.LogLevelSilent,
	}

	vmInstance, _ := vm.NewVM3D(config)
	defer vmInstance.Shutdown()

	data := make([]byte, 256)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		addr, _ := vmInstance.AllocateMemory(256, types.RegionTypeHeap)
		vmInstance.Store3D(addr, data)
		vmInstance.GetMerkleRoot()
	}
}

// BenchmarkIntegrationMerkleProof benchmarks merkle proof generation + verification
func BenchmarkIntegrationMerkleProof(b *testing.B) {
	config := vm.VMConfig{
		MemorySize:      1024 * 1024 * 10,
		GasLimit:        10000000,
		EnableProof:     true,
		MaxInstructions: 100000,
		LogLevel:        vm.LogLevelSilent,
	}

	vmInstance, _ := vm.NewVM3D(config)
	defer vmInstance.Shutdown()

	addr, _ := vmInstance.AllocateMemory(256, types.RegionTypeHeap)
	data := make([]byte, 256)
	vmInstance.Store3D(addr, data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		proof, _ := vmInstance.GetMerkleProof(addr)
		vmInstance.VerifyMemoryState(addr, data, proof)
	}
}

// BenchmarkIntegrationWithGas benchmarks workflow with gas consumption
func BenchmarkIntegrationWithGas(b *testing.B) {
	config := vm.VMConfig{
		MemorySize:      1024 * 1024 * 10,
		GasLimit:        100000000,
		EnableProof:     false,
		MaxInstructions: 1000000,
		LogLevel:        vm.LogLevelSilent,
	}

	vmInstance, _ := vm.NewVM3D(config)
	defer vmInstance.Shutdown()

	data := make([]byte, 256)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vmInstance.GetGasRemaining()
		addr, _ := vmInstance.AllocateMemory(256, types.RegionTypeHeap)
		vmInstance.Store3D(addr, data)
		vmInstance.Load3D(addr, 256)
		_ = vmInstance.GetGasUsed()
	}
}

// BenchmarkIntegrationMultipleAllocations benchmarks multiple sequential allocations
func BenchmarkIntegrationMultipleAllocations(b *testing.B) {
	config := vm.VMConfig{
		MemorySize:      1024 * 1024 * 10,
		GasLimit:        10000000,
		EnableProof:     false,
		MaxInstructions: 100000,
		LogLevel:        vm.LogLevelSilent,
	}

	vmInstance, _ := vm.NewVM3D(config)
	defer vmInstance.Shutdown()

	data := make([]byte, 256)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			addr, _ := vmInstance.AllocateMemory(256, types.RegionTypeHeap)
			vmInstance.Store3D(addr, data)
		}
	}
}

// BenchmarkIntegrationWithCompaction benchmarks workflow with compaction
func BenchmarkIntegrationWithCompaction(b *testing.B) {
	config := vm.VMConfig{
		MemorySize:      1024 * 1024 * 10,
		GasLimit:        10000000,
		EnableProof:     false,
		MaxInstructions: 100000,
		LogLevel:        vm.LogLevelSilent,
	}

	vmInstance, _ := vm.NewVM3D(config)
	defer vmInstance.Shutdown()

	// Pre-allocate
	for i := 0; i < 50; i++ {
		vmInstance.AllocateMemory(10240, types.RegionTypeHeap)
	}

	inst := &vm.Instruction3D{
		Opcode:   vm.OP_MCOMPACT3D,
		Operands: []vm.Operand{},
		Size:     1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vmInstance.AllocateMemory(256, types.RegionTypeHeap)
		vmInstance.ExecuteCompactInstruction(inst)
	}
}

// BenchmarkIntegrationFullWorkflow benchmarks complete VM workflow
func BenchmarkIntegrationFullWorkflow(b *testing.B) {
	config := vm.VMConfig{
		MemorySize:      1024 * 1024 * 10,
		GasLimit:        100000000,
		EnableProof:     true,
		MaxInstructions: 1000000,
		LogLevel:        vm.LogLevelSilent,
	}

	vmInstance, _ := vm.NewVM3D(config)
	defer vmInstance.Shutdown()

	data := make([]byte, 256)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Allocate multiple addresses
		addrs := make([]types.Address3D, 5)
		for j := 0; j < 5; j++ {
			addr, _ := vmInstance.AllocateMemory(256, types.RegionTypeHeap)
			addrs[j] = addr
			vmInstance.Store3D(addr, data)
		}

		// Read operations
		for _, addr := range addrs {
			vmInstance.Load3D(addr, 256)
		}

		// Merkle proof generation
		if len(addrs) > 0 {
			proof, _ := vmInstance.GetMerkleProof(addrs[0])
			vmInstance.VerifyMemoryState(addrs[0], data, proof)
		}

		// Get stats
		_ = vmInstance.GetStats()
		_ = vmInstance.GetGasRemaining()
	}
}
