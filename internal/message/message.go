package message

import (
	"encoding/binary"
	"io"
)

type messageID uint8

const lenSize = 4 // bytes

const (
	MsgChoke         messageID = 0 // MsgChoke chokes the receiver
	MsgUnchoke       messageID = 1 // MsgUnchoke unchokes the receiver
	MsgInterested    messageID = 2 // MsgInterested expresses interest in receiving data
	MsgNotInterested messageID = 3 // MsgNotInterested expresses disinterest in receiving data
	MsgHave          messageID = 4 // MsgHave alerts the receiver that the sender has downloaded a piece
	MsgBitfield      messageID = 5 // MsgBitfield encodes which pieces that the sender has downloaded
	MsgRequest       messageID = 6 // MsgRequest requests a block of data from the receiver
	MsgPiece         messageID = 7 // MsgPiece delivers a block of data to fulfill a request
	MsgCancel        messageID = 8 // MsgCancel cancels a request
)

var keepAlive = []byte{0, 0, 0, 0}

type Message struct {
	ID   messageID
	Data []byte
}

func (m *Message) Serialize() []byte {
	if m == nil {
		return keepAlive
	}

	msgLen := uint32(1 + len(m.Data))
	buf := make([]byte, lenSize+msgLen)
	// message len
	binary.BigEndian.PutUint32(buf, msgLen)
	// id
	buf[lenSize] = byte(m.ID)
	// message data
	copy(buf[lenSize+1:], m.Data)

	return buf
}

func Read(r io.Reader) (*Message, error) {
}
