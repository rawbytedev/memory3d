package merkle

import (
	"crypto/sha256"
	"sync"

	"github.com/rawbytedev/memory3d/internal/types"
)

// TreeHasher interface allows different hash implementations to be plugged in
// This abstraction enables future integration of different merkle tree variants
type TreeHasher interface {
	// Hash computes hash of data
	Hash(data []byte) []byte

	// CombineHashes combines two hashes into one
	CombineHashes(left, right []byte) []byte
}

// SHA256Hasher implements TreeHasher using SHA256
type SHA256Hasher struct{}

func (h *SHA256Hasher) Hash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

func (h *SHA256Hasher) CombineHashes(left, right []byte) []byte {
	combined := make([]byte, len(left)+len(right))
	copy(combined, left)
	copy(combined[len(left):], right)
	return h.Hash(combined)
}

// Tree3D represents a merkle tree for 3D memory space
// Provides proof generation and verification for memory contents
type Tree3D struct {
	// Root hash of the current tree state
	root []byte

	// Merkle tree node cache (address -> hash)
	nodeCache map[types.Address3D][]byte

	// Hasher implementation (can be swapped)
	hasher TreeHasher

	// Concurrency control
	mu sync.RWMutex

	// Statistics
	updates uint64
}

// NewTree3D creates a new merkle tree with default SHA256 hasher
func NewTree3D() *Tree3D {
	return &Tree3D{
		nodeCache: make(map[types.Address3D][]byte),
		hasher:    &SHA256Hasher{},
		root:      make([]byte, 32), // Empty root initially
	}
}

// NewTree3DCustom creates a merkle tree with a custom hasher
// This allows integration of different hash implementations
func NewTree3DCustom(hasher TreeHasher) *Tree3D {
	return &Tree3D{
		nodeCache: make(map[types.Address3D][]byte),
		hasher:    hasher,
		root:      make([]byte, 32),
	}
}

// Update updates the merkle tree with new data at a 3D address
// This is called whenever memory is written to
func (t *Tree3D) Update(addr types.Address3D, data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Hash the data
	nodeHash := t.hasher.Hash(data)

	// Store in cache
	t.nodeCache[addr] = nodeHash

	// Recompute root (simplified approach)
	// TODO: Optimize root computation for large trees
	t.recomputeRoot()

	t.updates++
	return nil
}

// GetRoot returns the current merkle root
func (t *Tree3D) GetRoot() []byte {
	t.mu.RLock()
	defer t.mu.RUnlock()

	root := make([]byte, len(t.root))
	copy(root, t.root)
	return root
}

// GenerateProof generates a merkle proof for an address
// Currently returns a simplified proof; can be optimized for full tree structure
func (t *Tree3D) GenerateProof(addr types.Address3D) ([][]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// TODO: Implement full merkle proof generation
	// For now, return the hash for the address
	if hash, exists := t.nodeCache[addr]; exists {
		return [][]byte{hash}, nil
	}

	// Return empty proof for non-existent addresses
	return [][]byte{}, nil
}

// VerifyProof verifies a merkle proof
// This can be used to prove memory contents without revealing the full tree
func (t *Tree3D) VerifyProof(addr types.Address3D, proof [][]byte, data []byte) (bool, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// TODO: Implement full proof verification
	// For now, verify simple hash match
	if len(proof) == 0 {
		return false, nil
	}

	expectedHash := t.hasher.Hash(data)
	actualHash := proof[0]

	// Compare hashes (constant-time comparison recommended)
	if len(expectedHash) != len(actualHash) {
		return false, nil
	}

	for i := range expectedHash {
		if expectedHash[i] != actualHash[i] {
			return false, nil
		}
	}

	return true, nil
}

// GetStatistics returns tree statistics
func (t *Tree3D) GetStatistics() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return map[string]interface{}{
		"updates":      t.updates,
		"cached_nodes": len(t.nodeCache),
		"root_hash":    t.root,
	}
}

// recomputeRoot recomputes merkle root from current cache
// This is a simplified implementation that XORs all node hashes
// TODO: Implement proper tree structure for production
func (t *Tree3D) recomputeRoot() {
	if len(t.nodeCache) == 0 {
		t.root = make([]byte, 32)
		return
	}

	// Start with zero hash
	root := make([]byte, 32)

	// Combine all hashes (simplified approach)
	for _, hash := range t.nodeCache {
		root = t.hasher.CombineHashes(root, hash)
	}

	t.root = root
}

// Clear clears all cached nodes and resets root
func (t *Tree3D) Clear() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.nodeCache = make(map[types.Address3D][]byte)
	t.root = make([]byte, 32)
	t.updates = 0

	return nil
}
