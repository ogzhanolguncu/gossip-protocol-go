package discovery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ogzhanolguncu/gossip-protocol/internal/node"
)

type DiscoveryClient struct {
	serverAddr      string
	localNode       *node.Node
	client          *http.Client
	discoveryTicker *time.Ticker
	done            chan struct{}
}

func NewDiscoveryClient(serverAddr string, localNode *node.Node) *DiscoveryClient {
	return &DiscoveryClient{
		serverAddr: serverAddr,
		localNode:  localNode,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		done: make(chan struct{}),
	}
}

func (dc *DiscoveryClient) Register() error {
	counter, version := dc.localNode.GetState()
	reqBody := RegisterRequest{
		Addr:    dc.localNode.GetAddr(),
		Version: version,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal register request: %w", err)
	}

	resp, err := dc.client.Post(
		fmt.Sprintf("http://%s/register", dc.serverAddr),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to register with discovery server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("discovery server returned status: %d", resp.StatusCode)
	}

	log.Printf("[Node %s] Registered with discovery server %s (counter=%d, version=%d)",
		dc.localNode.GetAddr(), dc.serverAddr, counter, version)
	return nil
}

func (dc *DiscoveryClient) StartDiscovery(interval time.Duration) {
	dc.discoveryTicker = time.NewTicker(interval)
	go func() {
		// Do initial discovery immediately
		dc.discoverPeers()

		for {
			select {
			case <-dc.discoveryTicker.C:
				dc.discoverPeers()
			case <-dc.done:
				return
			}
		}
	}()
}

func (dc *DiscoveryClient) discoverPeers() {
	resp, err := dc.client.Get(fmt.Sprintf("http://%s/peers", dc.serverAddr))
	if err != nil {
		log.Printf("[Node %s] Failed to get peer list: %v", dc.localNode.GetAddr(), err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[Node %s] Discovery server returned status: %d",
			dc.localNode.GetAddr(), resp.StatusCode)
		return
	}

	var peers []*PeerInfo
	if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
		log.Printf("[Node %s] Failed to decode peer list: %v",
			dc.localNode.GetAddr(), err)
		return
	}

	// Track current peers for cleanup
	currentPeers := make(map[string]bool)
	for _, peer := range peers {
		currentPeers[peer.Addr] = true
		if peer.Addr != dc.localNode.GetAddr() && peer.IsActive {
			dc.localNode.AddPeer(peer.Addr)
		}
	}

	// Remove peers that are no longer active
	for _, existingPeer := range dc.localNode.GetPeers() {
		if !currentPeers[existingPeer.Addr] {
			dc.localNode.RemovePeer(existingPeer.Addr)
		}
	}

	log.Printf("[Node %s] Updated peers from discovery server - active peers: %d",
		dc.localNode.GetAddr(), len(peers))
}

func (dc *DiscoveryClient) StartHeartbeat(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				_, version := dc.localNode.GetState()
				reqBody := RegisterRequest{
					Addr:    dc.localNode.GetAddr(),
					Version: version,
				}

				jsonData, err := json.Marshal(reqBody)
				if err != nil {
					log.Printf("[Node %s] Failed to marshal heartbeat: %v",
						dc.localNode.GetAddr(), err)
					continue
				}

				resp, err := dc.client.Post(
					fmt.Sprintf("http://%s/heartbeat", dc.serverAddr),
					"application/json",
					bytes.NewBuffer(jsonData),
				)
				if err != nil {
					log.Printf("[Node %s] Failed to send heartbeat: %v",
						dc.localNode.GetAddr(), err)
					continue
				}
				resp.Body.Close()
			case <-dc.done:
				return
			}
		}
	}()
}

func (dc *DiscoveryClient) Close() {
	close(dc.done)
	if dc.discoveryTicker != nil {
		dc.discoveryTicker.Stop()
	}
}
