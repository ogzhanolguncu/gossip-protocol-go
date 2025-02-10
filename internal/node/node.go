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

func (n *Node) sync() {
	peers := n.peers.GetPeers()
	if len(peers) == 0 {
		log.Printf("[Node %s] No peers available for sync", n.addr)
		return
	}

	log.Printf("[Node %s] Starting sync round with %d peers. Current state: version=%d, counter=%d",
		n.addr, len(peers), n.version, n.counter)

	// Randomly select peers to sync with
	numPeers := min(n.maxSyncPeers, len(peers))
	selectedPeers := make([]*Peer, len(peers))
	copy(selectedPeers, peers)
	rand.Shuffle(len(selectedPeers), func(i, j int) {
		selectedPeers[i], selectedPeers[j] = selectedPeers[j], selectedPeers[i]
	})

	for _, peer := range selectedPeers[:numPeers] {
		msg := &protocol.Message{
			Type:    protocol.MessageTypePull,
			Version: n.version,
			Counter: n.counter,
		}
		n.transport.Send(peer.Addr, msg)
	}
}

func (n *Node) handleMessage(msg *protocol.Message) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	log.Printf("[Node %s] Received message type=%d, version=%d, counter=%d",
		n.addr, msg.Type, msg.Version, msg.Counter)

	switch msg.Type {
	case protocol.MessageTypePull:
		if msg.Version < n.version {
			// Send our higher version
			response := &protocol.Message{
				Type:    protocol.MessageTypePush,
				Version: n.version,
				Counter: n.counter,
			}
			return n.transport.Send(n.addr, response)
		} else if msg.Version > n.version {
			// Update our state and acknowledge
			n.version = msg.Version
			n.counter = msg.Counter
			response := &protocol.Message{
				Type:    protocol.MessageTypePush,
				Version: msg.Version,
				Counter: msg.Counter,
			}
			return n.transport.Send(n.addr, response)
		}
	case protocol.MessageTypePush:
		if msg.Version > n.version {
			n.version = msg.Version
			n.counter = msg.Counter
			log.Printf("[Node %s] Updated state to version=%d, counter=%d",
				n.addr, n.version, n.counter)
		}
	}
	return nil
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
