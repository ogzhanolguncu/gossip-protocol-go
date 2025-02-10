package node

import "sync"

type Peer struct {
	Addr    string
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
		pm.peers[addr] = &Peer{Addr: addr}
	}
}

func (pm *PeerManager) RemovePeer(addr string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.peers, addr)
}

func (pm *PeerManager) UpdateVersion(addr string, version uint32) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if peer, exists := pm.peers[addr]; exists {
		peer.Version = version
	}
}

func (pm *PeerManager) GetPeers() []*Peer {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	peers := make([]*Peer, 0, len(pm.peers))
	for _, p := range pm.peers {
		peers = append(peers, &Peer{
			Addr:    p.Addr,
			Version: p.Version,
		})
	}
	return peers
}
