package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/ogzhanolguncu/gossip-protocol/internal/network"
	"github.com/ogzhanolguncu/gossip-protocol/internal/node"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	activeNodes := make(map[string]*network.Server)
	nodesMutex := sync.Mutex{}

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

		go func(id int, myAddr, peerAddr string, version uint32) {
			server, err := startNode(ctx, &wg, uint64(id), myAddr, peerAddr, version)
			if err != nil {
				log.Printf("Failed to start node %d: %v", id, err)
				return
			}

			nodesMutex.Lock()
			activeNodes[myAddr] = server
			nodesMutex.Unlock()
		}(id, myAddr, peerAddr, initVersion)

		time.Sleep(1 * time.Second)
	}

	sig := <-sigChan
	log.Printf("Received %v signal, initiating coordinated shutdown...", sig)
	cancel()

	nodesMutex.Lock()
	for addr, server := range activeNodes {
		counter, version := server.Node.GetState()
		log.Printf("❗ [Node %s] Final state - Version: %d, Counter: %d",
			addr, version, counter)
	}
	nodesMutex.Unlock()

	// Force immediate shutdown
	os.Exit(0)
}

func startNode(ctx context.Context, wg *sync.WaitGroup, id uint64, addr string, peerAddr string, version uint32) (*network.Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		wg.Done() // Important: don't forget to signal we're done if we fail
		return nil, fmt.Errorf("failed to start listener on %s: %v", addr, err)
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

	// Start HTTP server
	_, portStr, _ := net.SplitHostPort(addr)
	tcpPort, _ := strconv.Atoi(portStr)
	httpPort := tcpPort + 1000
	httpServer := network.NewHTTPServer(server, httpPort)

	go func() {
		if err := httpServer.StartHTTP(); err != nil {
			log.Printf("[Node %s] HTTP server failed: %v", addr, err)
		}
	}()

	// Start server in goroutine
	go func() {
		defer wg.Done()
		if err := server.Start(ctx); err != nil {
			log.Printf("[Node %s] Server failed: %v", addr, err)
		}
	}()

	return server, nil
}
