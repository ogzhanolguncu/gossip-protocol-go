package discovery

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

type DiscoveryServer struct {
	addr        string
	knownPeers  map[string]*PeerInfo
	mu          sync.RWMutex
	httpServer  *http.Server
	lastCleanup time.Time
}

type PeerInfo struct {
	Addr     string    `json:"addr"`
	LastSeen time.Time `json:"last_seen"`
	Version  uint32    `json:"version"`
	IsActive bool      `json:"is_active"`
}

type RegisterRequest struct {
	Addr    string `json:"addr"`
	Version uint32 `json:"version"`
}

func NewDiscoveryServer(addr string) *DiscoveryServer {
	ds := &DiscoveryServer{
		addr:        addr,
		knownPeers:  make(map[string]*PeerInfo),
		lastCleanup: time.Now(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/register", ds.handleRegister)
	mux.HandleFunc("/peers", ds.handlePeerList)
	mux.HandleFunc("/heartbeat", ds.handleHeartbeat)

	ds.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start cleanup goroutine
	go ds.cleanupInactivePeers()

	return ds
}

func (ds *DiscoveryServer) Start() error {
	log.Printf("[Discovery] Starting server on %s", ds.addr)
	return ds.httpServer.ListenAndServe()
}

func (ds *DiscoveryServer) Stop() error {
	return ds.httpServer.Close()
}

func (ds *DiscoveryServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ds.mu.Lock()
	ds.knownPeers[req.Addr] = &PeerInfo{
		Addr:     req.Addr,
		LastSeen: time.Now(),
		Version:  req.Version,
		IsActive: true,
	}
	ds.mu.Unlock()

	log.Printf("[Discovery] Registered new peer: %s", req.Addr)
	w.WriteHeader(http.StatusOK)
}

func (ds *DiscoveryServer) handlePeerList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ds.mu.RLock()
	activePeers := make([]*PeerInfo, 0)
	for _, peer := range ds.knownPeers {
		if peer.IsActive {
			activePeers = append(activePeers, peer)
		}
	}
	ds.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(activePeers)
}

func (ds *DiscoveryServer) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ds.mu.Lock()
	if peer, exists := ds.knownPeers[req.Addr]; exists {
		peer.LastSeen = time.Now()
		peer.Version = req.Version
		peer.IsActive = true
	}
	ds.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

func (ds *DiscoveryServer) cleanupInactivePeers() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ds.mu.Lock()
		now := time.Now()
		for _, peer := range ds.knownPeers {
			// Mark peers as inactive if not seen in the last minute
			if now.Sub(peer.LastSeen) > time.Minute {
				peer.IsActive = false
				log.Printf("[Discovery] Marked peer as inactive: %s", peer.Addr)
			}
		}
		ds.mu.Unlock()
	}
}
