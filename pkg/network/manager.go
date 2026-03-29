package network

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const (
	ContainerName       = "bedrock-xrpl-node"
	DefaultNodeReadyTimeout = 30 * time.Second
)

// Manager handles Docker-based XRPL node
type Manager struct {
	docker        *client.Client
	ledgerService *LedgerService
}

// NewManager creates a new network manager
func NewManager() (*Manager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &Manager{docker: cli}, nil
}

// Start starts the local XRPL node
func (m *Manager) Start(ctx context.Context, opts StartOptions) error {
	// Check if container already exists
	existing, err := m.getContainer(ctx)
	if err == nil && existing != nil {
		return fmt.Errorf("node is already running (container: %s)", existing.ID[:12])
	}

	// Pull the Docker image
	if err := m.pullImage(ctx, opts.DockerImage); err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	// Configure port bindings
	portBindings := nat.PortMap{
		"6006/tcp":  []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "6006"}},
		"5005/tcp":  []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "5005"}},
		"51235/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "51235"}},
	}

	// Create container using entrypoint/cmd/binds from config
	containerCfg := &container.Config{
		Image: opts.DockerImage,
		Cmd:   opts.Cmd,
		ExposedPorts: nat.PortSet{
			"6006/tcp":  struct{}{},
			"5005/tcp":  struct{}{},
			"51235/tcp": struct{}{},
		},
	}
	if len(opts.Entrypoint) > 0 {
		containerCfg.Entrypoint = opts.Entrypoint
	}

	resp, err := m.docker.ContainerCreate(ctx,
		containerCfg,
		&container.HostConfig{
			PortBindings: portBindings,
			Binds:        opts.Binds,
			AutoRemove:   false,
		},
		nil,
		nil,
		ContainerName,
	)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := m.docker.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Start ledger service if interval is configured
	if opts.LedgerInterval > 0 && opts.RPCURL != "" {
		ledgerService, err := NewLedgerService(opts.RPCURL, opts.LedgerInterval)
		if err != nil {
			// Log warning but don't fail - node is already running
			fmt.Printf("Warning: failed to create ledger service: %v\n", err)
			return nil
		}

		// Wait for node to be ready before starting ledger service
		if err := ledgerService.WaitForReady(ctx, DefaultNodeReadyTimeout); err != nil {
			fmt.Printf("Warning: timeout waiting for node to be ready: %v\n", err)
			return nil
		}

		// Start the ledger service with a background context so it continues
		// running even after the CLI command returns. The service will be
		// stopped when Stop() is called or when the process exits.
		if err := ledgerService.Start(context.Background()); err != nil {
			fmt.Printf("Warning: failed to start ledger service: %v\n", err)
			return nil
		}

		m.ledgerService = ledgerService
	}

	return nil
}

// Stop stops the local XRPL node
func (m *Manager) Stop(ctx context.Context) error {
	// Stop ledger service first
	if m.ledgerService != nil {
		m.ledgerService.Stop()
		m.ledgerService = nil
	}

	containerInfo, err := m.getContainer(ctx)
	if err != nil {
		return fmt.Errorf("node is not running")
	}

	// Stop container
	timeout := 10
	if err := m.docker.ContainerStop(ctx, containerInfo.ID, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Remove container
	if err := m.docker.ContainerRemove(ctx, containerInfo.ID, container.RemoveOptions{}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	return nil
}

// Status returns the status of the local node
func (m *Manager) Status(ctx context.Context) (*NodeStatus, error) {
	containerInfo, err := m.getContainer(ctx)
	if err != nil {
		return &NodeStatus{Running: false}, nil
	}

	status := &NodeStatus{
		Running:     containerInfo.State.Running,
		ContainerID: containerInfo.ID[:12],
		Image:       containerInfo.Config.Image,
		Ports:       formatPorts(containerInfo.NetworkSettings.Ports),
	}

	// Add ledger service status if available
	if m.ledgerService != nil {
		ledgerStatus := m.ledgerService.GetStatus()
		status.LedgerServiceRunning = ledgerStatus.Running
		status.LedgersAdvanced = ledgerStatus.LedgersAdvanced
		status.LastLedgerIndex = ledgerStatus.LastLedgerIndex
	}

	return status, nil
}

// Close closes the Docker client
func (m *Manager) Close() error {
	return m.docker.Close()
}

func (m *Manager) pullImage(ctx context.Context, imageName string) error {
	reader, err := m.docker.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		// Pull failed - check if the image exists locally (e.g. locally-built arm64 image)
		if m.imageExistsLocally(ctx, imageName) {
			return nil
		}
		return fmt.Errorf("image %q not found remotely or locally: %w", imageName, err)
	}
	defer reader.Close()

	// Read the pull output (required for pull to complete)
	_, err = io.Copy(io.Discard, reader)
	return err
}

func (m *Manager) imageExistsLocally(ctx context.Context, imageName string) bool {
	_, _, err := m.docker.ImageInspectWithRaw(ctx, imageName)
	return err == nil
}

func (m *Manager) getContainer(ctx context.Context) (*types.ContainerJSON, error) {
	containerJSON, err := m.docker.ContainerInspect(ctx, ContainerName)
	if err != nil {
		return nil, err
	}
	return &containerJSON, nil
}

func formatPorts(ports nat.PortMap) []string {
	var result []string
	for port, bindings := range ports {
		for _, binding := range bindings {
			result = append(result, fmt.Sprintf("%s->%s", binding.HostPort, port))
		}
	}
	return result
}
