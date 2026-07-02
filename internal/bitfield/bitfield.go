// Package bitfield implements struct bitfield
// for message type "Bitfield"
package bitfield

type Bitfield []byte

// HasPiece return if bitfield has piece (i.e. bit equals 1)
// under idx in byte array or no
func (bf Bitfield) HasPiece(idx int) bool {
	if idx < 0 { // because in block -1 / 8 = 0
		return false
	}

	block := idx / 8
	if block >= len(bf) {
		return false
	}

	offset := idx % 8

	return bf[block]>>(7-offset)&1 != 0
}

// SetPiece sets bit under idx to one
func (bf Bitfield) SetPiece(idx int) {
	if idx < 0 { // because in block -1 / 8 = 0
		return
	}

	block := idx / 8
	if block >= len(bf) {
		return
	}

	offset := idx % 8
	bf[block] |= 1 << (7 - offset)
}
