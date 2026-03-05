package vm

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rawbytedev/memory3d/internal/allocator"
	"github.com/rawbytedev/memory3d/internal/merkle"
	"github.com/rawbytedev/memory3d/internal/types"
)

// LogLevel defines verbosity of VM logging
type LogLevel byte

const (
	LogLevelSilent LogLevel = iota
	LogLevelError
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

type VM3D struct {
	// Core components
	allocator  *allocator.Allocator3D
	gasAccount *GasAccountant3D
	merkleTree *merkle.Tree3D
	registers  *Registers

	// Memory
	memory *MemoryManager
	cache  *SpatialCache

	// Execution state
	pc        uint64 // Program counter
	stack     []Frame
	callStack []uint64
	halted    bool   // Flag to indicate if VM has halted

	// Statistics
	stats   VMStats
	metrics VMMetrics

	// Configuration
	config VMConfig

	// Concurrency
	mu       sync.RWMutex
	haltChan chan struct{}
	paused   bool
}

type VMConfig struct {
	MemorySize       uint64   `json:"memory_size"`
	GasLimit         uint64   `json:"gas_limit"`
	EnableProof      bool     `json:"enable_proof"`
	EnableCompaction bool     `json:"enable_compaction"`
	MaxInstructions  uint64   `json:"max_instructions"`
	LogLevel         LogLevel `json:"log_level"`
}

type VMStats struct {
	InstructionsExecuted uint64        `json:"instructions_executed"`
	TotalGasUsed         uint64        `json:"total_gas_used"`
	Allocations          uint64        `json:"allocations"`
	MemoryUsage          uint64        `json:"memory_usage"`
	YPromotions          uint64        `json:"y_promotions"`
	StartTime            time.Time     `json:"start_time"`
	RunTime              time.Duration `json:"run_time"`
}

type VMMetrics struct {
	InstructionLatency []time.Duration `json:"-"`
	CacheHitRate       float64         `json:"cache_hit_rate"`
	Fragmentation      float64         `json:"fragmentation"`
	GasEfficiency      float64         `json:"gas_efficiency"`
}

type Frame struct {
	ReturnPC     uint64
	Registers    Registers
	Memory       []byte
	GasRemaining uint64
}

func NewVM3D(config VMConfig) (*VM3D, error) {
	allocConfig := allocator.Config{
		PlaneSize:        types.PlaneSize,
		MaxYPlanes:       1024,
		MaxYPromotions:   8,
		WorkerCount:      4,
		EnableConcurrent: false, // Disabled for stability; optimize in future
	}

	vm := &VM3D{
		allocator:  allocator.NewAllocator3D(allocConfig),
		gasAccount: NewAccountant(config.GasLimit),
		registers:  NewRegisters(),
		memory:     NewMemoryManager(),
		cache:      NewSpatialCache(),
		config:     config,
		haltChan:   make(chan struct{}),
		stats: VMStats{
			StartTime: time.Now(),
		},
	}

	// Link memory manager to allocator for region/plane access
	vm.memory.allocator = vm.allocator
	if config.EnableProof {
		vm.merkleTree = merkle.NewTree3D()
	}

	// Log if allocator is running in concurrent mode
	if vm.allocator != nil && vm.allocator.IsConcurrent() {
		fmt.Println("VM3D: allocator concurrency ENABLED for this VM")
	}

	return vm, nil
}

// Shutdown cleanly shuts down the VM and releases resources
func (vm *VM3D) Shutdown() {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.allocator.Shutdown()
}

// IsAllocatorConcurrent exposes whether the VM's allocator is running with workers
func (vm *VM3D) IsAllocatorConcurrent() bool {
	if vm.allocator == nil {
		return false
	}
	return vm.allocator.IsConcurrent()
}

func (vm *VM3D) Execute(program []byte) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	vm.pc = 0
	vm.halted = false
	vm.stats.StartTime = time.Now()

	for vm.pc < uint64(len(program)) && !vm.halted {
		if vm.paused {
			select {
			case <-vm.haltChan:
				return nil
			default:
				time.Sleep(time.Millisecond)
				continue
			}
		}

		// Fetch instruction
		inst, err := vm.fetchInstruction(program)
		if err != nil {
			return fmt.Errorf("fetch error at PC=%d: %w", vm.pc, err)
		}

		// Execute instruction
		start := time.Now()
		err = vm.executeInstruction(inst)
		latency := time.Since(start)

		// Update statistics
		vm.stats.InstructionsExecuted++
		vm.metrics.InstructionLatency = append(vm.metrics.InstructionLatency, latency)

		if err != nil {
			return fmt.Errorf("execution error at PC=%d: %w", vm.pc, err)
		}

		// Check gas
		if vm.gasAccount.Remaining() == 0 {
			return fmt.Errorf("gas exhausted")
		}

		// Check instruction limit
		if vm.stats.InstructionsExecuted >= vm.config.MaxInstructions {
			return fmt.Errorf("instruction limit exceeded")
		}

		vm.pc += uint64(inst.Size)
	}

	vm.stats.RunTime = time.Since(vm.stats.StartTime)
	return nil
}

