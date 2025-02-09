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

func NewPeerManager() *PeerManager {
	return &PeerManager{
		peers: make(map[string]*Peer),
	}
}

func (pm *PeerManager) AddPeer(addr string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.peers[addr]; !exists {
		pm.peers[addr] = &Peer{
			Addr: addr,
		}
	}
}

func (pm *PeerManager) Connect(addr string, conn net.Conn) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if peer, exists := pm.peers[addr]; exists {
		peer.Conn = conn
	} else {
		pm.peers[addr] = &Peer{
			Addr: addr,
			Conn: conn,
		}
	}
}

func (pm *PeerManager) Disconnect(addr string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if peer, exists := pm.peers[addr]; exists {
		if peer.Conn != nil {
			peer.Conn.Close()
		}
		peer.Conn = nil
	}
}

func (pm *PeerManager) UpdateVersionNumber(addr string, version uint32) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if peer, exists := pm.peers[addr]; exists {
		peer.Version = version
	}
}

func (pm *PeerManager) GetPeers() map[string]*Peer {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	// Return a copy to prevent concurrent map access
	peersCopy := make(map[string]*Peer, len(pm.peers))
	for k, v := range pm.peers {
		peersCopy[k] = v
	}
	return peersCopy
}
