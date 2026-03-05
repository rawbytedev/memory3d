package benchmarks

import (
	"testing"

	"github.com/rawbytedev/memory3d/internal/allocator"
	"github.com/rawbytedev/memory3d/internal/types"
)

// BenchmarkAllocateSmall benchmarks allocation of small blocks (1KB)
func BenchmarkAllocateSmall(b *testing.B) {
	config := allocator.Config{
		PlaneSize:        types.PlaneSize,
		MaxYPlanes:       1024,
		MaxYPromotions:   8,
		WorkerCount:      4,
		EnableConcurrent: false,
		EnablePooling:    true,
	}
	alloc := allocator.NewAllocator3D(config)
	defer alloc.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		alloc.Allocate(1024, types.RegionTypeHeap, 0)
	}
}

// BenchmarkAllocateMedium benchmarks allocation of medium blocks (64KB)
func BenchmarkAllocateMedium(b *testing.B) {
	config := allocator.Config{
		PlaneSize:        types.PlaneSize,
		MaxYPlanes:       1024,
		MaxYPromotions:   8,
		WorkerCount:      4,
		EnableConcurrent: false,
		EnablePooling:    true,
	}
	alloc := allocator.NewAllocator3D(config)
	defer alloc.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		alloc.Allocate(65536, types.RegionTypeHeap, 0)
	}
}

// BenchmarkAllocateLarge benchmarks allocation of large blocks (1MB)
func BenchmarkAllocateLarge(b *testing.B) {
	config := allocator.Config{
		PlaneSize:        types.PlaneSize,
		MaxYPlanes:       1024,
		MaxYPromotions:   8,
		WorkerCount:      4,
		EnableConcurrent: false,
		EnablePooling:    true,
	}
	alloc := allocator.NewAllocator3D(config)
	defer alloc.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		alloc.Allocate(1048576, types.RegionTypeHeap, 0)
	}
}

// BenchmarkAllocateMultiple benchmarks sequential allocations
func BenchmarkAllocateMultiple(b *testing.B) {
	config := allocator.Config{
		PlaneSize:        types.PlaneSize,
		MaxYPlanes:       1024,
		MaxYPromotions:   8,
		WorkerCount:      4,
		EnableConcurrent: false,
		EnablePooling:    true,
	}
	alloc := allocator.NewAllocator3D(config)
	defer alloc.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			alloc.Allocate(10240, types.RegionTypeHeap, 0)
		}
	}
}

// BenchmarkGetCompactionReport benchmarks compaction report generation
func BenchmarkGetCompactionReport(b *testing.B) {
	config := allocator.Config{
		PlaneSize:        types.PlaneSize,
		MaxYPlanes:       1024,
		MaxYPromotions:   8,
		WorkerCount:      4,
		EnableConcurrent: false,
		EnablePooling:    true,
	}
	alloc := allocator.NewAllocator3D(config)
	defer alloc.Shutdown()

	// Pre-allocate some blocks
	for i := 0; i < 100; i++ {
		alloc.Allocate(10240, types.RegionTypeHeap, 0)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		alloc.GetCompactionReport()
	}
}

// BenchmarkAnalyzeCompaction benchmarks compaction analysis
func BenchmarkAnalyzeCompaction(b *testing.B) {
	config := allocator.Config{
		PlaneSize:        types.PlaneSize,
		MaxYPlanes:       1024,
		MaxYPromotions:   8,
		WorkerCount:      4,
		EnableConcurrent: false,
		EnablePooling:    true,
	}
	alloc := allocator.NewAllocator3D(config)
	defer alloc.Shutdown()

	// Pre-allocate some blocks
	for i := 0; i < 100; i++ {
		alloc.Allocate(10240, types.RegionTypeHeap, 0)
	}

	compactConfig := allocator.DefaultCompactionConfig()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		alloc.AnalyzeCompaction(compactConfig)
	}
}
