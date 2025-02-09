package network

import (
	"fmt"
	"log"
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
	defer s.listener.Close()

	go func() {
		ticker := time.NewTicker(s.gossipInterval)
		defer ticker.Stop()

		for range ticker.C {
			if err := s.doGossipRound(); err != nil {
				log.Printf("[Node %s] Gossip round failed: %v", s.addr, err)
			}
		}
	}()

	log.Printf("[Node %s] Starting to accept connections", s.addr)
	buffer := make([]byte, node.MaxMessageSize)
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("[Node %s] Error accepting connection: %v", s.addr, err)
			continue
		}
		log.Printf("[Node %s] Accepted connection from %s", s.addr, conn.RemoteAddr())

		_, err = conn.Read(buffer)
		if err != nil {
			log.Printf("[Node %s] Error reading from %s: %v", s.addr, conn.RemoteAddr(), err)
			continue
		}

		isValid := validateMessage(buffer, s, conn)
		if !isValid {
			continue
		}

		s.peers.Connect(conn.RemoteAddr().String(), conn)

		msgType := buffer[0]
		counter, version := s.node.GetState()

		switch msgType {
		case node.MessageTypeVersionCheck:
			incVersion := node.DecodeVersionCheck(buffer)
			log.Printf("[Node %s] Received version check from %s (their version=%d, my version=%d)",
				s.addr, conn.RemoteAddr(), incVersion, version)

			if incVersion > version {
				log.Printf("[Node %s] Their version is higher, requesting update", s.addr)
				err := s.pullVersion(version, conn)
				if err != nil {
					log.Printf("[Node %s] Pull version failed: %v", s.addr, err)
					continue
				}

			} else if incVersion < version {
				log.Printf("[Node %s] Their version is lower, sending update", s.addr)
				pushMsg := node.EncodeCounterUpdate(version, counter)
				_, err := conn.Write(pushMsg)
				if err != nil {
					log.Printf("[Node %s] Failed to send update to %s: %v", s.addr, conn.RemoteAddr(), err)
					continue
				}
			} else {
				log.Printf("[Node %s] Versions are equal (%d), no update needed", s.addr, version)
			}
		default:
			log.Printf("[Node %s] Received unknown message type from %s: 0x%02x",
				s.addr, conn.RemoteAddr(), buffer[0])
		}
	}
}

func (s *Server) doGossipRound() error {
	peers := s.peers.GetPeers()
	if len(peers) == 0 {
		log.Printf("[Node %s] No peers configured, skipping gossip", s.addr)
		return nil
	}

	var peer *node.Peer
	for _, p := range peers {
		peer = p
		break
	}

	counter, version := s.node.GetState()
	log.Printf("[Node %s] Starting gossip round with peer %s (my state: version=%d, counter=%d)",
		s.addr, peer.Addr, version, counter)

	conn, err := net.Dial("tcp", peer.Addr)
	if err != nil {
		return fmt.Errorf("failed to connect to peer %s: %v", peer.Addr, err)
	}

	err = s.pushVersion(version, conn, peer)
	if err != nil {
		return fmt.Errorf("[Node %s] Push version failed: %v", s.addr, err)
	}

	return nil
}

func (s *Server) pushVersion(version uint32, conn net.Conn, peer *node.Peer) error {
	pushReq := node.EncodeVersionCheck(version)
	_, err := conn.Write(pushReq)
	if err != nil {
		return fmt.Errorf("failed to send version check to peer %s: %v", peer.Addr, err)
	}
	log.Printf("[Node %s] Sent version check (version=%d) to peer %s", s.addr, version, peer.Addr)

	response := make([]byte, node.MaxMessageSize)
	_, err = conn.Read(response)
	if err != nil {
		return fmt.Errorf("failed to read update from %s: %v", conn.RemoteAddr(), err)
	}

	peerVersion, peerCounter := node.Decode(response)
	log.Printf("[Node %s] Received update from %s (their version=%d, counter=%d)",
		s.addr, conn.RemoteAddr(), peerVersion, peerCounter)

	s.node.UpdateCounter(peerVersion, peerCounter)
	log.Printf("[Node %s] Updated state (new version=%d, new counter=%d)",
		s.addr, peerVersion, peerCounter)
	return nil
}

func (s *Server) pullVersion(version uint32, conn net.Conn) error {
	pullReq := node.EncodeVersionCheck(version)
	_, err := conn.Write(pullReq)
	if err != nil {
		return fmt.Errorf("failed to request update from %s: %v", conn.RemoteAddr(), err)
	}

	response := make([]byte, node.MaxMessageSize)
	_, err = conn.Read(response)
	if err != nil {
		return fmt.Errorf("failed to read update from %s: %v", conn.RemoteAddr(), err)
	}

	peerVersion, peerCounter := node.Decode(response)
	log.Printf("[Node %s] Received update from %s (their version=%d, counter=%d)",
		s.addr, conn.RemoteAddr(), peerVersion, peerCounter)

	s.node.UpdateCounter(peerVersion, peerCounter)
	log.Printf("[Node %s] Updated state (new version=%d, new counter=%d)",
		s.addr, peerVersion, peerCounter)

	return nil
}

func validateMessage(buffer []byte, s *Server, conn net.Conn) bool {
	bufferLength := len(buffer)
	if bufferLength < node.MinMessageSize {
		log.Printf("[Node %s] Message too short from %s: got %d bytes, minimum is %d bytes",
			s.addr, conn.RemoteAddr(), bufferLength, node.MinMessageSize)
		return false
	}
	if bufferLength > node.MaxMessageSize {
		log.Printf("[Node %s] Message too long from %s: got %d bytes, maximum is %d bytes",
			s.addr, conn.RemoteAddr(), bufferLength, node.MaxMessageSize)
		return false
	}
	if bufferLength != node.MinMessageSize && bufferLength != node.MaxMessageSize {
		log.Printf("[Node %s] Invalid message size from %s: got %d bytes, expected either %d or %d bytes",
			s.addr, conn.RemoteAddr(), bufferLength, node.MinMessageSize, node.MaxMessageSize)
		return false
	}
	return true
}
