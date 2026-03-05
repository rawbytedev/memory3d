# Memory3D Architecture Guide

## Table of Contents

1. [System Overview](#system-overview)
2. [Memory Model](#memory-model)
3. [Core Components](#core-components)
4. [Allocation Strategy](#allocation-strategy)
5. [Gas Metering](#gas-metering)
6. [Merkle Tree System](#merkle-tree-system)
7. [Memory Compaction](#memory-compaction)
8. [Concurrency Model](#concurrency-model)
9. [Performance Characteristics](#performance-characteristics)

## System Overview

Memory3D is a sophisticated memory management system designed for:

- **Spatial locality**: Organizing memory in hierarchical 3D dimensions
- **Cryptographic verification**: Merkle tree-based memory proofing
- **Resource metering**: Gas-based operation cost tracking
- **Intelligent consolidation**: Advanced fragmentation reduction

The system provides a VM-like interface where programs can allocate, access, and free memory while maintaining cryptographic guarantees about memory state.

## Memory Model

### 3D Address Space

```txt
Address = [X (Region), Y (Plane), Z (Offset)]
         [64-bit]      [32-bit]    [16-bit]
```

**Memory Hierarchy:**

```arch
System Memory
    ├── Region 0 (X=0)
    │   ├── Plane 0 (Y=0)
    │   │   ├── [Z=0000..0001] - Allocation 1
    │   │   ├── [Z=0100..0256] - Allocation 2
    │   │   └── ...
    │   ├── Plane 1 (Y=1)
    │   │   └── ...
    │   └── ...
    ├── Region 1 (X=1)
    │   └── ...
    └── ...
```

### Key Constants

| Constant | Value | Purpose |
| ---------- | ------- | --------- |
| MaxXRegions | 256 | Total number of regions |
| MaxYPlanes | 65,535 | Planes per region |
| MaxZOffset | 65,535 | Bytes per plane |
| PlaneSize | 65 KB | Physical plane capacity |

### Address Features

- **Byte serialization**: 14 bytes total (8+4+2)
- **Comparison**: Lexicographic ordering (X → Y → Z)
- **Validation**: Bounds checking via `IsValid()`
- **Serialization**: Binary and string formats

## Core Components

### 1. VM3D (Virtual Machine)

**Responsibilities:**

- Orchestrate all subsystems
- Execute instruction sequences
- Manage program counter and stack
- Track statistics and metrics

**Key Methods:**

```go
func (vm *VM3D) Execute(program []byte) error
func (vm *VM3D) Load3D(addr Address3D, size uint32) ([]byte, error)
func (vm *VM3D) Store3D(addr Address3D, data []byte) error
func (vm *VM3D) AllocateMemory(size uint32, regionType RegionType) (Address3D, error)
```

**State Management:**

```go
type VM3D struct {
    allocator  *Allocator3D      // Memory allocation
    gasAccount *GasAccountant3D  // Gas tracking
    merkleTree *Tree3D           // Proof system
    registers  *Registers        // CPU registers
    memory     *MemoryManager    // Memory abstraction
    cache      *SpatialCache     // Locality cache
    pc         uint64            // Program counter
    stats      VMStats           // Statistics
}
```

### 2. Allocator3D

**Responsibilities:**

- Allocate regions and planes
- Maintain free space information
- Track Y-promotions
- Manage region LRU cache

**Architecture:**

```arch
Allocator3D
├── Region Pool
│   ├── Region 0
│   │   ├── Plane 0 (bitmap)
│   │   ├── Plane 1 (bitmap)
│   │   └── L1/L2 Cache
│   └── ...
├── Region LRU (Least Recently Used)
├── Y-Promotion Engine
└── Statistics
```

**Allocation Strategy:**

1. **Fast Path**: Check LRU regions for available space
2. **Y-Promotion**: Consolidate fragments to higher planes
3. **New Region**: Allocate new region if needed

**Key Data Structures:**

```go
type Region struct {
    ID           uint64
    Type         RegionType
    Planes       []*Plane
    FreeBytes    uint64
    AllocCount   uint64
    RWMutex      sync.RWMutex
}

type Plane struct {
    ID           uint32
    Bitmap       []byte      // Allocation bitmap
    Allocations  map[uint16]*Allocation
    FreeBytes    uint16
}

type Allocation struct {
    ID       uint64
    Address  Address3D
    Size     uint32
    Type     RegionType
    Flags    AllocationFlags
}
```

### 3. MemoryManager

**Responsibilities:**

- Provide virtual memory abstraction
- Track allocations
- Enforce bounds checking
- Update access statistics

**Interface:**

```go
func (m *MemoryManager) Load3D(addr Address3D, size uint32) ([]byte, error)
func (m *MemoryManager) Store3D(addr Address3D, data []byte) error
func (m *MemoryManager) Copy3D(src, dst Address3D, size uint32) error
func (m *MemoryManager) UpdateAccess(addr Address3D)
```

### 4. GasAccountant3D

**Responsibilities:**

- Track gas consumption
- Implement refund mechanism
- Enforce gas limits
- Calculate operation costs

**Gas Model:**

```arch
Total Gas = Base + Per-Byte Cost + Context Overhead

Examples:
- Memory Load:  10 + (size * 2) gas
- Memory Store: 15 + (size * 2) gas
- Allocation:   20 + (size * 1) gas
- Y-Promotion:  100 (fixed)
```

**Key Methods:**

```go
func (g *GasAccountant3D) Consume(amount uint64) error
func (g *GasAccountant3D) Refund(amount uint64)
func (g *GasAccountant3D) Remaining() int
func (g *GasAccountant3D) Used() uint64
```

### 5. Tree3D (Merkle Tree)

**Responsibilities:**

- Maintain cryptographic hash of memory state
- Generate membership proofs
- Verify proof validity
- Update on memory write

**Architecture:**

```arch
Merkle Tree
├── Root Hash
├── Internal Nodes
│   ├── Branch 1
│   │   ├── Leaf Hash (Addr 1)
│   │   └── Leaf Hash (Addr 2)
│   └── Branch 2
└── Leaf Nodes (Memory)
```

**Operations:**

```go
func (t *Tree3D) Update(addr Address3D, data []byte)
func (t *Tree3D) GenerateProof(addr Address3D) ([][]byte, error)
func (t *Tree3D) VerifyProof(addr Address3D, proof [][]byte, data []byte) (bool, error)
func (t *Tree3D) GetRoot() []byte
```

### 6. SpatialCache

**Responsibilities:**

- Cache frequently accessed addresses
- Exploit locality patterns
- Reduce main memory access latency
- Track cache statistics

**Features:**

- LRU eviction policy
- Configurable size
- Hit/miss tracking
- Spatial locality optimization

## Allocation Strategy

### Multi-Level Allocation

```arch
Level 1: Fast Path (Existing Regions)
  - Check LRU regions for available space
  - O(k) where k = # regions in LRU
  - Typical hit rate: 90%+

Level 2: Y-Promotion (Fragment Consolidation)
  - Analyze fragmenting regions
  - Promote fragments to higher planes
  - Recovers ~20% fragmented space

Level 3: New Region
  - Allocate new region when needed
  - More expensive: O(n) setup
  - Rare in steady state
```

### LRU Management

Regions are tracked in an LRU list for fast allocation:

```arch
Recent Regions (frequently used)
├── Region 42 ← most recent
├── Region 15
├── Region 8
└── Region 1 ← least recent (evicted first)

Advance: Used region moves to front
Evict:   Least recently used region could be evicted
```

## Gas Metering

### Operation Costs

| Operation | Cost Formula |
| ----------- | -------------- |
| Load | 10 + (size * 2) |
| Store | 15 + (size * 2) |
| Allocate | 20 + (size * 1) |
| Free | 5 |
| Copy | 25 + (size * 2) |
| Compact | 50 + (consolidated_bytes * 0.5) |

### Cost Calculation

```go
func (vm *VM3D) calculateGasCost(inst *Instruction3D) uint64 {
    baseCost := map[Opcode]uint64{
        OP_MLOAD3D:  10,
        OP_MSTORE3D: 15,
        OP_MALLOC3D: 20,
        // ...
    }
    
    cost := baseCost[inst.Opcode]
    
    // Add per-operand costs
    for _, operand := range inst.Operands {
        if operand.Type == OT_SIZE {
            cost += uint64(operand.Value.(uint32)) * 2
        }
    }
    
    return cost
}
```

### Refund Mechanism

Compaction recovery:

```txt
Bytes Compacted × Gas Refund Percentage = Gas Recovered
Example: 10KB compacted × 50% = 5120 gas refunded
```

## Merkle Tree System

### Purpose

Provide cryptographic proof of memory contents without revealing data:

```txt
User: "Prove that address X contains hash H"
VM:   [generates Merkle proof]
User: [verifies proof against published root]
Result: Proof is valid → address X contains correct data
```

### Proof Generation

```go
proof, err := vm.GetMerkleProof(address3D)
// proof = [hash1, hash2, ..., hashN]
// Each hash is a Merkle branch needed for verification
```

### Proof Verification

```go
verified, err := vm.VerifyMemoryState(address, data, proof)
// verified = true if merkle proof is valid
// Data integrity + root commitment = trustless verification
```

### Root Commitment

```go
root := vm.GetMerkleRoot()
// Root is published/committed
// Later proofs can be verified against this root
// Prevents tampering with tree after commitment
```

## Memory Compaction

### Fragmentation Analysis

The system analyzes fragmentation across regions:

```go
report := vm.GetCompactionReport()
// Returns for each region:
// - Fragmentation percentage
// - Free fragments
// - Potential consolidation
```

### Y-Promotion Strategy

Consolidates allocations to reduce fragmentation:

```arch
Before Y-Promotion:
Plane 0: [Alloc1] [Free] [Alloc2] [Free] [Alloc3]
Plane 1: [Free...........................]

After Y-Promotion:
Plane 0: [Free...........................]
Plane 1: [Alloc1][Alloc2][Alloc3][Free....]

Benefits: Fewer regions needed, better cache locality
```

### Compaction Process

1. **Analyze**: Identify fragmented regions
2. **Plan**: Determine which allocations to move
3. **Execute**: Move allocations to consolidated planes
4. **Refund**: Return gas savings to executor

## Concurrency Model

### Thread Safety

All operations are thread-safe via RWMutex:

```go
type VM3D struct {
    mu sync.RWMutex      // Read-Write lock
    // ...
}

// Read operations (concurrent)
func (vm *VM3D) Load3D(...) {
    vm.mu.RLock()
    defer vm.mu.RUnlock()
    // ...
}

// Write operations (exclusive)
func (vm *VM3D) Store3D(...) {
    vm.mu.Lock()
    defer vm.mu.Unlock()
    // ...
}
```

### Lock Ordering

To prevent deadlocks, lock order is strictly maintained:

```txt
1. VM lock (vm.mu)
2. Region lock (region.RWMutex)
3. LRU lock (allocator.lruMutex)

Always acquire in this order, never reverse.
```

### Deadlock Prevention

- **Release before re-acquire**: Always unlock before taking higher-level locks
- **Non-blocking operations**: LRU promotion happens after region release
- **Atomic updates**: Use atomics for counters to avoid locking

## Performance Characteristics

### Time Complexity

| Operation | Typical | Worst Case |
| ----------- | --------- | ----------- |
| Allocate | O(k) | O(n) |
| Deallocate | O(1) | O(1) |
| Load | O(1) | O(1) |
| Store | O(1) | O(1) |
| Compact | O(k) | O(n) |
| Merkle Proof | O(log n) | O(log n) |

Where: k = regions in LRU, n = total regions

### Space Complexity

| Component | Space | Notes |
| ----------- | ------- | ------- |
| Regions | O(n) | n = # regions |
| Planes | O(n×m) | m = planes per region |
| Allocations | O(a) | a = # allocations |
| Merkle Tree | O(n) | n = # addresses |
| Cache | O(c) | c = cache size |

### Cache Behavior

- **L1 Cache**: Spatial cache (~100-1000 entries)
- **L2 Cache**: Region LRU (~10-100 regions)
- **Memory**: Plane-based allocation
- **Hit rates**: 80-95% typical

## Instruction Set

### Memory Operations

```go
OP_MLOAD3D   // Load from 3D address
OP_MSTORE3D  // Store to 3D address
OP_MALLOC3D  // Allocate memory
OP_MFREE3D   // Free memory
OP_MCOPY3D   // Copy memory block
```

### Spatial Operations

```go
OP_MQUERY3D   // Query allocation info
OP_MCOMPACT3D // Trigger compaction
OP_MDEFRAG3D  // Trigger defragmentation
```

### Gas Operations

```go
OP_MGAS3D    // Get remaining gas
OP_MREFUND3D // Trigger refund
```

### Control Flow

```go
OP_JUMP3D  // Jump to instruction
OP_CALL3D  // Call subroutine
OP_RET3D   // Return from subroutine
OP_HALT3D  // Stop execution
```

## Extensibility

### Custom Region Types

New region types can be defined for different purposes:

```go
const (
    RegionTypeHeap   RegionType = iota
    RegionTypeStack
    RegionTypeStatic
    RegionTypeSpecial // Custom type
)
```

### Custom Gas Models

Gas calculations can be customized per operation type:

```go
// Override default costs
customCosts := map[Opcode]uint64{
    OP_MLOAD3D:  5,  // Lower cost
    OP_MSTORE3D: 10, // Lower cost
}
```

---

**Version**: 0.2
