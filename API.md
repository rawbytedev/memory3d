# Memory3D API Reference

Complete API documentation for Memory3D virtual machine and components.

## Table of Contents

1. [VM3D](#vm3d-virtual-machine)
2. [Memory Operations](#memory-operations)
3. [Gas Metering](#gas-metering)
4. [Merkle Proofing](#merkle-proofing)
5. [Memory Analysis](#memory-analysis)
6. [Types & Constants](#types--constants)
7. [Error Handling](#error-handling)

---

## VM3D (Virtual Machine)

The main entry point for all Memory3D operations.

### Creating a VM

```go
func NewVM3D(config VMConfig) (*VM3D, error)
```

**Parameters:**

- `config` (VMConfig) - Virtual machine configuration

**Returns:**

- `*VM3D` - New virtual machine instance
- `error` - Configuration error if any

**Example:**

```go
config := vm.VMConfig{
    MemorySize:       1024 * 1024 * 100,  // 100MB
    GasLimit:         10000000,            // 10M gas
    EnableProof:      true,                // Enable Merkle tree
    EnableCompaction: true,                // Enable compaction
    MaxInstructions:  100000,              // Max 100K instructions
    LogLevel:         vm.LogLevelInfo,     // Info logging
}

vmInstance, err := vm.NewVM3D(config)
if err != nil {
    log.Fatal("Failed to create VM:", err)
}
defer vmInstance.Shutdown()
```

### VMConfig

Configuration structure for VM initialization.

```go
type VMConfig struct {
    MemorySize       uint64   `json:"memory_size"`
    GasLimit         uint64   `json:"gas_limit"`
    EnableProof      bool     `json:"enable_proof"`
    EnableCompaction bool     `json:"enable_compaction"`
    MaxInstructions  uint64   `json:"max_instructions"`
    LogLevel         LogLevel `json:"log_level"`
}
```

**Fields:**

| Field | Type | Default | Purpose |
| ------- | ------ | --------- | --------- |
| MemorySize | uint64 | 100MB | Total addressable memory |
| GasLimit | uint64 | 10M | Total gas budget |
| EnableProof | bool | false | Enable Merkle tree proofs |
| EnableCompaction | bool | true | Enable memory compaction |
| MaxInstructions | uint64 | 100K | Maximum instructions to execute |
| LogLevel | LogLevel | LogLevelInfo | Logging verbosity |

**Log Levels:**

```go
const (
    LogLevelSilent LogLevel = iota   // No output
    LogLevelError                     // Errors only
    LogLevelWarn                      // Warnings and errors
    LogLevelInfo                      // Info, warnings, errors
    LogLevelDebug                     // Full debug output
)
```

### Shutdown

```go
func (vm *VM3D) Shutdown()
```

Cleanly shut down the VM and release resources.

**Example:**

```go
vmInstance, _ := vm.NewVM3D(config)
defer vmInstance.Shutdown()
```

### Statistics

```go
func (vm *VM3D) GetStats() VMStats
```

Get current VM statistics.

**Returns:** `VMStats` struct

```go
type VMStats struct {
    InstructionsExecuted uint64        // Total instructions run
    TotalGasUsed         uint64        // Gas consumed
    Allocations          uint64        // Number of allocations
    MemoryUsage          uint64        // Bytes allocated
    YPromotions          uint64        // Y-promotion count
    StartTime            time.Time     // VM start time
    RunTime              time.Duration // Total runtime
}
```

**Example:**

```go
stats := vmInstance.GetStats()
fmt.Printf("Instructions: %d\n", stats.InstructionsExecuted)
fmt.Printf("Gas Used: %d\n", stats.TotalGasUsed)
fmt.Printf("Runtime: %v\n", stats.RunTime)
```

### Execution

```go
func (vm *VM3D) Execute(program []byte) error
```

Execute a program bytecode.

**Parameters:**

- `program` ([]byte) - Bytecode to execute

**Returns:**

- `error` - Execution error, if any

**Errors:**

- "no more instructions" - Program counter exceeded program length
- "incomplete instruction" - Instruction is incomplete
- "unsupported opcode" - Unknown opcode encountered
- "gas exhausted" - Gas limit reached
- "instruction limit exceeded" - Max instructions exceeded

**Example:**

```go
program := []byte{
    0x12, 0x00, 0x00, 0x01, 0x00, // OP_MALLOC3D (256 bytes)
    0x11, 0x00, 0x00, 0x00, 0x00, // OP_MSTORE3D
    // ...
    0xFF, // OP_HALT3D
}

err := vmInstance.Execute(program)
if err != nil {
    log.Fatal("Execution failed:", err)
}
```

---

## Memory Operations

### Load3D - Read from Memory

```go
func (vm *VM3D) Load3D(addr Address3D, size uint32) ([]byte, error)
```

Read data from a 3D memory address.

**Parameters:**

- `addr` (Address3D) - Memory address
- `size` (uint32) - Number of bytes to read

**Returns:**

- `[]byte` - Read data
- `error` - If address not allocated or read fails

**Gas Cost:** `10 + (size * 2)`

**Example:**

```go
addr := types.Address3D{X: 0, Y: 0, Z: 0}
data, err := vmInstance.Load3D(addr, 256)
if err != nil {
    log.Fatal("Load failed:", err)
}
fmt.Printf("Read %d bytes\n", len(data))
```

### Store3D - Write to Memory

```go
func (vm *VM3D) Store3D(addr Address3D, data []byte) error
```

Write data to a 3D memory address.

**Parameters:**

- `addr` (Address3D) - Target memory address
- `data` ([]byte) - Data to write

**Returns:**

- `error` - If address not allocated, insufficient gas, or write fails

**Gas Cost:** `15 + (len(data) * 2)`

**Features:**

- Auto-allocates if address unallocated
- Updates Merkle tree if enabled
- Updates cache
- Tracks access statistics

**Example:**

```go
addr := types.Address3D{X: 0, Y: 0, Z: 0}
data := []byte("Hello, Memory3D!")
err := vmInstance.Store3D(addr, data)
if err != nil {
    log.Fatal("Store failed:", err)
}
```

### AllocateMemory - Request Memory Block

```go
func (vm *VM3D) AllocateMemory(size uint32, regionType types.RegionType) (Address3D, error)
```

Allocate a memory block.

**Parameters:**

- `size` (uint32) - Size in bytes
- `regionType` (RegionType) - Heap, Stack, Static, etc.

**Returns:**

- `Address3D` - Allocated address
- `error` - If allocation fails (out of memory, etc.)

**Gas Cost:** `20 + (size * 1)`

**Region Types:**

```go
const (
    RegionTypeHeap   RegionType = iota  // General purpose heap
    RegionTypeStack                     // Stack memory
    RegionTypeStatic                    // Static data
    RegionTypeShared                    // Shared memory
)
```

**Example:**

```go
// Allocate 1KB in heap
addr, err := vmInstance.AllocateMemory(1024, types.RegionTypeHeap)
if err != nil {
    log.Fatal("Allocation failed:", err)
}
fmt.Printf("Allocated at: %v\n", addr)
```

### Free3D - Release Memory

```go
func (vm *VM3D) Free3D(addr Address3D) error
```

Free a previously allocated memory block.

**Parameters:**

- `addr` (Address3D) - Address to free

**Returns:**

- `error` - If address not found or invalid

**Gas Cost:** `5`

**Example:**

```go
err := vmInstance.Free3D(addr)
if err != nil {
    log.Fatal("Free failed:", err)
}
```

### Copy3D - Copy Memory Block

```go
func (vm *VM3D) Copy3D(src, dst Address3D, size uint32) error
```

Copy memory from source to destination.

**Parameters:**

- `src` (Address3D) - Source address
- `dst` (Address3D) - Destination address
- `size` (uint32) - Bytes to copy

**Returns:**

- `error` - If addresses invalid or copy fails

**Gas Cost:** `25 + (size * 2)`

**Example:**

```go
srcAddr := types.Address3D{X: 0, Y: 0, Z: 0}
dstAddr := types.Address3D{X: 0, Y: 0, Z: 256}
err := vmInstance.Copy3D(srcAddr, dstAddr, 256)
if err != nil {
    log.Fatal("Copy failed:", err)
}
```

---

## Gas Metering

### GetGasRemaining - Check Available Gas

```go
func (vm *VM3D) GetGasRemaining() int
```

Get remaining gas budget.

**Returns:** `int` - Remaining gas units

**Example:**

```go
remaining := vmInstance.GetGasRemaining()
if remaining < 10000 {
    fmt.Println("Low on gas!")
}
```

### GetGasUsed - Check Consumed Gas

```go
func (vm *VM3D) GetGasUsed() uint64
```

Get total gas consumed.

**Returns:** `uint64` - Total gas used

**Example:**

```go
used := vmInstance.GetGasUsed()
fmt.Printf("Gas used: %d / %d\n", used, config.GasLimit)
```

### Gas Cost Reference

| Operation | Base Cost | Per-Unit Cost | Notes |
| ----------- | ----------- | --------------- | ------- |
| Load | 10 | 2/byte | Minimum 10 gas |
| Store | 15 | 2/byte | Minimum 15 gas |
| Allocate | 20 | 1/byte | Linear scaling |
| Free | 5 | - | Fixed cost |
| Copy | 25 | 2/byte | Size-dependent |
| Compact | 50 | 0.5/byte | Refund on success |
| Y-Promotion | 100 | - | Fixed |

---

## Merkle Proofing

### GetMerkleProof - Generate Proof

```go
func (vm *VM3D) GetMerkleProof(addr Address3D) ([][]byte, error)
```

Generate a Merkle proof for a memory address.

**Parameters:**

- `addr` (Address3D) - Address to prove

**Returns:**

- `[][]byte` - Merkle proof (hash chain)
- `error` - If Merkle tree not enabled or address not found

**Prerequisites:**

- Merkle tree must be enabled in config (`EnableProof: true`)
- Address must have been written to at least once

**Example:**

```go
config := vm.VMConfig{EnableProof: true, ...}
vmInstance, _ := vm.NewVM3D(config)

// Allocate and write data
addr, _ := vmInstance.AllocateMemory(256, types.RegionTypeHeap)
vmInstance.Store3D(addr, []byte("data"))

// Generate proof
proof, err := vmInstance.GetMerkleProof(addr)
if err != nil {
    log.Fatal("Proof generation failed:", err)
}
fmt.Printf("Proof length: %d\n", len(proof))
```

### VerifyMemoryState - Verify Proof

```go
func (vm *VM3D) VerifyMemoryState(addr Address3D, data []byte, proof [][]byte) (bool, error)
```

Verify a Merkle proof for memory content.

**Parameters:**

- `addr` (Address3D) - Memory address
- `data` ([]byte) - Expected data
- `proof` ([][]byte) - Merkle proof

**Returns:**

- `bool` - True if proof is valid
- `error` - If verification fails

**Example:**

```go
// Generate proof
proof, _ := vmInstance.GetMerkleProof(addr)

// Verify proof
verified, err := vmInstance.VerifyMemoryState(addr, data, proof)
if err != nil {
    log.Fatal("Verification failed:", err)
}

if verified {
    fmt.Println("Memory state verified!")
} else {
    fmt.Println("Memory state invalid!")
}
```

### GetMerkleRoot - Get Tree Root

```go
func (vm *VM3D) GetMerkleRoot() []byte
```

Get the current Merkle tree root hash.

**Returns:**

- `[]byte` - Root hash (32 bytes for SHA-256)
- `nil` - If Merkle tree not enabled

**Example:**

```go
root := vmInstance.GetMerkleRoot()
if root != nil {
    fmt.Printf("Root: %x\n", root)
}
```

---

## Memory Analysis

### GetCompactionReport - Analyze Fragmentation

```go
func (vm *VM3D) GetCompactionReport() map[uint64]types.CompactionReport
```

Get fragmentation analysis for all regions.

**Returns:**

- `map[uint64]CompactionReport` - Report per region ID

```go
type CompactionReport struct {
    RegionID          uint64
    TotalSize         uint64
    UsedSize          uint64
    FreeSize          uint64
    Fragmentation     float64     // 0.0 to 1.0
    FragmentCount     uint64
    MovableAllocations []uint64
}
```

**Example:**

```go
report := vmInstance.GetCompactionReport()
for regionID, regionReport := range report {
    fmt.Printf("Region %d:\n", regionID)
    fmt.Printf("  Fragmentation: %.2f%%\n", regionReport.Fragmentation*100)
    fmt.Printf("  Free: %d bytes\n", regionReport.FreeSize)
}
```

### AnalyzeCompaction - Determine Moves

```go
func (vm *VM3D) AnalyzeCompaction(config allocator.CompactionConfig) []uint64
```

Analyze which allocations should be moved for consolidation.

**Parameters:**

- `config` (CompactionConfig) - Compaction parameters

**Returns:**

- `[]uint64` - List of allocation IDs to move

**CompactionConfig:**

```go
type CompactionConfig struct {
    FragmentationThreshold  float64 // Min fragmentation to trigger (0.0-1.0)
    MinFragmentSize         uint32  // Minimum fragment size to consider
    MaxMovesPerCycle        uint32  // Max allocations to move
    GasRefundPercentage     float64 // Refund percentage on consolidation
}

func DefaultCompactionConfig() CompactionConfig {
    return CompactionConfig{
        FragmentationThreshold: 0.3,
        MinFragmentSize:        64,
        MaxMovesPerCycle:       100,
        GasRefundPercentage:    0.5,
    }
}
```

**Example:**

```go
config := allocator.DefaultCompactionConfig()
allocsToMove := vmInstance.AnalyzeCompaction(config)
fmt.Printf("Can move %d allocations\n", len(allocsToMove))
```

### ExecuteCompactInstruction - Trigger Compaction

```go
func (vm *VM3D) ExecuteCompactInstruction(inst *Instruction3D) error
```

Execute a compaction instruction.

**Parameters:**

- `inst` (Instruction3D) - Compaction instruction

**Returns:**

- `error` - If compaction fails

**Example:**

```go
inst := &vm.Instruction3D{
    Opcode:   vm.OP_MCOMPACT3D,
    Operands: []vm.Operand{},
    Size:     1,
}

err := vmInstance.ExecuteCompactInstruction(inst)
if err != nil {
    log.Fatal("Compaction failed:", err)
}
```

---

## Arithmetic & Register Operations

### ADD3D - Register Addition

**Opcode:** `0x31`  
**Format:** `[opcode(1)][destReg(1)][srcReg1(1)][srcReg2(1)]` (4 bytes)  
**Gas Cost:** 3

Adds two 64-bit register values and stores the result in destination register.

**Operands:**

- `destReg` - Destination register (R0-R15)
- `srcReg1` - First source register (R0-R15)
- `srcReg2` - Second source register (R0-R15)

**Behavior:** `destReg = srcReg1 + srcReg2` (64-bit addition)

**Example:**

```go
// Build ADD3D instruction: R2 = R0 + R1
program := []byte{
    0x31, // OP_ADD3D
    0x02, // Destination: R2
    0x00, // Source 1: R0
    0x01, // Source 2: R1
}
```

### SUB3D - Register Subtraction

**Opcode:** `0x32`  
**Format:** `[opcode(1)][destReg(1)][srcReg1(1)][srcReg2(1)]` (4 bytes)  
**Gas Cost:** 3

Subtracts two 64-bit register values and stores the result in destination register.

**Operands:**

- `destReg` - Destination register (R0-R15)
- `srcReg1` - Minuend register (R0-R15)
- `srcReg2` - Subtrahend register (R0-R15)

**Behavior:** `destReg = srcReg1 - srcReg2` (64-bit subtraction)

**Example:**

```go
// Build SUB3D instruction: R3 = R0 - R1
program := []byte{
    0x32, // OP_SUB3D
    0x03, // Destination: R3
    0x00, // Minuend: R0
    0x01, // Subtrahend: R1
}
```

### MOV3D - Register Move

**Opcode:** `0x30`  
**Format:** `[opcode(1)][destReg(1)][srcReg(1)][padding(5)]` (8 bytes)  
**Gas Cost:** 2

Moves data from source register to destination register.

**Operands:**

- `destReg` - Destination register (R0-R15)
- `srcReg` - Source register (R0-R15)

**Behavior:** `destReg = srcReg` (data copy)

**Example:**

```go
// Build MOV3D instruction: R1 = R0
program := []byte{
    0x30, // OP_MOV3D
    0x01, // Destination: R1
    0x00, // Source: R0
    0x00, 0x00, 0x00, 0x00, 0x00, // Padding
}
```

### MSIZE3D - Query Allocation Size

**Opcode:** `0x15`  
**Format:** `[opcode(1)][x(8)][y(4)][z(2)][destReg(1)]` (16 bytes)  
**Gas Cost:** 5

Queries the allocation size at a 3D memory address and stores result in destination register.

**Operands:**

- `addr` - 3D address (X: region, Y: plane, Z: offset)
- `destReg` - Destination register for size result (R0-R15)

**Behavior:**

- If address is allocated: `destReg = allocation.Size`
- If address is not allocated: `destReg = 0`

**Example:**

```go
// Build MSIZE3D instruction to query allocation at (0, 0, 0) into R0
program := []byte{
    0x15,                // OP_MSIZE3D
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // X = 0
    0x00, 0x00, 0x00, 0x00, // Y = 0
    0x00, 0x00, // Z = 0
    0x00, // Destination: R0
}
```

---

## Control Flow Operations

### NOP - No Operation

**Opcode:** `0x00`  
**Format:** `[opcode(1)]` (1 byte)  
**Gas Cost:** 1

No operation instruction. Useful for padding or timing loops.

**Behavior:** Does nothing, advances PC by 1

**Example:**

```go
// Build NOP instruction
program := []byte{
    0x00, // OP_NOP
}
```

### HALT3D - Halt Execution

**Opcode:** `0xFF`  
**Format:** `[opcode(1)]` (1 byte)  
**Gas Cost:** 1

Halts program execution immediately. Sets execution flag to stop the VM loop.

**Behavior:** Sets halted flag and returns from Execute()

**Example:**

```go
// Build program with NOP and HALT
program := []byte{
    0x00, // OP_NOP
    0xFF, // OP_HALT3D - execution stops here
    0x00, // This instruction never executes
}
```

---

## Types & Constants

### Address3D

3D memory address.

```go
type Address3D struct {
    X uint64 // Region (64 bits)
    Y uint32 // Plane (32 bits)
    Z uint16 // Offset (16 bits)
}

// Constructor
func NewAddress(x uint64, y uint32, z uint16) Address3D

// Validation
func (a Address3D) IsValid() bool

// Serialization
func (a Address3D) Bytes() []byte
func (a Address3D) String() string

// Operations
func (a Address3D) AddZ(offset uint16) Address3D
func (a Address3D) NextY() Address3D
func (a Address3D) Compare(b Address3D) int
func (a Address3D) ManhattanDistance(b Address3D) uint64
```

### RegionType

Memory region classification.

```go
type RegionType uint8

const (
    RegionTypeHeap   RegionType = iota  // Heap memory
    RegionTypeStack                     // Stack memory
    RegionTypeStatic                    // Static data
    RegionTypeShared                    // Shared memory
)
```

### AllocationFlags

Allocation modifier flags.

```go
type AllocationFlags uint8

const (
    FlagReadOnly      AllocationFlags = 1 << iota
    FlagExecutable
    FlagProtected
    FlagTemporary
)
```

---

## Error Handling

### Common Errors

```go
"no more instructions"           // Program counter exceeded
"incomplete instruction"         // Truncated instruction
"unsupported opcode"            // Unknown opcode
"gas exhausted"                 // Gas limit exceeded
"instruction limit exceeded"    // Max instructions exceeded
"address not allocated"         // Address not found
"insufficient memory"           // Out of memory
"allocation failed"             // Allocation error
"merkle tree not enabled"       // Proof operations without tree
"invalid address"               // Address out of bounds
"access denied"                 // Permission denied
```

### Error Handling Pattern

```go
if err != nil {
    // Type assertion for specific errors
    if strings.Contains(err.Error(), "gas exhausted") {
        fmt.Println("Out of gas")
    } else if strings.Contains(err.Error(), "address not allocated") {
        fmt.Println("Invalid address")
    }
    // Handle error appropriately
    return err
}
```

---

## Performance Notes

### Optimal Parameters

```go
config := vm.VMConfig{
    MemorySize:       1024 * 1024 * 100,  // 100MB for most workloads
    GasLimit:         10000000,            // 10M gas typical
    EnableProof:      true,                // Enable if verification needed
    EnableCompaction: true,                // Always enable
    MaxInstructions:  100000,              // Adjust per use case
    LogLevel:         vm.LogLevelWarn,     // Info level for dev
}
```

### Benchmarking

```go
// Time operations
start := time.Now()
addr, _ := vmInstance.AllocateMemory(1024, types.RegionTypeHeap)
elapsed := time.Since(start)
fmt.Printf("Allocation: %v\n", elapsed)
```

---

**API Version:** 0.2.0
