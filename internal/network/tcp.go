package network

import (
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
			// Get the peer from our static config
			peers := s.peers.GetPeers()
			if len(peers) == 0 {
				log.Printf("No peers configured, skipping gossip")
				continue
			}

			// Get first (and only) peer in ring topology
			var peer *node.Peer
			for _, p := range peers {
				peer = p
				break
			}

			// Get our current state
			_, version := s.node.GetState()

			// Actively dial the peer instead of accepting
			conn, err := net.Dial("tcp", peer.Addr)
			if err != nil {
				log.Printf("Failed to connect to peer %s: %v", peer.Addr, err)
				continue
			}

			// Send version check
			pullReq := node.EncodeVersionCheck(version)
			_, err = conn.Write(pullReq)
			if err != nil {
				log.Printf("Failed to send version check to peer %s: %v", peer.Addr, err)
				continue
			}

			// // Read response
			// response := make([]byte, 13)
			// _, err = conn.Read(response)
			// if err != nil {
			// 	log.Printf("Failed to read response from peer %s: %v", peer.Addr, err)
			// 	continue
			// }
			//
			// // Handle response based on message type
			// msgType := response[0]
			// switch msgType {
			// case node.MessageTypeVersionCheck:
			// 	peerVersion := node.DecodeVersionCheck(response)
			// 	if peerVersion > version {
			// 		// Pull their state
			// 		pullReq := node.EncodeVersionCheck(version)
			// 		_, err := conn.Write(pullReq)
			// 		if err != nil {
			// 			log.Printf("Failed to request state from peer %s: %v", peer.Addr, err)
			// 			continue
			// 		}
			// 	} else if version > peerVersion {
			// 		// Push our state
			// 		pushMsg := node.Encode(version, counter)
			// 		_, err := conn.Write(pushMsg)
			// 		if err != nil {
			// 			log.Printf("Failed to push state to peer %s: %v", peer.Addr, err)
			// 			continue
			// 		}
			// 	}
			// case node.MessageTypeCounterUpdate:
			// 	peerVersion, peerCounter := node.Decode(response)
			// 	if peerVersion > version {
			// 		s.node.UpdateCounter(peerVersion, peerCounter)
			// 	}
			// }
		}
	}()

	buffer := make([]byte, 13)
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("Something went wrong when listening incoming calls from %s", conn.RemoteAddr())
			continue
		}

		_, err = conn.Read(buffer)
		if err != nil {
			log.Printf("Something went wrong when reading incoming call to buffer")
			continue
		}

		if len(buffer) == 0 || len(buffer) < 5 || len(buffer) > 13 {
			log.Printf("Incoming call from %s sent malformed data", conn.RemoteAddr())
		}

		// Add peer to peers
		s.peers.Connect(conn.RemoteAddr().String(), conn)

		msgType := buffer[0]
		switch msgType {
		case node.MessageTypeVersionCheck:
			// if len(buffer) != 5 {
			// 	log.Printf("Incoming call from %s sent malformed version check message: expected 5 bytes, got %d",
			// 		conn.RemoteAddr(), len(buffer))
			// 	continue
			// }

			incVersion := node.DecodeVersionCheck(buffer)
			counter, version := s.node.GetState()

			if incVersion > version {
				pullReq := node.EncodeVersionCheck(version)
				_, err := conn.Write(pullReq)
				if err != nil {
					log.Printf("Could not pull current version '%d' from peer '%s'", version, conn.RemoteAddr())
					continue
				}

				response := make([]byte, 13)
				_, err = conn.Read(response)
				if err != nil {
					log.Printf("Could not read counter update from peer %s: %v", conn.RemoteAddr(), err)
					continue
				}

				// TODO: Response message type from decode and check against current msgType if there is a mismatch, bail
				peerVersion, peerCounter := node.Decode(response)
				if msgType != node.MessageTypeCounterUpdate {
					log.Printf("Expected counter update message, got message type %d from peer %s", msgType, conn.RemoteAddr())
					continue
				}

				s.node.UpdateCounter(peerVersion, peerCounter)
			} else if incVersion < version {
				pushMsg := node.Encode(version, counter)
				_, err := conn.Write(pushMsg)
				if err != nil {
					log.Printf("Could not push current version '%d' and '%d' counter to peer '%s'", version, counter, conn.RemoteAddr())
					continue
				}
			} else if incVersion == version {
				log.Printf("Could not push current version '%d' and '%d' counter to peer '%s' because version are same", version, counter, conn.RemoteAddr())
				continue
			} else {
				panic("something went wrong when checking version numbers")
			}

		case node.MessageTypeCounterUpdate:
			if len(buffer) != 13 {
				log.Printf("Incoming call from %s sent malformed counter update message: expected 13 bytes, got %d",
					conn.RemoteAddr(), len(buffer))
			}

			incVersion, incCounter := node.Decode(buffer)
			counter, version := s.node.GetState()

			if incVersion > version {
				s.node.UpdateCounter(incVersion, incCounter)
			} else if version > incVersion {
				pushMsg := node.Encode(version, counter)
				_, err := conn.Write(pushMsg)
				if err != nil {
					log.Printf("Could not push current version '%d' and '%d' counter to peer '%s'", version, counter, conn.RemoteAddr())
					continue
				}
			} else if version == incVersion {
				log.Printf("Could not push current version '%d' and '%d' counter to peer '%s'", version, counter, conn.RemoteAddr())
				continue
			} else {
				panic("something went wrong when checking version numbers")
			}
		default:
			log.Printf("Incoming call from %s sent unknown message type: 0x%02x",
				conn.RemoteAddr(), buffer[0])
		}
	}
}

// // This should ask or answer to peers incoming call.
// // If peer is calling us we should write our version number to him, if we are calling calling peer we should ask its local version.
// func (s *Server) exchangeVersionNumbers() {}
//
// // This should update node's local counter value hence local version number.
// // This should be invoked only if exchanged number is higher than our local number.
// func (s *Server) updateCounter() {}
