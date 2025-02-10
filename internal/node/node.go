package node

import (
	"sync"

	"github.com/ogzhanolguncu/gossip-protocol/internal/protocol"
	"github.com/ogzhanolguncu/gossip-protocol/internal/transport"
)

type Node struct {
	addr      string
	counter   uint64
	version   uint32
	transport protocol.Transport
	peers     *PeerManager
	mu        sync.RWMutex
}

func NewNode(addr string, counter uint64, version uint32) (*Node, error) {
	t, err := transport.NewTCPTransport(addr)
	if err != nil {
		return nil, err
	}

	n := &Node{
		addr:      addr,
		counter:   counter,
		version:   version,
		transport: t,
		peers:     NewPeerManager(),
	}

	t.Listen(n.handleMessage)
	return n, nil
}

func (n *Node) handleMessage(msg *protocol.Message) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	switch msg.Type {
	case protocol.MessageTypePush:
		if msg.Version > n.version {
			n.version = msg.Version
			n.counter = msg.Counter
		}
	case protocol.MessageTypePull:
		response := &protocol.Message{
			Type:    protocol.MessageTypePush,
			Version: n.version,
			Counter: n.counter,
		}
		return n.transport.Send(n.addr, response)
	}

	return nil
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

func (n *Node) Close() error {
	return n.transport.Close()
}
