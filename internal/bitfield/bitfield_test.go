package bitfield

import (
	"bytes"
	"math/bits"
	"testing"
)

func TestHasPiece(t *testing.T) {
	t.Parallel()
	// index 0: 00001000, reversed 00010000
	// index 1: 11010111, reversed 11101011
	bf := Bitfield{bits.Reverse8(8), bits.Reverse8(215)}

	tests := []struct {
		name  string
		input int
		want  bool
	}{
		{
			name:  "byte 8, bit 3",
			input: 3,
			want:  true,
		},
		{
			name:  "byte 8, bit 2",
			input: 2,
			want:  false,
		},
		{
			name:  "byte 8, bit 4",
			input: 4,
			want:  false,
		},
		{
			name:  "byte 215, block + offset, bit 0",
			input: 8,
			want:  true,
		},
		{
			name:  "byte 215, block + offset, bit 1",
			input: 9,
			want:  true,
		},
		{
			name:  "byte 215, block + offset, bit 2",
			input: 10,
			want:  true,
		},
		{
			name:  "byte 215, block + offset, bit 3",
			input: 11,
			want:  false,
		},
		{
			name:  "large index",
			input: 100,
			want:  false,
		},
		{
			name:  "negative index zero-block",
			input: -1,
			want:  false,
		},
		{
			name:  "negative index",
			input: -10,
			want:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := bf.HasPiece(tc.input)

			if got != tc.want {
				t.Fatal("test failed")
			}
		})
	}
}

func TestHasPieceZeros(t *testing.T) {
	t.Parallel()
	bf := Bitfield{0b00000000}
	want := false

	for i := range 8 {
		if bf.HasPiece(i) != want {
			t.Fatal("true in only-zeros-byte")
		}
	}
}

func TestHasPieceOnes(t *testing.T) {
	t.Parallel()
	bf := Bitfield{0b11111111}
	want := true

	for i := range 8 {
		if bf.HasPiece(i) != want {
			t.Fatal("false in only-ones-byte")
		}
	}
}

func TestSetPiece(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		bf    Bitfield
		index int
		want  Bitfield
	}{
		{
			name:  "simple",
			bf:    Bitfield{0b01010101, 0b11110000},
			index: 2, //				v
			want:  Bitfield{0b01110101, 0b11110000},
		},
		{
			name:  "already one",
			bf:    Bitfield{0b01010101, 0b11110000},
			index: 3, //				 v
			want:  Bitfield{0b01010101, 0b11110000},
		},
		{
			name:  "simple next block",
			bf:    Bitfield{0b01010101, 0b11110000},
			index: 12, //				 							v
			want:  Bitfield{0b01010101, 0b11111000},
		},
		{
			name:  "large index",
			bf:    Bitfield{0b01010101, 0b11110000},
			index: 100,
			want:  Bitfield{0b01010101, 0b11110000},
		},
		{
			name:  "negative index zero block",
			bf:    Bitfield{0b01010101, 0b11110000},
			index: -1,
			want:  Bitfield{0b01010101, 0b11110000},
		},
		{
			name:  "negative index non-zero block",
			bf:    Bitfield{0b01010101, 0b11110000},
			index: -10,
			want:  Bitfield{0b01010101, 0b11110000},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.bf.SetPiece(tc.index)

			if !bytes.Equal(tc.bf, tc.want) {
				t.Fatalf("incorrect want: %v, have: %v", tc.want, tc.bf)
			}
		})
	}
}

func TestSetPieceByte(t *testing.T) {
	t.Parallel()
	bf := Bitfield{0}

	fillByPower := byte(0)
	if bf[0] != fillByPower {
		t.Fatal("zero not equal zero")
	}

	for i := range 8 {
		bf.SetPiece(i)
		fillByPower |= 1 << (7 - i)

		if bf[0] != fillByPower {
			t.Fatalf("failed power 2 check, has: %08b, want: %08b", bf[0], fillByPower)
		}
	}
}
