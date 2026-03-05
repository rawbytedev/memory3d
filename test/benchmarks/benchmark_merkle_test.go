package benchmarks

import (
	"testing"

	"github.com/rawbytedev/memory3d/internal/merkle"
	"github.com/rawbytedev/memory3d/internal/types"
)

// BenchmarkMerkleUpdate benchmarks merkle tree update operations
func BenchmarkMerkleUpdate(b *testing.B) {
	tree := merkle.NewTree3D()
	addr := types.Address3D{X: 0, Y: 0, Z: 0}
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.Update(addr, data)
	}
}

// BenchmarkMerkleGetRoot benchmarks merkle root retrieval
func BenchmarkMerkleGetRoot(b *testing.B) {
	tree := merkle.NewTree3D()
	addr := types.Address3D{X: 0, Y: 0, Z: 0}
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	tree.Update(addr, data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.GetRoot()
	}
}

// BenchmarkMerkleGenerateProof benchmarks merkle proof generation
func BenchmarkMerkleGenerateProof(b *testing.B) {
	tree := merkle.NewTree3D()
	addr := types.Address3D{X: 0, Y: 0, Z: 0}
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	tree.Update(addr, data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.GenerateProof(addr)
	}
}

// BenchmarkMerkleVerifyProof benchmarks merkle proof verification
func BenchmarkMerkleVerifyProof(b *testing.B) {
	tree := merkle.NewTree3D()
	addr := types.Address3D{X: 0, Y: 0, Z: 0}
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	tree.Update(addr, data)
	proof, _ := tree.GenerateProof(addr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tree.VerifyProof(addr, proof, data)
	}
}

// BenchmarkMerkleMultipleUpdates benchmarks multiple updates to different addresses
func BenchmarkMerkleMultipleUpdates(b *testing.B) {
	tree := merkle.NewTree3D()
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			addr := types.Address3D{X: uint64(j), Y: 0, Z: 0}
			tree.Update(addr, data)
		}
	}
}

// BenchmarkMerkleSequentialProofs benchmarks sequential proof generation
func BenchmarkMerkleSequentialProofs(b *testing.B) {
	tree := merkle.NewTree3D()
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	// Pre-populate with addresses
	for j := 0; j < 10; j++ {
		addr := types.Address3D{X: uint64(j), Y: 0, Z: 0}
		tree.Update(addr, data)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			addr := types.Address3D{X: uint64(j), Y: 0, Z: 0}
			tree.GenerateProof(addr)
		}
	}
}
