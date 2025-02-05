package node

import "sync"

type Node struct {
	addr    string //  This node's address (e.g., "localhost:8080")
	counter uint64 //  Value we are gossiping about
	version uint32 //  Version number of our counter

	mu sync.RWMutex
}

func NewNode(addr string, counter uint64) *Node {
	return &Node{
		addr:    addr,
		counter: counter,
		version: 1, // Initial version starts from "1"
		mu:      sync.RWMutex{},
	}
}

// This will used only for internal increments for initial setup
func (n *Node) _UpdateCounter(counter uint64) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.counter = counter
	n.version++
}

func (n *Node) GetState() (counter uint64, version uint32) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.counter, n.version
}

func (n *Node) UpdateCounter(version uint32, counter uint64) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	if version > n.version {
		n.version = version
		n.counter = counter
		return true
	}
	return false
}