func (vm *VM3D) fetchInstruction(program []byte) (*Instruction3D, error) {
	remaining := uint64(len(program)) - vm.pc
	if remaining < 1 {
		return nil, fmt.Errorf("no more instructions")
	}

	// Determine instruction size based on opcode
	opcode := Opcode(program[vm.pc])
	var instSize uint32

	switch opcode {
	case OP_MLOAD3D, OP_MSTORE3D, OP_MCOPY3D:
		instSize = 20
	case OP_MALLOC3D:
		instSize = 14
	case OP_MFREE3D:
		instSize = 15
	case OP_MSIZE3D:
		instSize = 16
	case OP_MOV3D:
		instSize = 8
	case OP_ADD3D, OP_SUB3D:
		instSize = 4
	default:
		// Default small instruction (NOP, HALT3D, etc.)
		instSize = 1
	}

	if uint64(instSize) > remaining {
		return nil, fmt.Errorf("incomplete instruction")
	}

	data := program[vm.pc : vm.pc+uint64(instSize)]
	return DecodeInstruction(data)
}

func (vm *VM3D) executeInstruction(inst *Instruction3D) error {
	// Check gas for this instruction
	gasCost := vm.calculateGasCost(inst)
	if err := vm.gasAccount.Consume(gasCost); err != nil {
		return err
	}

	// Execute based on opcode
	switch inst.Opcode {
	case OP_MLOAD3D:
		return vm.executeMLoad3D(inst)
	case OP_MSTORE3D:
		return vm.executeMStore3D(inst)
	case OP_MALLOC3D:
		return vm.executeMAlloc3D(inst)
	case OP_MFREE3D:
		return vm.executeMFree3D(inst)
	case OP_MCOPY3D:
		return vm.executeMCopy3D(inst)
	case OP_MQUERY3D:
		return vm.executeMQuery3D(inst)
	case OP_MCOMPACT3D:
		return vm.executeMCompact3D(inst)
	case OP_NOP:
		return vm.executeNOP()
	case OP_ADD3D:
		return vm.executeAdd3D(inst)
	case OP_SUB3D:
		return vm.executeSub3D(inst)
	case OP_MOV3D:
		return vm.executeMov3D(inst)
	case OP_MSIZE3D:
		return vm.executeMSize3D(inst)
	case OP_HALT3D:
		return vm.executeHalt3D()
	default:
		return fmt.Errorf("unsupported opcode: 0x%x", inst.Opcode)
	}
}

func (vm *VM3D) executeHalt3D() error {
	// Signal halt to execution loop by setting the halted flag
	vm.halted = true
	return nil
}

func (vm *VM3D) executeMCompact3D(inst *Instruction3D) error {
	// Get compaction analysis
	config := allocator.DefaultCompactionConfig()
	allocsToMove := vm.allocator.AnalyzeCompaction(config)

	if len(allocsToMove) == 0 {
		// No compaction needed, no-op
		return nil
	}

	// Perform compaction - simplified: just calculate gas refund
	// In a full implementation, would actually relocate allocations
	compactedSize := uint64(0)
	for range allocsToMove {
		// Each allocation averages ~100 bytes on average
		compactedSize += 100
	}

	// Calculate gas refund based on consolidation
	// Assume 1 gas per byte compacted, apply refund percentage
	gasRefund := uint64(float64(compactedSize) * config.GasRefundPercentage)
	if gasRefund > 0 {
		vm.gasAccount.Refund(gasRefund)
	}

	// Update stats
	storedMem := atomic.LoadUint64(&vm.stats.MemoryUsage)
	if storedMem > compactedSize {
		atomic.AddUint64(&vm.stats.MemoryUsage, ^uint64(compactedSize-1)) // Decrease by compactedSize
	}

	return nil
}

func (vm *VM3D) calculateGasCost(inst *Instruction3D) uint64 {
	// Base cost per opcode
	baseCost := map[Opcode]uint64{
		OP_MLOAD3D:  10,
		OP_MSTORE3D: 15,
		OP_MALLOC3D: 20,
		OP_MFREE3D:  5,
		OP_MCOPY3D:  25,
		OP_MQUERY3D: 8,
		OP_MCOMPACT3D: 30,
		OP_NOP:      1,
		OP_ADD3D:    3,
		OP_SUB3D:    3,
		OP_MOV3D:    2,
		OP_MSIZE3D:  5,
		OP_HALT3D:   1,
	}

	cost := baseCost[inst.Opcode]

	// Add cost based on operands
	for _, operand := range inst.Operands {
		if operand.Type == OT_SIZE {
			if size, ok := operand.Value.(uint32); ok {
				cost += uint64(size) * 2 // 2 gas per byte
			}
		}
	}

	return cost
}

