package benchmarks

import (
	"testing"

	"github.com/rawbytedev/memory3d/internal/vm"
)

// BenchmarkGasConsume benchmarks gas consumption
func BenchmarkGasConsume(b *testing.B) {
	gasAcct := vm.NewAccountant(10000000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gasAcct.Consume(100)
	}
}

// BenchmarkGasRefund benchmarks gas refund
func BenchmarkGasRefund(b *testing.B) {
	gasAcct := vm.NewAccountant(10000000)
	gasAcct.Consume(5000000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gasAcct.Refund(1000)
	}
}

// BenchmarkGasRemaining benchmarks gas remaining lookup
func BenchmarkGasRemaining(b *testing.B) {
	gasAcct := vm.NewAccountant(10000000)
	gasAcct.Consume(5000000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gasAcct.Remaining()
	}
}

// BenchmarkGasUsed benchmarks gas used lookup
func BenchmarkGasUsed(b *testing.B) {
	gasAcct := vm.NewAccountant(10000000)
	gasAcct.Consume(5000000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gasAcct.Used()
	}
}

// BenchmarkGasConsumeRefundCycle benchmarks consume + refund cycles
func BenchmarkGasConsumeRefundCycle(b *testing.B) {
	gasAcct := vm.NewAccountant(10000000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gasAcct.Consume(1000)
		gasAcct.Refund(500)
	}
}
