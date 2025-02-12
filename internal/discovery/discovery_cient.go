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
