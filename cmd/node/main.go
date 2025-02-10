package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ogzhanolguncu/gossip-protocol/internal/node"
)

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	nodes := make(map[string]*node.Node)

	// Start nodes first
	configs := []struct {
		addr     string
		counter  uint64
		version  uint32
		peerAddr string
	}{
		{"localhost:8001", 1, 1, "localhost:8002"},
		{"localhost:8002", 2, 2, "localhost:8003"},
		{"localhost:8003", 3, 3, "localhost:8001"},
	}

	for _, cfg := range configs {
		n, err := node.NewNode(cfg.addr, cfg.counter, cfg.version, 2*time.Second)
		if err != nil {
			log.Printf("Failed to start node %s: %v", cfg.addr, err)
			continue
		}
		nodes[cfg.addr] = n
	}

	// Connect peers after all nodes are started
	for addr, n := range nodes {
		for _, cfg := range configs {
			if cfg.addr == addr {
				n.AddPeer(cfg.peerAddr)
				break
			}
		}
	}

	// Wait for shutdown
	<-sigChan
	log.Println("Shutting down...")

	for addr, n := range nodes {
		counter, version := n.GetState()
		log.Printf("Node %s final state - Version: %d, Counter: %d",
			addr, version, counter)
		n.Close()
	}
}
