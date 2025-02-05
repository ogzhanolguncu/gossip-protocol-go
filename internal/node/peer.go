package node

import (
	"net"
	"sync"
)

type Peer struct {
	Addr    string
	Conn    net.Conn
	Version uint32
}

type PeerManager struct {
	mu    sync.RWMutex
	peers map[string]*Peer
}

func (pm *PeerManager) Connect(addr string, conn net.Conn) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.peers[addr] = &Peer{
		Addr: addr,
		Conn: conn,
	}
}
