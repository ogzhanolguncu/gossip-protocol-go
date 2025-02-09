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

	// If we already have this peer
	if peer, exists := pm.peers[addr]; exists {
		// Close existing connection if it exists
		if peer.Conn != nil {
			peer.Conn.Close()
		}
		peer.Conn = conn
	} else {
		// New peer
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
			peer.Conn = nil
		}
	}
}

func (pm *PeerManager) UpdateVersionNumber(addr string, version uint32) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if peer, exists := pm.peers[addr]; exists {
		peer.Version = version
	}
}

// GetPeers now only returns peers with valid addresses (not ephemeral ports)
func (pm *PeerManager) GetPeers() map[string]*Peer {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peersCopy := make(map[string]*Peer, len(pm.peers))
	for k, v := range pm.peers {
		// Only copy peers with original addresses (not ephemeral ports)
		if v.Addr == k {
			peersCopy[k] = &Peer{
				Addr:    v.Addr,
				Conn:    v.Conn,
				Version: v.Version,
			}
		}
	}
	return peersCopy
}
