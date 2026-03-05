# FAQ - Frequently Asked Questions

Common questions about Memory3D and their answers.

## General Questions

### What is Memory3D?

Memory3D is a sophisticated 3D virtual machine designed for efficient memory management with cryptographic verification capabilities. It extends traditional linear memory models into three dimensions (Regions, Planes, Offsets) for better spatial organization and features like Merkle tree proofs and gas metering.

### Who should use Memory3D?

Memory3D is suitable for:

- Systems requiring cryptographic memory proofs
- Applications with complex memory hierarchies
- Gas-metered execution environments
- Distributed systems needing memory verification
- Research in memory management architectures

### Is Memory3D production-ready?

Memory3D is at version 0.1.0. However, production use should include:

- Thorough testing in your environment
- Security review if handling sensitive data
- Performance benchmarking for your workload
- Monitoring and alerting setup

### What are the memory limits?

```txt
Maximum Addressable Space = 256 × 65,535 × 65,535 bytes
                          ≈ 1.1 exabytes

Practical limits depend on:
- System RAM available
- Configuration settings
- Workload patterns
```

## Installation & Setup

### How do I install Memory3D?

```bash
go get github.com/rawbytedev/memory3d
```

### What Go version is required?

Go 1.25.2 or later.

### Do I need external dependencies?

Only `testify` for testing. Core library has no external dependencies.

### Can I use Memory3D in production?

Yes, with appropriate testing and validation. See the [Contributing Guide](./CONTRIBUTING.md) for development practices.

## Usage Questions

### How do I create my first VM?

See [Getting Started](./GETTING_STARTED.md) for a complete tutorial. Quick example:

```go
config := vm.VMConfig{
    MemorySize:  100 * 1024 * 1024,  // 100MB
    GasLimit:    10000000,
    EnableProof: true,
}
vmInstance, _ := vm.NewVM3D(config)
defer vmInstance.Shutdown()
```

### What are the best practices for memory allocation?

1. **Allocate early**: Pre-allocate memory blocks in initialization
2. **Use pools**: Implement allocation pools for fixed-size blocks
3. **Clean up promptly**: Always free memory when done
4. **Monitor fragmentation**: Use compaction reports regularly
5. **Batch operations**: Group allocations together

### How much gas should I budget?

Typical operations:

```txt
Load 256 bytes:    10 + 512 = 522 gas
Store 256 bytes:   15 + 512 = 527 gas
Allocate 256 bytes: 20 + 256 = 276 gas
Free:              5 gas

Example workload (100 allocations + 100 stores + 100 loads):
≈ 27,600 + 52,700 + 52,200 = 132,500 gas
```

### Can I dynamically resize allocated blocks?

No, allocations are fixed-size. To resize:

```go
// Old block
oldAddr, _ := vm.AllocateMemory(256, types.RegionTypeHeap)
vm.Store3D(oldAddr, oldData)

// New block (larger)
newAddr, _ := vm.AllocateMemory(512, types.RegionTypeHeap)
vm.Copy3D(oldAddr, newAddr, 256)
vm.Free3D(oldAddr)
```

### How does the 3D addressing work?

```txt
Address = [Region (X), Plane (Y), Offset (Z)]
          [64-bit]     [32-bit]    [16-bit]

Example: Address3D{X: 0, Y: 0, Z: 0}
- Region 0, Plane 0, Byte 0

Each Region contains up to 65K Planes
Each Plane contains up to 64KB of data
```

## Performance Questions

### Why is my VM slower than expected?

Potential causes:

1. **High fragmentation**: Check with `GetCompactionReport()`
2. **Gas calculations**: Verify gas costs for your operations
3. **Lock contention**: Check if multiple threads access VM
4. **Cache misses**: Monitor spatial locality patterns
5. **Allocation strategy**: Review allocation patterns

### How can I optimize performance?

1. **Enable compaction**: Set `EnableCompaction: true`
2. **Use larger blocks**: Fewer allocations = better throughput
3. **Batch operations**: Group reads/writes together
4. **Pool memory**: Reuse allocated blocks
5. **Profile**: Use `pprof` to identify bottlenecks

### What's the overhead of Merkle proofs?

```txt
Generation: ~500ns per address
Verification: Depends on proof path length, typically <10µs
Memory: ~32 bytes per leaf (SHA-256 hash)
```

### Should I enable proofs in production?

Only if you need cryptographic verification:

- **Enable if**: Need proof of memory state
- **Disable if**: Don't need verification (slight performance gain)

## Memory Management

### What's the difference between regions and planes?

```txt
Region: Top-level memory container (256 max)
  └── Plane: Sub-container holding actual data (65K per region)
      └── Allocations: Individual memory blocks
```

### What's Y-Promotion?

Y-Promotion consolidates fragmented allocations:

```txt
Before: Plane 0: [A] [gap] [B] [gap] [C]
After:  Plane 0: [gap] [gap] [gap] [gap] [gap]
        Plane 1: [A][B][C][gap]...

Benefits: Better cache locality, fewer regions needed
```

### How does compaction work?

1. **Analyze**: Scan for fragmented regions
2. **Plan**: Identify allocations to move
3. **Execute**: Move allocations to consolidated planes
4. **Refund**: Return gas savings

Use `AnalyzeCompaction()` to preview without executing.

### Can I defragment manually?

