package types

import "testing"

func TestBitmapSetClearFindContiguous(t *testing.T) {
	b := NewBitmap(64)

	if free := b.FreeBytes(); free != 64 {
		t.Fatalf("expected 64 free bytes, got %d", free)
	}

	// Find a contiguous block of 8 bytes
	pos, ok := b.FindContiguous(8)
	if !ok {
		t.Fatalf("expected to find contiguous block of size 8")
	}
	if pos != 0 {
		t.Fatalf("expected start pos 0, got %d", pos)
	}

	// Reserve the range
	b.SetRange(pos, pos+8)
	for i := uint16(0); i < 8; i++ {
		if !b.IsSet(pos + i) {
			t.Fatalf("bit %d should be set", pos+i)
		}
	}

	if free := b.FreeBytes(); free != 56 {
		t.Fatalf("expected 56 free bytes after set, got %d", free)
	}

	// Clear the range and ensure free bytes restored
	b.ClearRange(pos, pos+8)
	for i := uint16(0); i < 8; i++ {
		if b.IsSet(pos + i) {
			t.Fatalf("bit %d should be clear", pos+i)
		}
	}
	if free := b.FreeBytes(); free != 64 {
		t.Fatalf("expected 64 free bytes after clear, got %d", free)
	}
}

func TestBitmapFreeAndUsedBytes(t *testing.T) {
	b := NewBitmap(100)
	// Set 10 bits scattered
	for i := uint16(0); i < 10; i++ {
		b.Set(i * 3)
	}

	used := b.UsedBytes()
	if used != 10 {
		t.Fatalf("expected 10 used bytes, got %d", used)
	}
	if free := b.FreeBytes(); free != 90 {
		t.Fatalf("expected 90 free bytes, got %d", free)
	}

	// Clear one and check counts
	b.Clear(0)
	if b.IsSet(0) {
		t.Fatalf("bit 0 should be cleared")
	}
	if used := b.UsedBytes(); used != 9 {
		t.Fatalf("expected 9 used bytes after clear, got %d", used)
	}
}

func TestFindContiguousEdgeCases(t *testing.T) {
	b := NewBitmap(10)

	// size 0 should fail
	if _, ok := b.FindContiguous(0); ok {
		t.Fatalf("FindContiguous with size 0 should fail")
	}

	// Fill most of the bitmap (positions 0..7 occupied)
	b.SetRange(0, 8)
	// There should be a contiguous block of size 2 at position 8
	pos, ok := b.FindContiguous(2)
	if !ok {
		t.Fatalf("expected to find contiguous block of size 2")
	}
	if pos != 8 {
		t.Fatalf("expected position 8 for block of size 2, got %d", pos)
	}

	// Requesting a block larger than remaining free space should fail
	if _, ok := b.FindContiguous(3); ok {
		t.Fatalf("expected no contiguous block of size 3")
	}
}
