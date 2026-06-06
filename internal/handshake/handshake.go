package handshake

import (
	"fmt"
	"io"
)

// BEP 3 SPECIFICATION
const (
	pstrLen      = 19
	pstr         = "BitTorrent protocol" // assume const pstr, maybe changed in future
	reservedLen  = 8
	infoHashLen  = 20
	peerIDLen    = 20
	handshakeLen = 1 + pstrLen + reservedLen + infoHashLen + peerIDLen
)

type Handshake struct {
	InfoHash [infoHashLen]byte
	PeerID   [peerIDLen]byte
}

func New(infoHash, peerID [peerIDLen]byte) *Handshake {
	return &Handshake{
		InfoHash: infoHash,
		PeerID:   peerID,
	}
}

func (h *Handshake) Serialize() []byte {
	buf := make([]byte, handshakeLen)

	buf[0] = byte(pstrLen) // Pstr len
	cur := 1

	cur += copy(buf[cur:], pstr)          // Pstr
	cur += reservedLen                    // reserved bytes
	cur += copy(buf[cur:], h.InfoHash[:]) // InfoHash
	copy(buf[cur:], h.PeerID[:])          // PeerID

	return buf
}

func Read(r io.Reader) (*Handshake, error) {
	var lengthBuf [1]byte

	if _, err := io.ReadFull(r, lengthBuf[:]); err != nil {
		return nil, fmt.Errorf("read pstr length during handshake: %w", err)
	}

	// verify pstr length
	pstrLenHandshake := int(lengthBuf[0])
	if pstrLenHandshake != pstrLen {
		return nil, fmt.Errorf("incorrect pstr length: %d", pstrLenHandshake)
	}

	// reading the remaining data
	var handshakeBuf [handshakeLen - 1]byte
	if _, err := io.ReadFull(r, handshakeBuf[:]); err != nil {
		return nil, fmt.Errorf("read handshake: %w", err)
	}

	// verify pstr
	pstrHandshake := string(handshakeBuf[:pstrLen])
	if pstrHandshake != pstr {
		return nil, fmt.Errorf("got incorrect handshake: %s", pstrHandshake)
	}

	h := Handshake{}
	// skip pstr and reserved bits
	infoHashStart := pstrLen + reservedLen
	// place after infoHash
	infoHashEnd := infoHashStart + infoHashLen

	copy(h.InfoHash[:], handshakeBuf[infoHashStart:infoHashEnd])
	copy(h.PeerID[:], handshakeBuf[infoHashEnd:])

	return &h, nil
}
