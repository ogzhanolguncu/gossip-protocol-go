package network

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

// HTTP SERVER
type HTTPServer struct {
	server *Server
	port   int
	mux    *http.ServeMux
	once   sync.Once
}

// NewHTTPServer creates a new HTTP server wrapper
func NewHTTPServer(server *Server, port int) *HTTPServer {
	return &HTTPServer{
		server: server,
		port:   port,
		mux:    http.NewServeMux(),
	}
}

// StartHTTP starts the HTTP server
func (h *HTTPServer) StartHTTP() error {
	// Register routes only once using sync.Once
	h.once.Do(func() {
		h.mux.HandleFunc("/update", h.handleUpdate)
		h.mux.HandleFunc("/state", h.handleState)
	})

	// Start HTTP server with our custom mux
	addr := fmt.Sprintf(":%d", h.port)
	return http.ListenAndServe(addr, h.mux)
}

// Response structs
type StateResponse struct {
	Value   uint64 `json:"value"`
	Version uint32 `json:"version"`
}

type UpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Value   uint64 `json:"value"`
	Version uint32 `json:"version"`
}

// handleUpdate handles POST requests to update the value and increment version
func (h *HTTPServer) handleUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse value from query parameter
	valueStr := r.URL.Query().Get("value")
	if valueStr == "" {
		http.Error(w, "Missing 'value' parameter", http.StatusBadRequest)
		return
	}

	value, err := strconv.ParseUint(valueStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid value", http.StatusBadRequest)
		return
	}

	// Get current version and increment by 1
	_, currentVersion := h.server.Node.GetState()
	newVersion := currentVersion + 1

	// Update value with incremented version
	h.server.Node.UpdateCounter(newVersion, value)

	// Return success response with new state
	response := UpdateResponse{
		Success: true,
		Message: fmt.Sprintf("Value updated to %d with version %d", value, newVersion),
		Value:   value,
		Version: newVersion,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleState returns the current state
func (h *HTTPServer) handleState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	value, version := h.server.Node.GetState()
	response := StateResponse{
		Value:   value,
		Version: version,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
