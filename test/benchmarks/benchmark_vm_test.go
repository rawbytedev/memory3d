package benchmarks

import (
	"testing"

	"github.com/rawbytedev/memory3d/internal/types"
	"github.com/rawbytedev/memory3d/internal/vm"
)

// BenchmarkVMCreate benchmarks VM creation
func BenchmarkVMCreate(b *testing.B) {
	config := vm.VMConfig{
		MemorySize:      1024 * 1024,
		GasLimit:        1000000,
		EnableProof:     false,
		MaxInstructions: 10000,
		LogLevel:        vm.LogLevelSilent,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vmInstance, _ := vm.NewVM3D(config)
		vmInstance.Shutdown()
	}
}

// BenchmarkVMAllocate benchmarks VM memory allocation
func BenchmarkVMAllocate(b *testing.B) {
	config := vm.VMConfig{
		MemorySize:      1024 * 1024 * 10,
		GasLimit:        10000000,
		EnableProof:     false,
		MaxInstructions: 100000,
		LogLevel:        vm.LogLevelSilent,
	}

	vmInstance, _ := vm.NewVM3D(config)
	defer vmInstance.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vmInstance.AllocateMemory(1024, types.RegionTypeHeap)
	}
}

// BenchmarkVMStore3D benchmarks memory write operations
func BenchmarkVMStore3D(b *testing.B) {
	config := vm.VMConfig{
		MemorySize:      1024 * 1024 * 10,
		GasLimit:        10000000,
		EnableProof:     false,
		MaxInstructions: 100000,
		LogLevel:        vm.LogLevelSilent,
	}

	vmInstance, _ := vm.NewVM3D(config)
	defer vmInstance.Shutdown()

	addr, _ := vmInstance.AllocateMemory(4096, types.RegionTypeHeap)
	data := make([]byte, 256)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vmInstance.Store3D(addr, data)
	}
}

// BenchmarkVMLoad3D benchmarks memory read operations
func BenchmarkVMLoad3D(b *testing.B) {
	config := vm.VMConfig{
		MemorySize:      1024 * 1024 * 10,
		GasLimit:        10000000,
		EnableProof:     false,
		MaxInstructions: 100000,
		LogLevel:        vm.LogLevelSilent,
	}

	vmInstance, _ := vm.NewVM3D(config)
	defer vmInstance.Shutdown()

	addr, _ := vmInstance.AllocateMemory(4096, types.RegionTypeHeap)
	data := make([]byte, 256)
	vmInstance.Store3D(addr, data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vmInstance.Load3D(addr, uint32(len(data)))
	}
}

// BenchmarkVMGetStats benchmarks stats retrieval
func BenchmarkVMGetStats(b *testing.B) {
	config := vm.VMConfig{
		MemorySize:      1024 * 1024,
		GasLimit:        1000000,
		EnableProof:     false,
		MaxInstructions: 10000,
		LogLevel:        vm.LogLevelSilent,
	}

	vmInstance, _ := vm.NewVM3D(config)
	defer vmInstance.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vmInstance.GetStats()
	}
}

// BenchmarkVMGetGasInfo benchmarks gas info retrieval
func BenchmarkVMGetGasInfo(b *testing.B) {
	config := vm.VMConfig{
		MemorySize:      1024 * 1024,
		GasLimit:        1000000,
		EnableProof:     false,
		MaxInstructions: 10000,
		LogLevel:        vm.LogLevelSilent,
	}

	vmInstance, _ := vm.NewVM3D(config)
	defer vmInstance.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vmInstance.GetGasRemaining()
		vmInstance.GetGasUsed()
	}
}

// BenchmarkVMCompaction benchmarks compaction execution
func BenchmarkVMCompaction(b *testing.B) {
	config := vm.VMConfig{
		MemorySize:      1024 * 1024 * 10,
		GasLimit:        10000000,
		EnableProof:     false,
		MaxInstructions: 100000,
		LogLevel:        vm.LogLevelSilent,
	}

	vmInstance, _ := vm.NewVM3D(config)
	defer vmInstance.Shutdown()

	// Pre-allocate some memory
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
		vmInstance.ExecuteCompactInstruction(inst)
	}
}
