package network

import (
	"net"
	"time"

	"github.com/ogzhanolguncu/gossip-protocol/internal/node"
)

type Server struct {
	addr           string
	listener       net.Listener
	node           *node.Node
	peers          *node.PeerManager
	gossipInterval time.Duration
}

func NewServer(addr string, listener net.Listener, node *node.Node, peers *node.PeerManager, gossipInterval time.Duration) *Server {
	return &Server{
		addr,
		listener,
		node,
		peers,
		gossipInterval,
	}
}

// Set up a TCP listener for incoming connections
// Start a goroutine to accept and handle new connections
// Start periodic push-pull with peers where:
//   - Select a random peer
//   - Exchange version numbers with peer
//   - If our version is higher: we push our counter to them
//   - If their version is higher: we pull their counter
//   - If versions are equal: no data exchange needed
//   - Update local version number if we pulled new data
func (s *Server) Start() error {
	return nil
}
