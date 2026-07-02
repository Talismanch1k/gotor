package client

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/Talismanch1k/gotor/internal/bitfield"
	"github.com/Talismanch1k/gotor/internal/handshake"
	"github.com/Talismanch1k/gotor/internal/message"
	"github.com/Talismanch1k/gotor/internal/peer"
)

const (
	connTimeout      = 3 * time.Second
	handshakeTimeout = 3 * time.Second
	bitfieldTimeout  = 5 * time.Second
)

type Client struct {
	Conn     net.Conn
	Chocked  bool
	Bitfield bitfield.Bitfield
	peer     peer.Peer
	infoHash [20]byte
	peerID   [20]byte
}

// New creates a new TCP connection with peer
// and proceeds handshake and read bitfield message
func New(peer peer.Peer, peerID, infoHash [20]byte) (*Client, error) {
	conn, err := net.DialTimeout("tcp", peer.Address(), connTimeout)
	if err != nil {
		return nil, fmt.Errorf("establish connection: %w", err)
	}

	_, err = Handshake(conn, peerID, infoHash)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("do handshake: %w", err)
	}

	bf, err := Bitfield(conn)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("get bitfield: %w", err)
	}

	return &Client{
		Conn:     conn,
		Chocked:  true,
		Bitfield: bf,
		peer:     peer,
		infoHash: infoHash,
		peerID:   peerID,
	}, nil
}

func Handshake(conn net.Conn, peerID, infoHash [20]byte) (*handshake.Handshake, error) {
	err := conn.SetDeadline(time.Now().Add(handshakeTimeout))
	if err != nil {
		return nil, fmt.Errorf("set deadline for handshake request: %w", err)
	}
	defer func() {
		if derr := conn.SetDeadline(time.Time{}); derr != nil {
			err = fmt.Errorf("clear deadline for handshake request: %w", derr)
		}
	}()

	req := handshake.New(infoHash, peerID)
	_, err = conn.Write(req.Serialize())
	if err != nil {
		return nil, fmt.Errorf("write handshake request: %w", err)
	}

	res, err := handshake.Read(conn)
	if err != nil {
		return nil, fmt.Errorf("read handshake response: %w", err)
	}

	if !bytes.Equal(req.InfoHash[:], res.InfoHash[:]) {
		return nil, fmt.Errorf("expect infohash %x but got %x in response", req.InfoHash, res.InfoHash)
	}

	return res, nil
}

func Bitfield(conn net.Conn) (bitfield.Bitfield, error) {
	err := conn.SetDeadline(time.Now().Add(bitfieldTimeout))
	if err != nil {
		return nil, fmt.Errorf("set deadline for bitfield request: %w", err)
	}
	defer func() {
		if derr := conn.SetDeadline(time.Time{}); derr != nil {
			err = fmt.Errorf("clear deadline for handshake request: %w", err)
		}
	}()

	msg, err := message.Read(conn)
	if err != nil {
		return nil, fmt.Errorf("receive bitfield message: %w", err)
	}

	if msg == nil {
		return nil, errors.New("expect bitfield, got nil")
	}

	if msg.ID != message.MsgBitfield {
		return nil, fmt.Errorf("expect bitfield, got message id: %d", msg.ID)
	}

	return msg.Data, nil
}

func (c *Client) ReadMessage() (*message.Message, error) {
	msg, err := message.Read(c.Conn)
	if err != nil {
		return nil, fmt.Errorf("read message: %w", err)
	}

	return msg, nil
}

func (c *Client) SendRequest(index, begin, length int) error {
	if index < 0 || begin < 0 || length < 0 {
		return fmt.Errorf("some arguments < 0, index=%d, begin=%d, length=%d", index, begin, length)
	}

	msg := message.MakeRequest(uint32(index), uint32(begin), uint32(length))
	if _, err := c.Conn.Write(msg.Serialize()); err != nil {
		return fmt.Errorf("send request message: %w", err)
	}

	return nil
}

func (c *Client) SendHave(index int) error {
	if index < 0 {
		return fmt.Errorf("index cannot be negative, index=%d", index)
	}

	msg := message.MakeHave(uint32(index))
	if _, err := c.Conn.Write(msg.Serialize()); err != nil {
		return fmt.Errorf("send have message: %w", err)
	}

	return nil
}

func (c *Client) SendInterested() error {
	msg := message.Message{ID: message.MsgInterested}
	if _, err := c.Conn.Write(msg.Serialize()); err != nil {
		return fmt.Errorf("send interested message: %w", err)
	}

	return nil
}
