package network

import "time"

// StartOptions configures how to start the local node
type StartOptions struct {
	DockerImage    string
	Entrypoint     []string
	Cmd            []string
	Binds          []string
	ConfigDir      string
	LedgerInterval time.Duration
	RPCURL         string
}

// NodeStatus represents the current status of the node
type NodeStatus struct {
	Running     bool
	ContainerID string
	Image       string
	Ports       []string
	// Ledger service status
	LedgerServiceRunning bool
	LedgersAdvanced      uint64
	LastLedgerIndex      uint64
}
