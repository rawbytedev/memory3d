package main

import (
	"fmt"
	"log"

	"github.com/rawbytedev/memory3d/internal/allocator"
	"github.com/rawbytedev/memory3d/internal/merkle"
	"github.com/rawbytedev/memory3d/internal/types"
	"github.com/rawbytedev/memory3d/internal/vm"
)

func main() {
	
	fmt.Println("=== Memory3D Validation Test ===\n")

	// Test 1: Type System
	fmt.Println("✓ Test 1: RegionType Enum")
	testRegionTypes()

	// Test 2: Gas Accountant
	fmt.Println("\n✓ Test 2: GasAccountant")
	testGasAccountant()

	// Test 3: Allocator
	fmt.Println("\n✓ Test 3: Allocator3D")
	testAllocator()

	// Test 4: Memory Manager
	fmt.Println("\n✓ Test 4: MemoryManager")
	testMemoryManager()

	// Test 5: Merkle Tree
	fmt.Println("\n✓ Test 5: Merkle Tree")
	testMerkleTree()

	// Test 6: SpatialCache
	fmt.Println("\n✓ Test 6: SpatialCache")
	testSpatialCache()

	// Test 7: Instruction Decoders
	fmt.Println("\n✓ Test 7: Instruction Decoders")
	testInstructionDecoders()

	// Test 8: VM Creation
	fmt.Println("\n✓ Test 8: VM3D Creation")
	testVMCreation()

	fmt.Println("\n=== All Validation Tests Passed! ===")
}

func testRegionTypes() {
	regionTypes := []types.RegionType{
		types.RegionTypeInvalid,
		types.RegionTypeVMControl,
		types.RegionTypeCode,
		types.RegionTypeState,
		types.RegionTypeHeap,
		types.RegionTypeStack,
		types.RegionTypeShared,
		types.RegionTypeTransient,
	}

	for _, rt := range regionTypes {
		region := types.NewRegion(uint64(rt), rt)
		if region == nil {
			log.Fatal("Failed to create region")
		}
		perms := types.GetDefaultPermissions(rt)
		fmt.Printf("  - RegionType %d: Readable=%v, Writable=%v\n", rt, perms.Readable, perms.Writable)
	}
}

func testGasAccountant() {
	accountant := vm.NewAccountant(1000000)
	if accountant == nil {
		log.Fatal("Failed to create gas accountant")
	}

	// Test gas consumption
	err := accountant.Consume(100)
	if err != nil {
		log.Fatalf("Failed to consume gas: %v", err)
	}

	remaining := accountant.Remaining()
	if remaining <= 0 {
		log.Fatalf("Invalid remaining gas: %d", remaining)
	}

	fmt.Printf("  - Gas Limit: 1000000\n")
	fmt.Printf("  - Gas Consumed: 100\n")
	fmt.Printf("  - Gas Remaining: %d\n", remaining)
}

func testAllocator() {
	config := allocator.Config{
		PlaneSize:        types.PlaneSize,
		MaxYPlanes:       1024,
		MaxYPromotions:   8,
		WorkerCount:      1,
		EnableConcurrent: true,
	}

	alloc := allocator.NewAllocator3D(config)
	if alloc == nil {
		log.Fatal("Failed to create allocator")
	}

	// Test allocation
	allocation, err := alloc.Allocate(1024, types.RegionTypeHeap, 0)
	if err != nil {
		log.Fatalf("Failed to allocate: %v", err)
	}

	if !allocation.Address.IsValid() {
		log.Fatal("Invalid allocation address")
	}

	fmt.Printf("  - Allocated 1024 bytes at address: %v\n", allocation.Address)

	// Test free
	err = alloc.Free(allocation)
	if err != nil {
		log.Fatalf("Failed to free: %v", err)
	}

	fmt.Printf("  - Successfully freed allocation\n")
}

func testMemoryManager() {
	mm := vm.NewMemoryManager()
	if mm == nil {
		log.Fatal("Failed to create memory manager")
	}

	fmt.Printf("  - MemoryManager created successfully\n")
}