Yes, use compaction instruction:

```go
inst := &vm.Instruction3D{
    Opcode: vm.OP_MCOMPACT3D,
}
vm.ExecuteCompactInstruction(inst)
```

## Concurrency

### Is Memory3D thread-safe?

Yes, with provisos:

- All public methods are thread-safe
- Uses RWMutex for synchronization
- Safe for concurrent reads
- Serialized writes

```go
// Safe: Multiple goroutines can read
go func() { data, _ := vm.Load3D(addr, 256) }()

// Safe: Multiple goroutines can write (serialized)
go func() { vm.Store3D(addr, data) }()
```

### Can I use one VM from multiple threads?

Yes, but writes are serialized internally. For true parallel execution, use multiple VMs:

```go
vm1, _ := vm.NewVM3D(config)  // Thread 1
vm2, _ := vm.NewVM3D(config)  // Thread 2
// Each has own memory space
```

### What are deadlock prevention measures?

Memory3D uses strict lock ordering:

1. VM lock
2. Region lock
3. LRU lock

Locks are released before re-acquiring to prevent circular waits.

## Troubleshooting

### How do I debug allocation issues?

```go
// Get detailed stats
stats := vm.GetStats()
fmt.Printf("Allocations: %d\n", stats.Allocations)
fmt.Printf("Memory usage: %d\n", stats.MemoryUsage)

// Get compaction report
report := vm.GetCompactionReport()
for _, r := range report {
    fmt.Printf("Fragmentation: %.2f%%\n", r.Fragmentation*100)
}
```

### Why do I get "gas exhausted"?

Each operation costs gas:

```bash
# Check remaining gas
remaining := vm.GetGasRemaining()

# Increase limit in config
config.GasLimit = 100000000  // Higher limit
```

### How do I receive error details?

Wrap errors with context:

```go
if err != nil {
    log.Printf("Operation failed: %v", err)
    // Error message includes context
}
```

### Can I use different region types?

Yes, region types affect allocation strategy:

```go
addr1, _ := vm.AllocateMemory(256, types.RegionTypeHeap)     // General
addr2, _ := vm.AllocateMemory(256, types.RegionTypeStack)    // Stack
addr3, _ := vm.AllocateMemory(256, types.RegionTypeStatic)   // Static
```

## Security Questions

### Is Memory3D suitable for untrusted input?

With caveats:

- Validate input sizes before allocation
- Set reasonable gas limits
- Use separate VM instances per tenant
- Monitor execution time

```go
// Validate before use
if userSize > MaxAllowedSize {
    return fmt.Errorf("allocation too large")
}
```

### How secure are Merkle proofs?

Merkle proofs provide:

- **Integrity**: Detect any tampering
- **Authenticity**: Verify state at specific point
- **Non-repudiation**: Prover can't deny later

Security depends on:

- Root hash publication/commitment
- Hash algorithm strength (SHA-256)
- Proof chain validation

### Should I trust gas costs?

Yes, gas costs are deterministic based on:

- Operation type
- Operation size
- Fixed overhead costs

Gas provides:

- Resource protection
- DoS attack prevention
- Fair resource allocation

## Development Questions

### How do I run tests?

```bash
# All tests
go test ./...

# Specific package
go test ./internal/allocator/...

# With coverage
go test -cover ./...

# Benchmarks
go test -bench=. ./test/benchmarks/...
```

### How do I contribute?

See [Contributing Guide](./CONTRIBUTING.md) for full details:

1. Fork repository
2. Create feature branch
3. Make changes with tests
4. Submit pull request
5. Respond to review feedback

### What's the development roadmap?

**Priority 3**: Performance optimizations
**Priority 4**: Extended instruction set
**Priority 5**: External integrations

See [Changelog](./CHANGELOG.md) for detailed roadmap.

### How do I report a bug?

1. Check existing issues (no duplicates)
2. Include:
   - Steps to reproduce
   - Expected behavior
   - Actual behavior
   - System info (OS, Go version)
3. Attach minimal reproducible example

For security bugs, [email](radiationbolt@gmail.com)

## API Questions

### Where's the complete API reference?

See [API.md](./API.md) for comprehensive reference including:

- VM creation and configuration
- Memory operations
- Gas metering
- Merkle proofing
- Analysis functions

### How do I use the instruction set?

Most uses should go through the VM API. Low-level instruction execution:

```go
inst := &vm.Instruction3D{
    Opcode:   vm.OP_MLOAD3D,
    Operands: []vm.Operand{...},
    Size:     20,
}
err := vm.executeInstruction(inst)
```

### Can I extend the instruction set?

Yes, by:

1. Adding new opcodes
2. Implementing execution functions
3. Calculating appropriate gas costs
4. Adding tests and documentation

See [Contributing Guide](./CONTRIBUTING.md).

## License & Legal

### What license is Memory3D under?

Apache 2.0 License - see [LICENSE](./LICENSE) file.

### Are there any patents?

No patents on Memory3D core technology.

### Can I use Memory3D in proprietary software?

Yes, Apache 2.0 License permits commercial use. Just include license notice.

---

**Still have questions?**

- Open an [Issue](https://github.com/rawbytedev/memory3d/issues)
- Check [API Reference](./API.md)
- See [Examples](./examples/)
- Read [Architecture](./ARCHITECTURE.md)
