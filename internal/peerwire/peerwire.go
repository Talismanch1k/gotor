package peerwire

import (
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Talismanch1k/gotor/internal/client"
	"github.com/Talismanch1k/gotor/internal/message"
	"github.com/Talismanch1k/gotor/internal/peer"
)

// Process:
// -> send interested
// <- get unchoke
// -> send msgRequest, to get pieces=x begin=y length=MaxBlockSize
// <- get msgPiece with data...
// Verifying sha1
// -> send Have with index=x

const (
	// TODO: increase when done with code
	MaxPeerConnections   = 30           // max connection with peers, 1 peer = 1 connection
	MaxRequests          = 5            // max number of requests sent to a single peer at any given time
	MaxBlockSize         = 16 * 1 << 10 // default block size from bep 3
	MaxFailedConnections = 5
)

const (
	pieceTimeout = 30 * time.Second
)

type Torrent struct {
	name string

	peers  []peer.Peer
	peerID [20]byte

	infoHash    [20]byte
	pieceHashes [][20]byte
	pieceLength int
	length      int
}

type pieceWork struct {
	hash  [20]byte
	index int
}

type pieceResult struct {
	buf   []byte
	index int
}

type pieceProcess struct {
	pieceWork       *pieceWork
	client          *client.Client
	buf             []byte
	downloaded      int
	requested       int
	currentRequests int
}

// Pick peer
// make client
// unchoke
// send requests
// download piece
// verify piece
// again
// otherwise close client and change peer

func (p *pieceProcess) readMessage() error {
	msg, err := p.client.ReadMessage()
	if err != nil {
		return fmt.Errorf("read message: %w", err)
	}

	// keep alive
	if msg == nil {
		return nil
	}

	switch msg.ID {
	case message.MsgUnchoke:
		p.client.Chocked = false

	case message.MsgChoke:
		p.client.Chocked = true

	case message.MsgHave:
		index, err := message.ReadHave(msg)
		if err != nil {
			return fmt.Errorf("read have message: %w", err)
		}
		p.client.Bitfield.SetPiece(index)

	case message.MsgPiece:
		n, err := message.ReadPiece(p.pieceWork.index, p.buf, msg)
		if err != nil {
			return fmt.Errorf("read piece message: %w", err)
		}
		p.downloaded += n
		p.currentRequests--
	}

	return nil
}

// one goroutine = one piece, because channel laws
func (t *Torrent) downloadPiece(state *pieceProcess) (*pieceResult, error) {
	err := state.client.Conn.SetDeadline(time.Now().Add(pieceTimeout))
	if err != nil {
		return nil, fmt.Errorf("set deadline for peer connection: %w", err)
	}

	defer func() {
		if derr := state.client.Conn.SetDeadline(time.Time{}); derr != nil {
			err = fmt.Errorf("set deadline for peer connection: %w", derr)
		}
	}()

	pieceSize := t.pieceSize(state.pieceWork.index)

	for state.downloaded < pieceSize {
		if !state.client.Chocked {
			for state.currentRequests < MaxRequests && state.requested < pieceSize {
				blockSize := min(MaxBlockSize, pieceSize-state.requested)

				if err := state.client.SendRequest(state.pieceWork.index, state.requested, blockSize); err != nil {
					return nil, fmt.Errorf("request piece: %w", err)
				}

				state.currentRequests++
				state.requested += blockSize
			}
		}

		if err := state.readMessage(); err != nil {
			return nil, fmt.Errorf("read message: %w", err)
		}

	}

	return &pieceResult{
		index: state.pieceWork.index,
		buf:   state.buf,
	}, nil
}

func (t *Torrent) processPeer(ctx context.Context, peer peer.Peer, jobs chan *pieceWork, res chan<- *pieceResult) error {
	c, err := client.New(peer, t.peerID, t.infoHash)
	if err != nil {
		return fmt.Errorf("make client for peer: %w", err)
	}
	defer func() { _ = c.Conn.Close() }()

	var piece *pieceWork
	peerRetries := 0

	for peerRetries <= MaxFailedConnections {
		// take new piece
		select {
		case <-ctx.Done():
			return ctx.Err()
		case piece = <-jobs: // must me empty on cancelled context, so locking here
		}

		// check with bitfield
		if !c.Bitfield.HasPiece(piece.index) {
			peerRetries++
			jobs <- piece
			continue
		}

		// if peer has our piece, send interested
		err = c.SendInterested()
		if err != nil {
			jobs <- piece
			return fmt.Errorf("process with peer: %w", err)
		}

		state := pieceProcess{
			pieceWork: piece,
			client:    c,
			buf:       make([]byte, t.pieceSize(piece.index)),
		}

		// start downloading
		result, err := t.downloadPiece(&state)
		if err != nil {
			jobs <- piece
			return fmt.Errorf("download with peer: %w", err)
		}

		if !result.matchesHash(piece) {
			peerRetries++
			slog.Warn("piece failed hash check", "index", result.index)
			jobs <- piece
			continue
		}

		res <- result
	}

	return nil
}

// WARN: SIMPLE IMPLEMENTATION, NEEDS UPGRADE
func (t *Torrent) downloadWorker(
	ctx context.Context,
	peers chan peer.Peer,
	jobs chan *pieceWork,
	res chan<- *pieceResult,
) {
	for {
		var peer peer.Peer

		select {
		case <-ctx.Done():
			return
		case peer = <-peers:
		}

		err := t.processPeer(ctx, peer, jobs, res)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			slog.Warn("processing peer", "err", err)
		}
		peers <- peer // put peer back
	}
}

// matchesHash check if res.buf has equal hash as want.hash
func (res *pieceResult) matchesHash(want *pieceWork) bool {
	return sha1.Sum(res.buf) == want.hash
}

func (t *Torrent) Download(ctx context.Context) ([]byte, error) {
	if len(t.peers) == 0 {
		return nil, errors.New("zero peers available, cannot download torrent")
	}

	peers := make(chan peer.Peer, len(t.peers))
	jobs := make(chan *pieceWork, len(t.pieceHashes))
	res := make(chan *pieceResult)

	// load peers
	for _, p := range t.peers {
		peers <- p
	}

	// load jobs
	for j := range len(t.pieceHashes) {
		jobs <- &pieceWork{
			index: j,
			hash:  t.pieceHashes[j],
		}
	}

	wg := &sync.WaitGroup{}

	// context for cancelling goroutines, after finish downloading
	downloadCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	workers := min(MaxPeerConnections, len(t.peers))

	// start MaxPeerConnections workers
	for range workers {
		wg.Go(func() { t.downloadWorker(downloadCtx, peers, jobs, res) })
	}

	go func() {
		wg.Wait()
		close(peers)
		close(jobs)
		close(res)
	}()

	downloaded := 0
	data := make([]byte, t.length)
	for piece := range res {
		if piece.index < 0 || piece.index >= len(t.pieceHashes) {
			return nil, fmt.Errorf("got unexpected index: %d", piece.index)
		}

		offset := piece.index * t.pieceLength
		copy(data[offset:], piece.buf)

		downloaded++
		if downloaded == len(t.pieceHashes) {
			cancel()
		}
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("outer context was cancelled: %w", err)
	}

	return data, nil
}

func (t *Torrent) pieceSize(index int) int {
	begin := index * t.pieceLength
	end := min(begin+t.pieceLength, t.length)

	return end - begin
}
