# Memory3D - A 3D Virtual Memory Machine

[![Go Version](https://img.shields.io/badge/Go-1.25.2-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-Apache 2.0-green.svg)](#license)

**Memory3D** is a sophisticated 3D virtual machine designed for efficient memory management, spatial operations, and cryptographic memory proofing. It provides a unique approach to memory addressing using three dimensions (Regions, Planes, and Offsets) combined with advanced features like Merkle tree verification, gas metering, and intelligent memory compaction.

## Table of Contents

- [Overview](#overview)
- [Key Features](#key-features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Usage](#usage)
- [API Reference](#api-reference)
- [Examples](#examples)
- [Performance](#performance)
- [Contributing](#contributing)
- [License](#license)

## Overview

Memory3D is an advanced virtual machine that extends traditional linear memory models into three dimensions:

```arch
Region (X) → Plane (Y) → Offset (Z)
   |            |           |
   |            |           └─ 16-bit offset within plane (64KB)
   |            └──────────── 32-bit plane index (65K planes)
   └────────────────────────── 64-bit region identifier (256 regions)
```

This 3D addressing scheme enables:

- **Spatial locality optimization** through region and plane organization
- **Efficient fragmentation management** via plane-based allocation
- **Memory proofing** through Merkle tree integration
- **Scaled memory systems** without traditional heap overhead

## Key Features

### 3D Memory Management

- **Multi-dimensional addressing**: Region → Plane → Offset structure
- **Configurable memory sizes**: From kilobytes to terabytes
- **Intelligent allocation**: Fast-path allocation with region LRU cache

### Cryptographic Memory Proofs

- **Merkle tree integration**: Generate and verify memory state proofs
- **Zero-knowledge proofs**: Prove memory contents without disclosure
- **Privacy-preserving**: Verify allocations without exposing data

### Gas Metering System

- **Operation metering**: Track computational cost of operations
- **Refund mechanism**: Recover gas from memory consolidation
- **Configurable limits**: Set per-VM gas budgets

### Memory Compaction & Defragmentation

- **Intelligent consolidation**: Analyze fragmentation and plan moves
- **Y-Promotion**: Dynamically promote allocations to higher planes
- **Compaction reports**: Detailed fragmentation analysis

### Performance Monitoring

- **Real-time statistics**: Track allocations, gas usage, Y-promotions
- **Latency profiling**: Instruction-level performance metrics
- **Cache efficiency**: Monitor spatial locality and hit rates

### Concurrent Operations

- **Thread-safe design**: Built-in synchronization for multi-threaded access
- **Private vs. shared**: Optional worker-based concurrency model
- **Deadlock-free**: Careful lock ordering prevents contention

## Architecture

The Memory3D system consists of several key components:

```s
┌─────────────────────────────────────────┐
│           VM3D (Virtual Machine)        │
├─────────────────────────────────────────┤
│  Allocator3D    GasAccountant    Cache  │
│  (3D Malloc)    (Metering)      (Spatial)
├─────────────────────────────────────────┤
│  Types System (Regions, Planes, Allocation)
│  Merkle Tree (Proof System)
└─────────────────────────────────────────┘
```

**Core Components:**

| Component | Purpose |
| ----------- | --------- |
| **VM3D** | Main virtual machine orchestrator |
| **Allocator3D** | 3D memory allocation and deallocation |
| **MemoryManager** | Virtual memory addressing and access |
| **GasAccountant** | Operation metering and cost tracking |
| **Tree3D (Merkle)** | Cryptographic verification system |
| **SpatialCache** | Locality-based caching layer |

For detailed architecture information, see [ARCHITECTURE.md](./ARCHITECTURE.md).

## Quick Start

### Installation

```bash
go get github.com/rawbytedev/memory3d
```

### Basic Usage

```go
package main

import (
    "fmt"
    "github.com/rawbytedev/memory3d/internal/types"
    "github.com/rawbytedev/memory3d/internal/vm"
)

func main() {
    // Create VM with configuration
    config := vm.VMConfig{
        MemorySize:      1024 * 1024 * 100, // 100MB
        GasLimit:        10000000,
        EnableProof:     true,
        MaxInstructions: 100000,
        LogLevel:        vm.LogLevelInfo,
    }
    
    vmInstance, err := vm.NewVM3D(config)
    if err != nil {
        panic(err)
    }
    defer vmInstance.Shutdown()
    
    // Allocate memory
    addr, err := vmInstance.AllocateMemory(256, types.RegionTypeHeap)
    if err != nil {
        panic(err)
    }
    
    // Store data
    data := []byte("Hello, Memory3D!")
    err = vmInstance.Store3D(addr, data)
    if err != nil {
        panic(err)
    }
    
    // Load data back
    loaded, err := vmInstance.Load3D(addr, 256)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Stored: %s\nLoaded: %s\n", string(data), string(loaded))
    
    // Get statistics
    stats := vmInstance.GetStats()
    fmt.Printf("Gas used: %d\n", stats.TotalGasUsed)
}
```

## Usage

### Creating and Configuring a VM

```go
config := vm.VMConfig{
    MemorySize:       1024 * 1024 * 1024, // 1GB
    GasLimit:         100000000,           // 100M gas
    EnableProof:      true,                // Enable Merkle proofs
    EnableCompaction: true,                // Enable auto-compaction
    MaxInstructions:  1000000,             // Max 1M instructions
    LogLevel:         vm.LogLevelInfo,     // Info level logging
}

vmInstance, err := vm.NewVM3D(config)
if err != nil {
    log.Fatal(err)
}
```

### Memory Operations

```go
// Allocate memory
addr, err := vmInstance.AllocateMemory(1024, types.RegionTypeHeap)

// Store data
data := []byte("your data here")
err = vmInstance.Store3D(addr, data)

// Load data
loaded, err := vmInstance.Load3D(addr, 1024)

// Copy memory
destAddr, _ := vmInstance.AllocateMemory(1024, types.RegionTypeHeap)
err = vmInstance.Copy3D(addr, destAddr, 1024)

// Free memory
err = vmInstance.Free3D(addr)
```

### Memory Proofing

```go
// Generate Merkle proof
proof, err := vmInstance.GetMerkleProof(addr)

// Verify memory state
verified, err := vmInstance.VerifyMemoryState(addr, data, proof)

// Get Merkle root
root := vmInstance.GetMerkleRoot()
```

### Gas Metering

```go
// Check remaining gas
remaining := vmInstance.GetGasRemaining()

// Check used gas
used := vmInstance.GetGasUsed()

// Query gas costs
cost := vm.CalculateGasCostForOperation(opcode, operands)
```

### Memory Analysis

```go
// Get compaction report
report := vmInstance.GetCompactionReport()
for regionID, regionReport := range report {
    fmt.Printf("Region %d: %.2f%% fragmented\n", 
        regionID, regionReport.Fragmentation*100)
}

// Analyze compaction opportunities
config := allocator.DefaultCompactionConfig()
allocsToMove := vmInstance.AnalyzeCompaction(config)
fmt.Printf("Can move %d allocations to reduce fragmentation\n", 
    len(allocsToMove))
```

## API Reference

See [API.md](./API.md) for comprehensive API documentation including:

- VM3D interface
- Memory operations
- Gas metering
- Merkle proofing
- Compaction and analysis

## Examples

### Example 1: Simple Memory Allocation

```go
// See examples/simple_allocation.go
```

### Example 2: Memory Proofing

```go
// Create VM with Merkle tree enabled
config := vm.VMConfig{EnableProof: true, ...}
vmInstance, _ := vm.NewVM3D(config)

// Allocate and store data
addr, _ := vmInstance.AllocateMemory(256, types.RegionTypeHeap)
vmInstance.Store3D(addr, data)

// Generate and verify proof
proof, _ := vmInstance.GetMerkleProof(addr)
verified, _ := vmInstance.VerifyMemoryState(addr, data, proof)
```

### Example 3: Gas-Limited Execution

```go
// Create low-gas-budget VM for resource-constrained environments
config := vm.VMConfig{
    GasLimit:  100000, // Only 100K gas available
    ...
}
vmInstance, _ := vm.NewVM3D(config)

// Operations will fail when gas is exhausted
alloc, err := vmInstance.AllocateMemory(1024, types.RegionTypeHeap)
// err will be non-nil if gas exhausted
```

## Performance

### Benchmarks

Memory3D includes comprehensive benchmarks for:

- **Allocation operations**: Small, medium, large allocations
- **Memory operations**: Load, store, copy
- **Gas metering**: Consumption and refund tracking
- **Merkle operations**: Proof generation and verification
- **Compaction**: Analysis and execution

Run benchmarks:

```bash
go test -bench=. ./test/benchmarks/...
```

### Typical Performance (on i5-4200U @ 1.6GHz)

| Operation | Time (ns) | Notes |
| ----------- | ----------- | ------- |
| Small allocation | ~2,300 | 256 bytes |
| Store operation | ~1,400 | Single 256B write |
| Load operation | ~960 | Single 256B read |
| Merkle proof gen | ~500 | Single address |
| Gas operations | <200 | Simple tracking |

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

### Development Setup

```bash
# Clone repository
git clone https://github.com/rawbytedev/memory3d.git
cd memory3d

# Run tests
go test ./...

# Run benchmarks
go test -bench=. ./test/benchmarks/...

# Run integration tests
go test -v ./test/integration/...
```

## License

Memory3D is licensed under the Apache 2.0 License. See [LICENSE](./LICENSE) file for details.

## Acknowledgments

- Built with Go 1.25.2
- Testing framework: testify
- Memory model inspired by EVM and RISC-V

## Contact & Support

For issues, questions, or suggestions:

- **GitHub Issues**: [memory3d/issues](https://github.com/rawbytedev/memory3d/issues)
- **Documentation**: See docs/ directory

---

**Status**: Active Development | **Version**: 0.1.0
