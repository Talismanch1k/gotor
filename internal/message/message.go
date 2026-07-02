package message

import (
	"encoding/binary"
	"fmt"
	"io"
)

const lenSize = 4 // bytes

type ID uint8

const (
	MsgChoke         ID = iota // MsgChoke chokes the receiver
	MsgUnchoke                 // MsgUnchoke unchokes the receiver
	MsgInterested              // MsgInterested expresses interest in receiving data
	MsgNotInterested           // MsgNotInterested expresses disinterest in receiving data
	MsgHave                    // MsgHave alerts the receiver that the sender has downloaded a piece
	MsgBitfield                // MsgBitfield encodes which pieces that the sender has downloaded
	MsgRequest                 // MsgRequest requests a block of data from the receiver
	MsgPiece                   // MsgPiece delivers a block of data to fulfill a request
	MsgCancel                  // MsgCancel cancels a request
)

var keepAlive = []byte{0, 0, 0, 0}

type Message struct {
	ID   ID
	Data []byte
}

func (m *Message) Serialize() []byte {
	if m == nil {
		return keepAlive
	}

	msgLen := uint32(1 + len(m.Data))
	buf := make([]byte, lenSize+msgLen)
	binary.BigEndian.PutUint32(buf, msgLen) // message len
	buf[lenSize] = byte(m.ID)               // id
	copy(buf[lenSize+1:], m.Data)           // message data

	return buf
}

func Read(r io.Reader) (*Message, error) {
	var lenBuf [lenSize]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, fmt.Errorf("read message length: %w", err)
	}

	length := binary.BigEndian.Uint32(lenBuf[:])
	if length == 0 {
		return nil, nil
	}

	var msgID [1]byte
	if _, err := io.ReadFull(r, msgID[:]); err != nil {
		return nil, fmt.Errorf("read message type: %w", err)
	}

	data := make([]byte, length-1)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("read message type: %w", err)
	}

	return &Message{
		ID:   ID(msgID[0]),
		Data: data,
	}, nil
}

func MakeRequest(index, begin, length uint32) *Message {
	data := make([]byte, 12)

	binary.BigEndian.PutUint32(data[0:4], index)
	binary.BigEndian.PutUint32(data[4:8], begin)
	binary.BigEndian.PutUint32(data[8:12], length)

	return &Message{
		ID:   MsgRequest,
		Data: data,
	}
}

func ReadPiece(index int, buf []byte, msg *Message) (int, error) {
	if msg.ID != MsgPiece {
		return 0, fmt.Errorf("expected message type %s, but got %s", MsgPiece, msg.ID)
	}

	if len(msg.Data) < 8 {
		return 0, fmt.Errorf("data is too short for %s type message (%d)", MsgPiece, len(msg.Data))
	}

	msgIndex := int(binary.BigEndian.Uint32(msg.Data[0:4]))
	if index != msgIndex {
		return 0, fmt.Errorf("expected index %d, but got %d", index, msgIndex)
	}

	begin := int(binary.BigEndian.Uint32(msg.Data[4:8]))
	if begin >= len(buf) {
		return 0, fmt.Errorf("begin offset too large, buffer len=%d, got begin=%d", len(buf), begin)
	}

	data := msg.Data[8:]
	if begin+len(data) > len(buf) {
		return 0, fmt.Errorf("too long data %d, with offset %d for buffer len=%d", len(data), begin, len(buf))
	}

	copied := copy(buf[begin:], data)

	return copied, nil
}

func ReadHave(msg *Message) (int, error) {
	if msg.ID != MsgHave {
		return 0, fmt.Errorf("expected message type %s, but got %s", MsgHave, msg.ID)
	}

	if len(msg.Data) < 4 {
		return 0, fmt.Errorf("data is too short for %s type message (%d)", MsgHave, len(msg.Data))
	}

	index := int(binary.BigEndian.Uint32(msg.Data))
	return index, nil
}

func MakeHave(index uint32) *Message {
	data := make([]byte, 4)

	binary.BigEndian.PutUint32(data[0:4], index)

	return &Message{
		ID:   MsgHave,
		Data: data,
	}
}

func (id ID) String() string {
	switch id {
	case MsgChoke:
		return "choke"
	case MsgUnchoke:
		return "unchoke"
	case MsgInterested:
		return "interested"
	case MsgNotInterested:
		return "not interested"
	case MsgHave:
		return "have"
	case MsgBitfield:
		return "bitfield"
	case MsgRequest:
		return "request"
	case MsgPiece:
		return "piece"
	case MsgCancel:
		return "cancel"
	default:
		return "unknown message type"
	}
}
