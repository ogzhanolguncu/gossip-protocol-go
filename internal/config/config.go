package config

import "time"

type Config struct {
	// Address this node listens on (e.g., "localhost:8080")
	ListenAddr string

	// List of peer addresses to connect to (e.g., ["localhost:8081", "localhost:8082"])
	PeerAddresses []string

	// How often to attempt gossip with peers (e.g., every 5 seconds)
	GossipInterval time.Duration

	// Initial counter value for this node
	InitialCounter uint64
}

// You could load this from environment variables, command line flags, or a config file
func LoadConfig() (*Config, error) {
	return nil, nil
}
