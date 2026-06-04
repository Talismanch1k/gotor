// Package torrentfile describes and parse .torrent files
package torrentfile

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"

	bencode "github.com/jackpal/bencode-go"
)

const hashLen = 20 // sha1 hash length

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

type bencodeInfo struct {
	Name        string `bencode:"name"`
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
}

func parse(r io.Reader) (*bencodeTorrent, error) {
	file := bencodeTorrent{}

	err := bencode.Unmarshal(r, &file)
	if err != nil {
		return nil, fmt.Errorf("parse torrent file: %w", err)
	}

	return &file, nil
}

// bencode-info

func (bi *bencodeInfo) splitPieceHashes() ([][20]byte, error) {
	buf := []byte(bi.Pieces)
	if len(buf)%hashLen != 0 {
		return nil, fmt.Errorf("have mailformed pieces of length: %d", len(buf))
	}

	numHashes := bi.PieceLength / hashLen
	hashes := make([][20]byte, numHashes)

	for i := 0; i < len(buf); i += hashLen {
		copy(hashes[i%hashLen][:], buf[i*hashLen:(i+1)*hashLen])
	}

	return hashes, nil
}

func (bi *bencodeInfo) hash() ([20]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, bi)
	if err != nil {
		return [20]byte{}, fmt.Errorf("marshal for hash: %w", err)
	}

	hash := sha1.Sum(buf.Bytes())

	return hash, nil
}