func testMerkleTree() {
	tree := merkle.NewTree3D()
	if tree == nil {
		log.Fatal("Failed to create merkle tree")
	}

	addr := types.Address3D{X: 0, Y: 0, Z: 0}
	data := []byte("test data")

	err := tree.Update(addr, data)
	if err != nil {
		log.Fatalf("Failed to update tree: %v", err)
	}

	root := tree.GetRoot()
	fmt.Printf("  - Tree updated with data at %v\n", addr)
	fmt.Printf("  - Root hash: % x\n", root[:8])
	fmt.Printf("  - Tree statistics: %v\n", tree.GetStatistics())
}

func testSpatialCache() {
	cache := vm.NewSpatialCache()
	if cache == nil {
		log.Fatal("Failed to create spatial cache")
	}

	addr := types.Address3D{X: 0, Y: 0, Z: 0}
	data := []byte("cached data")

	cache.Put(addr, data)
	retrieved, found := cache.Get(addr)
	if !found || string(retrieved) != "cached data" {
		log.Fatal("Cache retrieval failed")
	}

	fmt.Printf("  - Successfully cached and retrieved data at %v\n", addr)

	cache.Invalidate(addr)
	_, found = cache.Get(addr)
	if found {
		log.Fatal("Cache invalidation failed")
	}

	fmt.Printf("  - Successfully invalidated cache entry\n")
}

func testInstructionDecoders() {
	// Test MLOAD3D decoder (20 bytes)
	mloadData := make([]byte, 20)
	mloadData[0] = byte(vm.OP_MLOAD3D)
	inst, err := vm.DecodeInstruction(mloadData)
	if err != nil {
		log.Fatalf("Failed to decode MLOAD3D: %v", err)
	}
	fmt.Printf("  - MLOAD3D decoded: size=%d\n", inst.Size)

	// Test MALLOC3D decoder (14 bytes)
	mallocData := make([]byte, 14)
	mallocData[0] = byte(vm.OP_MALLOC3D)
	inst, err = vm.DecodeInstruction(mallocData)
	if err != nil {
		log.Fatalf("Failed to decode MALLOC3D: %v", err)
	}
	fmt.Printf("  - MALLOC3D decoded: size=%d\n", inst.Size)

	// Test MFREE3D decoder (15 bytes)
	freeData := make([]byte, 15)
	freeData[0] = byte(vm.OP_MFREE3D)
	inst, err = vm.DecodeInstruction(freeData)
	if err != nil {
		log.Fatalf("Failed to decode MFREE3D: %v", err)
	}
	fmt.Printf("  - MFREE3D decoded: size=%d\n", inst.Size)

	// Test MCOPY3D decoder (20 bytes)
	copyData := make([]byte, 20)
	copyData[0] = byte(vm.OP_MCOPY3D)
	inst, err = vm.DecodeInstruction(copyData)
	if err != nil {
		log.Fatalf("Failed to decode MCOPY3D: %v", err)
	}
	fmt.Printf("  - MCOPY3D decoded: size=%d\n", inst.Size)
}

func testVMCreation() {
	config := vm.VMConfig{
		MemorySize:       1024 * 1024 * 64,
		GasLimit:         1000000,
		EnableProof:      true,
		EnableCompaction: false,
		MaxInstructions:  1000,
		LogLevel:         vm.LogLevelInfo,
	}

	v, err := vm.NewVM3D(config)
	if err != nil {
		log.Fatalf("Failed to create VM: %v", err)
	}

	if v == nil {
		log.Fatal("VM is nil")
	}

	fmt.Printf("  - VM3D created successfully\n")
	fmt.Printf("  - Memory size: %d bytes\n", config.MemorySize)
	fmt.Printf("  - Gas limit: %d\n", config.GasLimit)
	fmt.Printf("  - Proof enabled: %v\n", config.EnableProof)
}
