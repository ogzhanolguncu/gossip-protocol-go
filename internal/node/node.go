package node

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/ogzhanolguncu/gossip-protocol/internal/protocol"
	"github.com/ogzhanolguncu/gossip-protocol/internal/transport"
)

type Node struct {
	addr         string
	counter      uint64
	version      uint32
	transport    protocol.Transport
	peers        *PeerManager
	mu           sync.RWMutex
	syncInterval time.Duration
	maxSyncPeers int
	ctx          context.Context
	cancel       context.CancelFunc
}

func NewNode(addr string, counter uint64, version uint32, syncInterval time.Duration) (*Node, error) {
	t, err := transport.NewTCPTransport(addr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	n := &Node{
		addr:         addr,
		counter:      counter,
		version:      version,
		transport:    t,
		peers:        NewPeerManager(),
		syncInterval: syncInterval,
		maxSyncPeers: 3,
		ctx:          ctx,
		cancel:       cancel,
	}

	t.Listen(n.handleMessage)
	go n.startAntiEntropy()
	return n, nil
}

func (n *Node) startAntiEntropy() {
	ticker := time.NewTicker(n.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.sync()
		}
	}
}

func (n *Node) handleMessage(sourceAddr string, msg *protocol.Message) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	log.Printf("[Node %s] Received message from %s type=%d, version=%d, counter=%d",
		n.addr, sourceAddr, msg.Type, msg.Version, msg.Counter)

	switch msg.Type {
	case protocol.MessageTypePull:
		response := &protocol.Message{
			Type:    protocol.MessageTypePush,
			Version: n.version,
			Counter: n.counter,
		}

		// Update our state if pull has higher version
		if msg.Version > n.version {
			n.version = msg.Version
			n.counter = msg.Counter
			// Propagate update to others
			n.propagateUpdate(sourceAddr)
		}

		return n.transport.Send(sourceAddr, response)

	case protocol.MessageTypePush:
		// Update state if push has higher version
		if msg.Version > n.version {
			n.version = msg.Version
			n.counter = msg.Counter
			// Propagate update to others
			n.propagateUpdate(sourceAddr)
		}
	}
	return nil
}

func (n *Node) sync() {
	n.mu.RLock()
	peers := n.peers.GetPeers()
	currentVersion := n.version
	n.mu.RUnlock()

	if len(peers) == 0 {
		log.Printf("[Node %s] No peers available for sync", n.addr)
		return
	}

	numPeers := min(n.maxSyncPeers, len(peers))
	selectedPeers := make([]*Peer, len(peers))
	copy(selectedPeers, peers)
	rand.Shuffle(len(selectedPeers), func(i, j int) {
		selectedPeers[i], selectedPeers[j] = selectedPeers[j], selectedPeers[i]
	})

	// Use a WaitGroup to track sync completion
	var wg sync.WaitGroup
	for _, peer := range selectedPeers[:numPeers] {
		wg.Add(1)
		go func(peerAddr string) {
			defer wg.Done()
			msg := &protocol.Message{
				Type:    protocol.MessageTypePull,
				Version: currentVersion,
				Counter: n.counter,
			}
			if err := n.transport.Send(peerAddr, msg); err != nil {
				log.Printf("[Node %s] Failed to sync with peer %s: %v",
					n.addr, peerAddr, err)
			}
		}(peer.Addr)
	}

	// Wait for all sync attempts to complete with timeout
	syncTimeout := time.NewTimer(n.syncInterval / 2)
	syncDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(syncDone)
	}()

	select {
	case <-syncDone:
		// All syncs completed
	case <-syncTimeout.C:
		log.Printf("[Node %s] Sync round timed out", n.addr)
	}
}

// propagateUpdate sends the current state to all peers except the source
func (n *Node) propagateUpdate(sourceAddr string) {
	updateMsg := &protocol.Message{
		Type:    protocol.MessageTypePush,
		Version: n.version,
		Counter: n.counter,
	}

	peers := n.peers.GetPeers()
	for _, peer := range peers {
		// Skip the source to avoid sending the update back
		if peer.Addr == sourceAddr {
			continue
		}

		go func(peerAddr string) {
			if err := n.transport.Send(peerAddr, updateMsg); err != nil {
				log.Printf("[Node %s] Failed to propagate update to peer %s: %v",
					n.addr, peerAddr, err)
			}
		}(peer.Addr)
	}
}

func (n *Node) AddPeer(addr string) {
	n.peers.AddPeer(addr)
	log.Printf("[Node %s] Added peer %s", n.addr, addr)
}

func (n *Node) Close() error {
	n.cancel()
	return n.transport.Close()
}

func (n *Node) UpdateState(counter uint64) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.counter = counter
	n.version++
}

func (n *Node) GetState() (uint64, uint32) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.counter, n.version
}

func (n *Node) GetAddr() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.addr
}

func (n *Node) GetPeers() []*Peer {
	return n.peers.GetPeers()
}

func (n *Node) RemovePeer(peerAddr string) {
	n.peers.RemovePeer(peerAddr)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
