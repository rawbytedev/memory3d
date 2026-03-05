# Getting Started with Memory3D

A quick start guide to get you started with Memory3D.

## Table of Contents

1. [Installation](#installation)
2. [Your First Program](#your-first-program)
3. [Basic Operations](#basic-operations)
4. [Common Patterns](#common-patterns)
5. [Troubleshooting](#troubleshooting)

## Installation

### Prerequisites

- Go 1.25.2+

### Step 1: Get the Package

```bash
go get github.com/rawbytedev/memory3d
```

### Step 2: Import in Your Project

```go
import (
    "github.com/rawbytedev/memory3d/internal/types"
    "github.com/rawbytedev/memory3d/internal/vm"
)
```

### Step 3: Create a go.mod (if needed)

```bash
go mod init myproject
go mod tidy
```

## Your First Program

### Step 1: Set Up the VM

```go
package main

import (
    "fmt"
    "log"

    "github.com/rawbytedev/memory3d/internal/types"
    "github.com/rawbytedev/memory3d/internal/vm"
)

func main() {
    // Create VM configuration
    config := vm.VMConfig{
        MemorySize:       1024 * 1024,      // 1 MB
        GasLimit:         1000000,           // 1M gas
        EnableProof:      false,             // Don't need proofs yet
        EnableCompaction: true,              // Keep memory efficient
        MaxInstructions:  10000,             // 10K instructions max
        LogLevel:         vm.LogLevelWarn,   // Show warnings
    }

    // Create the VM
    vmInstance, err := vm.NewVM3D(config)
    if err != nil {
        log.Fatal("Failed to create VM:", err)
    }
    defer vmInstance.Shutdown()

    fmt.Println("VM created successfully!")
}
```

### Step 2: Allocate Memory

```go
func main() {
    // ... previous setup code ...

    // Allocate 256 bytes in heap
    addr, err := vmInstance.AllocateMemory(256, types.RegionTypeHeap)
    if err != nil {
        log.Fatal("Allocation failed:", err)
    }

    fmt.Printf("Allocated at: %v\n", addr)
    fmt.Printf("Address format: X=%d, Y=%d, Z=%d\n", 
        addr.X, addr.Y, addr.Z)
}
```

### Step 3: Write Data

```go
func main() {
    // ... previous code ...

    // Write data to memory
    data := []byte("Hello, Memory3D!")
    err = vmInstance.Store3D(addr, data)
    if err != nil {
        log.Fatal("Store failed:", err)
    }

    fmt.Println("Data written successfully!")
}
```

### Step 4: Read Data Back

```go
func main() {
    // ... previous code ...

    // Read data back
    loaded, err := vmInstance.Load3D(addr, 256)
    if err != nil {
        log.Fatal("Load failed:", err)
    }

    fmt.Printf("Loaded: %s\n", string(loaded))
}
```

### Complete First Program

```go
package main

import (
    "fmt"
    "log"

    "github.com/rawbytedev/memory3d/internal/types"
    "github.com/rawbytedev/memory3d/internal/vm"
)

func main() {
    // Create VM
    config := vm.VMConfig{
        MemorySize:       1024 * 1024,
        GasLimit:         1000000,
        EnableProof:      false,
        EnableCompaction: true,
        MaxInstructions:  10000,
        LogLevel:         vm.LogLevelWarn,
    }

    vmInstance, err := vm.NewVM3D(config)
    if err != nil {
        log.Fatal("VM creation failed:", err)
    }
    defer vmInstance.Shutdown()

    // Allocate memory
    addr, err := vmInstance.AllocateMemory(256, types.RegionTypeHeap)
    if err != nil {
        log.Fatal("Allocation failed:", err)
    }

    // Write data
    data := []byte("Hello, Memory3D!")
    err = vmInstance.Store3D(addr, data)
    if err != nil {
        log.Fatal("Store failed:", err)
    }

    // Read data
    loaded, err := vmInstance.Load3D(addr, 256)
    if err != nil {
        log.Fatal("Load failed:", err)
    }

    // Display results
    fmt.Printf("Original: %s\n", string(data))
    fmt.Printf("Loaded:   %s\n", string(loaded))
    fmt.Printf("Match:    %v\n", string(data) == string(loaded))

    // Show stats
    stats := vmInstance.GetStats()
    fmt.Printf("\nStatistics:\n")
    fmt.Printf("Gas used: %d\n", stats.TotalGasUsed)
    fmt.Printf("Allocations: %d\n", stats.Allocations)
    fmt.Printf("Memory usage: %d bytes\n", stats.MemoryUsage)
}
```

**Run it:**

```bash
go run main.go
```

**Output:**

```sh
Original: Hello, Memory3D!
Loaded:   Hello, Memory3D!
Match:    true

Statistics:
Gas used: 49
Allocations: 1
Memory usage: 16 bytes
```

## Basic Operations

### Arithmetic Operations (ADD3D, SUB3D)

Memory3D includes CPU-like arithmetic operations for register-based computation.

```go
func example_arithmetic(vmInstance *vm.VM3D) {
    // Create a simple program that performs arithmetic
    // R0 = 100, R1 = 50, then R2 = R0 + R1 = 150, finally R3 = R2 - R1 = 100
    
    config := vm.VMConfig{
        MemorySize:       1024 * 1024,
        GasLimit:         10000,
        EnableProof:      false,
        EnableCompaction: false,
        MaxInstructions:  100,
    }
    
    vm, _ := vm.NewVM3D(config)
    defer vm.Shutdown()
    
    // Set initial register values
    vm.GetRegisters().SetUint64(vm.R0, 100)
    vm.GetRegisters().SetUint64(vm.R1, 50)
    
    // Build program with ADD3D instruction
    // [opcode][destReg][srcReg1][srcReg2]
    program := []byte{
        0x31,       // ADD3D opcode
        byte(vm.R2), // R2 = destination
        byte(vm.R0), // R0 = first source
        byte(vm.R1), // R1 = second source
    }
    
    err := vm.Execute(program)
    if err != nil {
        log.Fatal("Execution failed:", err)
    }
    
    // Check result
    result := vm.GetRegisters().GetUint64(vm.R2)
    fmt.Printf("R2 (R0 + R1) = %d (expected 150)\n", result)
}
```

### Register Move (MOV3D)

Move data between registers for manipulation.

```go
func example_registerMove(vmInstance *vm.VM3D) {
    config := vm.VMConfig{
        MemorySize:       1024 * 1024,
        GasLimit:         10000,
        EnableProof:      false,
        EnableCompaction: false,
        MaxInstructions:  100,
    }
    
    vmInst, _ := vm.NewVM3D(config)
    defer vmInst.Shutdown()
    
    // Set up source register with data
    sourceData := []byte{0x42, 0x13, 0x37, 0xAB}
    vmInst.GetRegisters().Set(vm.R0, sourceData)
    
    // Build MOV3D program: R1 = R0
    // Format: [opcode(1)][destReg(1)][srcReg(1)][padding(5)]
    program := []byte{
        0x30, // MOV3D opcode
        byte(vm.R1), // Destination: R1
        byte(vm.R0), // Source: R0
        0, 0, 0, 0, 0, // Padding
    }
    
    err := vmInst.Execute(program)
    if err != nil {
        log.Fatal("Execution failed:", err)
    }
    
    // Verify data was moved
    destData := vmInst.GetRegisters().Get(vm.R1)
    fmt.Printf("Register moved: %v\n", bytes.Equal(sourceData, destData))
}
```

### Query Allocation Size (MSIZE3D)

Query the size of memory allocations.

```go
func example_querySize(vmInstance *vm.VM3D) {
    config := vm.VMConfig{
        MemorySize:       1024 * 1024,
        GasLimit:         10000,
        EnableProof:      false,
        EnableCompaction: false,
        MaxInstructions:  100,
    }
    
    vmInst, _ := vm.NewVM3D(config)
    defer vmInst.Shutdown()
    
    // Allocate memory
    addr, _ := vmInst.AllocateMemory(1024, types.RegionTypeHeap)
    
    // Build MSIZE3D program to query size
    // Format: [opcode][x(8)][y(4)][z(2)][destReg]
    program := make([]byte, 16)
    program[0] = 0x15 // MSIZE3D opcode
    binary.BigEndian.PutUint64(program[1:9], addr.X)
    binary.BigEndian.PutUint32(program[9:13], addr.Y)
    binary.BigEndian.PutUint16(program[13:15], addr.Z)
    program[15] = byte(vm.R0) // Result destination
    
    err := vmInst.Execute(program)
    if err != nil {
        log.Fatal("Execution failed:", err)
    }
    
    // Check result
    size := vmInst.GetRegisters().GetUint64(vm.R0)
    fmt.Printf("Allocation size: %d bytes\n", size)
}
```

### Multiple Allocations

```go
func example_multiAlloc(vmInstance *vm.VM3D) {
    addresses := make([]types.Address3D, 5)

    // Allocate 5 blocks
    for i := 0; i < 5; i++ {
        addr, err := vmInstance.AllocateMemory(128, types.RegionTypeHeap)
        if err != nil {
            log.Fatal("Allocation failed:", err)
        }
        addresses[i] = addr
        fmt.Printf("Allocated block %d at %v\n", i, addr)
    }

    // Write to each
    for i, addr := range addresses {
        data := []byte(fmt.Sprintf("Block %d data", i))
        vmInstance.Store3D(addr, data)
    }

    // Read from each
    for i, addr := range addresses {
        data, _ := vmInstance.Load3D(addr, 128)
        fmt.Printf("Block %d: %s\n", i, string(data))
    }
}
```

### Memory Copying

```go
func example_memcopy(vmInstance *vm.VM3D) {
    // Allocate source
    src, err := vmInstance.AllocateMemory(256, types.RegionTypeHeap)
    if err != nil {
        log.Fatal("Source allocation failed:", err)
    }

    // Write source data
    srcData := []byte("Data to copy")
    vmInstance.Store3D(src, srcData)

    // Allocate destination
    dst, err := vmInstance.AllocateMemory(256, types.RegionTypeHeap)
    if err != nil {
        log.Fatal("Dest allocation failed:", err)
    }

    // Copy
    err = vmInstance.Copy3D(src, dst, uint32(len(srcData)))
    if err != nil {
        log.Fatal("Copy failed:", err)
    }

    // Verify copy
    dstData, _ := vmInstance.Load3D(dst, 256)
    fmt.Printf("Copy successful: %s\n", string(dstData))
}
```

### Memory Freeing

```go
func example_memfree(vmInstance *vm.VM3D) {
    // Allocate
    addr, _ := vmInstance.AllocateMemory(256, types.RegionTypeHeap)
    vmInstance.Store3D(addr, []byte("Temporary data"))

    // Use the data
    data, _ := vmInstance.Load3D(addr, 256)
    fmt.Printf("Using: %s\n", string(data))

    // Free when done
    err := vmInstance.Free3D(addr)
    if err != nil {
        log.Fatal("Free failed:", err)
    }

    fmt.Println("Memory freed")

    // Accessing freed memory will fail
    _, err = vmInstance.Load3D(addr, 256)
    if err != nil {
        fmt.Printf("Expected error: %v\n", err)
    }
}
```

### Gas Monitoring

```go
func example_gasMonitoring(vmInstance *vm.VM3D) {
    fmt.Printf("Gas at start: %d\n", vmInstance.GetGasRemaining())

    // Each operation consumes gas
    vmInstance.AllocateMemory(256, types.RegionTypeHeap)
    fmt.Printf("After allocate: %d\n", vmInstance.GetGasRemaining())

    vmInstance.Store3D(types.Address3D{X: 0, Y: 0, Z: 0}, []byte("test"))
    fmt.Printf("After store: %d\n", vmInstance.GetGasRemaining())

    // Check total used
    fmt.Printf("Total gas used: %d\n", vmInstance.GetGasUsed())
}
```

## Common Patterns

### Pattern 1: Work with Fixed-Size Blocks

```go
func workWithBlocks(vmInstance *vm.VM3D, blockSize uint32, count int) {
    blockSize := uint32(1024) // 1KB blocks
    blocks := make([]types.Address3D, count)

    // Allocate all blocks upfront
    for i := 0; i < count; i++ {
        addr, _ := vmInstance.AllocateMemory(blockSize, types.RegionTypeHeap)
        blocks[i] = addr
    }

    // Process blocks
    for i, addr := range blocks {
        data := make([]byte, blockSize)
        // ... fill data ...
        vmInstance.Store3D(addr, data)
    }

    // Clean up
    for _, addr := range blocks {
        vmInstance.Free3D(addr)
    }
}
```

### Pattern 2: Track Address Objects

```go
type MemoryBlock struct {
    Addr Address3D
    Size uint32
    Data []byte
}

func createBlock(vmInstance *vm.VM3D, size uint32) (*MemoryBlock, error) {
    addr, err := vmInstance.AllocateMemory(size, types.RegionTypeHeap)
    if err != nil {
        return nil, err
    }

    return &MemoryBlock{
        Addr: addr,
        Size: size,
    }, nil
}

func (b *MemoryBlock) Write(vm *vm.VM3D, data []byte) error {
    return vm.Store3D(b.Addr, data)
}

func (b *MemoryBlock) Read(vm *vm.VM3D) ([]byte, error) {
    return vm.Load3D(b.Addr, b.Size)
}

func (b *MemoryBlock) Free(vm *vm.VM3D) error {
    return vm.Free3D(b.Addr)
}

// Usage
block, _ := createBlock(vmInstance, 256)
defer block.Free(vmInstance)
block.Write(vmInstance, []byte("data"))
data, _ := block.Read(vmInstance)
```

### Pattern 3: Memory Allocation Pool

```go
type MemoryPool struct {
    vm        *vm.VM3D
    blockSize uint32
    available []types.Address3D
}

func NewMemoryPool(vm *vm.VM3D, blockSize uint32, count int) *MemoryPool {
    pool := &MemoryPool{
        vm:        vm,
        blockSize: blockSize,
        available: make([]types.Address3D, 0, count),
    }

    // Pre-allocate blocks
    for i := 0; i < count; i++ {
        addr, _ := vm.AllocateMemory(blockSize, types.RegionTypeHeap)
        pool.available = append(pool.available, addr)
    }

    return pool
}

func (p *MemoryPool) Get() (types.Address3D, error) {
    if len(p.available) == 0 {
        // Allocate new block if pool exhausted
        return p.vm.AllocateMemory(p.blockSize, types.RegionTypeHeap)
    }

    addr := p.available[len(p.available)-1]
    p.available = p.available[:len(p.available)-1]
    return addr, nil
}

func (p *MemoryPool) Return(addr types.Address3D) {
    p.available = append(p.available, addr)
}

// Usage
pool := NewMemoryPool(vmInstance, 256, 10)
block, _ := pool.Get()
pool.Return(block)
```

### Pattern 4: Error-Safe Memory Management

```go
func safeMemoryOperation(vmInstance *vm.VM3D) error {
    // Allocate
    addr, err := vmInstance.AllocateMemory(256, types.RegionTypeHeap)
    if err != nil {
        return fmt.Errorf("allocation failed: %w", err)
    }
    // defer free in case of error
    defer vmInstance.Free3D(addr)

    // Store
    if err := vmInstance.Store3D(addr, []byte("data")); err != nil {
        return fmt.Errorf("store failed: %w", err)
    }

    // Load
    data, err := vmInstance.Load3D(addr, 256)
    if err != nil {
        return fmt.Errorf("load failed: %w", err)
    }

    fmt.Printf("Data: %s\n", string(data))
    return nil
}
```

## Troubleshooting

### Issue: "gas exhausted"

**Cause:** Operations consumed all available gas.

**Solution:**

```go
// Increase gas limit
config := vm.VMConfig{
    GasLimit: 10000000,  // Increase from default
    // ... other settings ...
}
```

### Issue: "address not allocated"

**Cause:** Trying to access unallocated memory.

**Solution:**

```go
// Ensure address is allocated first
addr, _ := vmInstance.AllocateMemory(256, types.RegionTypeHeap)
vmInstance.Store3D(addr, data)  // Now safe
```

### Issue: "merkle tree not enabled"

**Cause:** Calling proof functions without enabling Merkle tree.

**Solution:**

```go
config := vm.VMConfig{
    EnableProof: true,  // Add this
    // ... other settings ...
}
```

### Issue: Slow Performance

**Cause:** Suboptimal allocation patterns or high fragmentation.

**Solution:**

```go
// Use block pooling
// Batch allocations together
// Enable compaction
config.EnableCompaction = true

// Check fragmentation
report := vmInstance.GetCompactionReport()
for _, r := range report {
    if r.Fragmentation > 0.5 {
        // Trigger compaction
    }
}
```

### Issue: Memory Not Freed

**Cause:** Forgot to call Free3D or defer cleanup.

**Solution:**

```go
// Always use defer or cleanup pattern
addr, _ := vmInstance.AllocateMemory(256, types.RegionTypeHeap)
defer vmInstance.Free3D(addr)  // Add this
```

## Next Steps

1. **Read the API Reference**: See [API.md](./API.md) for complete API documentation
2. **Understand Architecture**: Read [ARCHITECTURE.md](./ARCHITECTURE.md) for technical deep-dive
3. **Explore Examples**: Check examples/ directory for more complex patterns
4. **Run Tests**: `go test ./...` to verify installation
5. **Run Benchmarks**: `go test -bench=. ./test/benchmarks/...` to see performance

## Quick Reference

```go
// Create VM
vm, _ := vm.NewVM3D(config)
defer vm.Shutdown()

// Allocate
addr, _ := vm.AllocateMemory(256, types.RegionTypeHeap)

// Store
_ = vm.Store3D(addr, []byte("data"))

// Load
data, _ := vm.Load3D(addr, 256)

// Free
_ = vm.Free3D(addr)

// Stats
stats := vm.GetStats()
gas := vm.GetGasRemaining()
```

---

**Happy coding!** For more information, see the [main README](./README.md).
