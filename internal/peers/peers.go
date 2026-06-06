package peers

import (
	"encoding/binary"
	"fmt"
	"net"
)

const (
	// size in bytes
	ipSize   = 4
	portSize = 2
	peerSize = ipSize + portSize
)

type Peer struct {
	IP   net.IP
	Port uint16
}

func Unmarshal(peersBin []byte) ([]Peer, error) {
	numPeers := len(peersBin) / peerSize
	if len(peersBin)%peerSize != 0 {
		return nil, fmt.Errorf("got malformed")
	}

	peers := make([]Peer, numPeers)
	for i := range peersBin {
		offset := i * peerSize
		peers[i].IP = net.IP(peersBin[offset : offset+ipSize])
		peers[i].Port = binary.BigEndian.Uint16(peersBin[offset+ipSize : offset+peerSize])
	}

	return peers, nil
}
