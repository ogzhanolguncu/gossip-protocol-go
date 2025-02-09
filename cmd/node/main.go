package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ogzhanolguncu/gossip-protocol/internal/network"
	"github.com/ogzhanolguncu/gossip-protocol/internal/node"
)

func main() {
	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a WaitGroup to track all nodes
	var wg sync.WaitGroup

	nodes := map[int]struct {
		addrs   [2]string
		version uint32
	}{
		1: {[2]string{"localhost:8001", "localhost:8002"}, 1},
		2: {[2]string{"localhost:8002", "localhost:8003"}, 2},
		3: {[2]string{"localhost:8003", "localhost:8001"}, 3},
	}

	for id, config := range nodes {
		myAddr, peerAddr := config.addrs[0], config.addrs[1]
		initVersion := config.version
		wg.Add(1)

		err := startNode(ctx, &wg, uint64(id), myAddr, peerAddr, initVersion)
		if err != nil {
			log.Printf("Failed to start node %d: %v", id, err)
			continue
		}
	}

	<-sigChan
	log.Println("Received shutdown signal, initiating graceful shutdown...")
	cancel()

	shutdownComplete := make(chan struct{})
	go func() {
		wg.Wait()
		close(shutdownComplete)
	}()

	select {
	case <-shutdownComplete:
		log.Println("All nodes shut down successfully")
	case <-time.After(10 * time.Second):
		log.Println("Shutdown timed out")
	}
}

func startNode(ctx context.Context, wg *sync.WaitGroup, id uint64, addr string, peerAddr string, version uint32) error {
	defer wg.Done()

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start listener on %s: %v", addr, err)
	}

	peerManager := node.NewPeerManager()
	peerManager.AddPeer(peerAddr)
	n := node.NewNode(addr, id, version)

	server := network.NewServer(
		addr,
		listener,
		n,
		peerManager,
		2*time.Second,
	)

	// Start server with context
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("Node %d server error: %v", id, err)
		}
	}()

	return nil
}
