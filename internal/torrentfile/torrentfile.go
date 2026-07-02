// Package torrentfile describes and parse .torrent files
package torrentfile

import (
	"fmt"
	"io"
	"net/url"
	"strconv"

	"github.com/Talismanch1k/gotor/internal/bencode"
)

type TorrentFile struct {
	// metainfo
	Announce string // URL of the tracker

	// metainfo dict
	Name        string
	PieceHashes [][20]byte // sha1 hash of each file
	PieceLength int        // number of bytes each files is split into
	Length      int        // Length of the file in bytes

	// hash of metainfo dict
	InfoHash [20]byte // sha1 hash of metainfo (bencodeInfo struct)
}

func (t *TorrentFile) Read(r io.Reader) (*TorrentFile, error) {
	ben, err := bencode.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parse bencode from torrent file: %w", err)
	}

	infoHash, err := ben.Info.Hash()
	if err != nil {
		return nil, fmt.Errorf("get info hash: %w", err)
	}

	pieceHashes, err := ben.Info.SplitPieceHashes()
	if err != nil {
		return nil, fmt.Errorf("split piece hashes: %w", err)
	}

	torrentfile := TorrentFile{
		Name:        ben.Info.Name,
		Announce:    ben.Announce,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: ben.Info.PieceLength,
		Length:      ben.Info.Length,
	}

	return &torrentfile, nil
}

func (t *TorrentFile) buildTrackerURL(peerID [20]byte, port uint16) (string, error) {
	base, err := url.Parse(t.Announce)
	if err != nil {
		return "", fmt.Errorf("parse torrent announce for URL: %w", err)
	}

	params := url.Values{
		"info_hash":  []string{string(t.InfoHash[:])},
		"peer_id":    []string{string(peerID[:])},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(t.Length)},
	}

	base.RawQuery = params.Encode()

	return base.String(), nil
}
