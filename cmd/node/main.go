package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ogzhanolguncu/gossip-protocol/internal/discovery"
	"github.com/ogzhanolguncu/gossip-protocol/internal/node"
)

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start discovery server
	discoveryServer := discovery.NewDiscoveryServer("localhost:8000")
	go discoveryServer.Start()

	nodes := make(map[string]*node.Node)
	discoveryClients := make(map[string]*discovery.DiscoveryClient)

	// Node configurations
	configs := []struct {
		addr    string
		counter uint64
		version uint32
	}{
		{"localhost:8001", 1, 1},
		{"localhost:8002", 2, 2},
		{"localhost:8003", 3, 3},
		{"localhost:8004", 4, 4},
		{"localhost:8005", 5, 5},
		{"localhost:8006", 6, 6},
		{"localhost:8007", 7, 7},
		{"localhost:8008", 8, 8},
	}

	// Start nodes and register with discovery
	for _, cfg := range configs {
		// Create and start node
		n, err := node.NewNode(cfg.addr, cfg.counter, cfg.version, 2*time.Second)
		if err != nil {
			log.Printf("Failed to start node %s: %v", cfg.addr, err)
			continue
		}
		nodes[cfg.addr] = n

		// Create discovery client
		dc := discovery.NewDiscoveryClient("localhost:8000", n)
		discoveryClients[cfg.addr] = dc

		// Register with discovery server
		if err := dc.Register(); err != nil {
			log.Printf("Failed to register node %s: %v", cfg.addr, err)
			continue
		}

		// Start peer discovery (check every 10 seconds)
		dc.StartDiscovery(10 * time.Second)

		// Start heartbeat (every 30 seconds)
		dc.StartHeartbeat(30 * time.Second)
	}

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down...")

	// Clean up nodes and discovery clients
	for addr, n := range nodes {
		counter, version := n.GetState()
		log.Printf("Node %s final state - Version: %d, Counter: %d",
			addr, version, counter)

		// Clean up discovery client
		if dc, exists := discoveryClients[addr]; exists {
			dc.Close()
		}

		// Clean up node
		n.Close()
	}

	// Stop discovery server
	discoveryServer.Stop()
}
