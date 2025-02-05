# Gossip Protocol Implementation

A simple implementation of a gossip protocol using Go, demonstrating the push-pull approach for eventual consistency across a network of nodes.

## Overview

This project implements a basic gossip protocol where nodes periodically exchange their counter values using a push-pull mechanism. Each node maintains a counter and a version number. When nodes communicate:

- They first exchange version numbers
- If one node has a higher version, it shares its counter value with the other node
- The node with the lower version updates its state
- No data is exchanged if versions are equal

## Features

- Push-pull gossip mechanism
- Version-based conflict resolution
- TCP-based peer-to-peer communication
- Configurable gossip intervals
- Thread-safe state management

## Getting Started

### Running a Node

1. Clone the repository:

```bash
git clone github.com/ogzhanolguncu/gossip-protocol
cd gossip-protocol
```

2. Build the project:

```bash
go build
```

3. Run a node:

```bash
./gossip-protocol
```

## Configuration

The application can be configured using the `Config` struct in `config.go`:

- `ListenAddr`: Address for the node to listen on
- `PeerAddresses`: List of peer addresses to connect to
- `GossipInterval`: How often to gossip with peers
- `InitialCounter`: Starting counter value for the node

## Architecture

- `node/`: Core node logic and state management
- `network/`: TCP server implementation
- `config/`: Configuration management

## License

[Add your chosen license here]

## Contributing

Feel free to open issues and pull requests!