// GetMerkleProof generates a merkle proof for a given address
// Returns nil if merkle tree is not enabled or address doesn't exist
func (vm *VM3D) GetMerkleProof(addr types.Address3D) ([][]byte, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if vm.merkleTree == nil {
		return nil, fmt.Errorf("merkle tree not enabled")
	}

	return vm.merkleTree.GenerateProof(addr)
}

// VerifyMemoryState verifies that memory at an address matches the given data
// using the merkle tree proof system
func (vm *VM3D) VerifyMemoryState(addr types.Address3D, data []byte, proof [][]byte) (bool, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if vm.merkleTree == nil {
		return false, fmt.Errorf("merkle tree not enabled")
	}

	return vm.merkleTree.VerifyProof(addr, proof, data)
}

// GetMerkleRoot returns the current merkle root hash
func (vm *VM3D) GetMerkleRoot() []byte {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if vm.merkleTree == nil {
		return nil
	}

	return vm.merkleTree.GetRoot()
}

// GetStats returns a copy of current VM statistics
func (vm *VM3D) GetStats() VMStats {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.stats
}

// GetGasRemaining returns remaining gas
func (vm *VM3D) GetGasRemaining() int {
	return vm.gasAccount.Remaining()
}

// GetGasUsed returns total gas used
func (vm *VM3D) GetGasUsed() uint64 {
	return vm.gasAccount.Used()
}

// GetMemoryManager returns the memory manager for testing
func (vm *VM3D) GetMemoryManager() *MemoryManager {
	return vm.memory
}

// GetRegisters returns the registers for testing and operations
func (vm *VM3D) GetRegisters() *Registers {
	return vm.registers
}

// Load3D reads data from 3D memory address
func (vm *VM3D) Load3D(addr types.Address3D, size uint32) ([]byte, error) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.memory.Load3D(addr, size)
}

// Store3D writes data to 3D memory address and updates merkle tree
func (vm *VM3D) Store3D(addr types.Address3D, data []byte) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	// Store to memory
	if err := vm.memory.Store3D(addr, data); err != nil {
		// If address not allocated, attempt to allocate a block at that address
		if err.Error() == fmt.Sprintf("address not allocated: %v", addr) {
			// Allocate a new block sized to data in Heap region
			alloc, aerr := vm.allocator.Allocate(uint32(len(data)), types.RegionTypeHeap, 0)
			if aerr != nil {
				return aerr
			}
			// Track allocation
			vm.memory.trackAllocation(alloc)

			// If allocation address differs, adjust target to allocation address
			addr = alloc.Address

			// Try storing again
			if serr := vm.memory.Store3D(addr, data); serr != nil {
				return serr
			}
		} else {
			return err
		}
	}

	// Update cache
	vm.cache.Put(addr, data)

	// Update Merkle tree if enabled
	if vm.merkleTree != nil {
		vm.merkleTree.Update(addr, data)
	}

	// Update access statistics
	vm.memory.UpdateAccess(addr)
	if err := vm.gasAccount.Consume(uint64(15 * len(data))); err != nil {
		return err
	} // Th
	return nil
}

// NewMerkleTree creates a new merkle tree instance (helper for testing/direct use)
func NewMerkleTree() *merkle.Tree3D {
	return merkle.NewTree3D()
}

// GetCompactionReport returns fragmentation analysis for all regions
func (vm *VM3D) GetCompactionReport() map[uint64]types.CompactionReport {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	return vm.allocator.GetCompactionReport()
}

// AnalyzeCompaction determines which allocations should be moved for consolidation
func (vm *VM3D) AnalyzeCompaction(config allocator.CompactionConfig) []uint64 {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	return vm.allocator.AnalyzeCompaction(config)
}

// AllocateMemory allocates a new block and registers it with the memory manager
func (vm *VM3D) AllocateMemory(size uint32, regionType types.RegionType) (types.Address3D, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	alloc, err := vm.allocator.Allocate(size, regionType, 0)
	if err != nil {
		return types.Address3D{}, err
	}
	vm.memory.trackAllocation(alloc)
	return alloc.Address, nil
}

// ExecuteCompactInstruction executes a compaction instruction for testing
func (vm *VM3D) ExecuteCompactInstruction(inst *Instruction3D) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	return vm.executeMCompact3D(inst)
}
