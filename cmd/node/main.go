package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/ogzhanolguncu/gossip-protocol/internal/network"
	"github.com/ogzhanolguncu/gossip-protocol/internal/node"
)

func startNode(id uint64, addr string, peerAddr string, version uint32) error {
	// Create listener
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start listener on %s: %v", addr, err)
	}

	// Initialize peer manager
	peerManager := node.NewPeerManager()
	peerManager.AddPeer(peerAddr) // Add next node in ring

	// Create node with initial counter
	n := node.NewNode(addr, id, version) // You'll need to implement this

	// Create and start server
	server := network.NewServer(
		addr,
		listener,
		n,
		peerManager,
		2*time.Second, // gossip interval
	)

	return server.Start()
}

func main() {
	// Create a 3-node ring: 8001 -> 8002 -> 8003 -> 8001
	nodes := map[int]struct {
		addrs   [2]string
		version uint32
	}{
		1: {[2]string{"localhost:8001", "localhost:8002"}, 1},
		2: {[2]string{"localhost:8002", "localhost:8003"}, 2},
		3: {[2]string{"localhost:8003", "localhost:8001"}, 3},
	}

	// Start each node in a separate goroutine
	for id, config := range nodes {
		myAddr, peerAddr := config.addrs[0], config.addrs[1]
		initVersion := config.version

		go func(id int, myAddr, peerAddr string, version uint32) {
			log.Printf("Starting node %d at %s, connecting to %s with version %d",
				id, myAddr, peerAddr, version)
			if err := startNode(uint64(id), myAddr, peerAddr, version); err != nil {
				log.Printf("Node %d failed: %v", id, err)
			}
		}(id, myAddr, peerAddr, initVersion)
	}

	// Keep main goroutine alive
	select {}
}
