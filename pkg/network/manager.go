package network

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
)

// dockerPlatform returns the Docker `linux/<arch>` platform string for the
// host. Falling back to amd64 means an Apple Silicon host without an arm64
// image still works (via Rosetta) but a host that has an arm64-only image
// available locally is no longer forced through an unnecessary amd64 pull.
func dockerPlatform() (osName, arch string) {
	switch runtime.GOARCH {
	case "arm64", "aarch64":
		return "linux", "arm64"
	default:
		return "linux", "amd64"
	}
}

const (
	ContainerName           = "bedrock-xrpl-node"
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

	// Pull the Docker image, preferring the host-native platform with an amd64
	// fallback (see pullImage).
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

	// The platform argument is left nil so Docker uses the image that pullImage
	// actually obtained. We must not assert a platform here: the classic
	// (non-containerd) image store ignores the --platform hint when pulling a
	// single-arch image, so it may have fetched amd64 even though arm64 was
	// requested — asserting arm64 would then fail container creation. A nil
	// platform runs whatever was obtained (under emulation if needed), matching
	// `docker run`'s default behaviour.
	resp, err := m.docker.ContainerCreate(ctx,
		containerCfg,
		&container.HostConfig{
			PortBindings: portBindings,
			Binds:        opts.Binds,
			AutoRemove:   false,
		},
		nil, // networking config
		nil, // platform (see note above)
		ContainerName,
	)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := m.docker.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Start ledger service if interval is configured. The node container is
	// already running, so failures here leave the user with a usable (if
	// degraded) environment — we surface warnings on stderr instead of
	// swallowing them onto stdout the way the previous code did.
	if opts.LedgerInterval > 0 && opts.RPCURL != "" {
		ledgerService, err := NewLedgerService(opts.RPCURL, opts.LedgerInterval)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create ledger service: %v\n", err)
			return nil
		}

		if err := ledgerService.WaitForReady(ctx, DefaultNodeReadyTimeout); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: node not ready within %s: %v\n", DefaultNodeReadyTimeout, err)
			return nil
		}

		// Start the ledger service with a background context so it continues
		// running even after the CLI command returns. The service will be
		// stopped when Stop() is called or when the process exits.
		if err := ledgerService.Start(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to start ledger service: %v\n", err)
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

// StreamLogs streams the container's stdout/stderr to the provided writer.
// Set follow=true to keep streaming as new lines arrive (like `docker logs -f`).
// Returns an error if the node is not running.
func (m *Manager) StreamLogs(ctx context.Context, w io.Writer, follow bool, tail string) error {
	containerInfo, err := m.getContainer(ctx)
	if err != nil {
		return fmt.Errorf("node is not running")
	}

	reader, err := m.docker.ContainerLogs(ctx, containerInfo.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Tail:       tail,
		Timestamps: false,
	})
	if err != nil {
		return fmt.Errorf("failed to open container logs: %w", err)
	}
	defer reader.Close()

	// rippled containers run without a TTY, so ContainerLogs returns Docker's
	// multiplexed stream, where each frame carries an 8-byte stdout/stderr
	// header. stdcopy de-multiplexes it back into clean output; a raw io.Copy
	// would dump those frame headers into the terminal as binary garbage.
	if _, err := stdcopy.StdCopy(w, w, reader); err != nil && err != io.EOF {
		return fmt.Errorf("error reading logs: %w", err)
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

// pullImage pulls imageName, preferring the host-native platform and falling
// back to linux/amd64 (which runs under emulation on Apple Silicon) before
// giving up on the registry. The container is created with a nil platform so it
// always runs whatever this pull actually obtained — see the note in Start for
// why the requested platform must not be asserted at create time.
func (m *Manager) pullImage(ctx context.Context, imageName string) error {
	platOS, platArch := dockerPlatform()

	// Prefer the host-native platform, then fall back to amd64. The amd64
	// fallback matters for amd64-only multi-arch manifests, where the daemon
	// reports an in-stream "no matching manifest" error for the arm64 request.
	candidates := []string{fmt.Sprintf("%s/%s", platOS, platArch)}
	if platArch != "amd64" {
		candidates = append(candidates, "linux/amd64")
	}

	var lastErr error
	for _, plat := range candidates {
		reader, err := m.docker.ImagePull(ctx, imageName, image.PullOptions{Platform: plat})
		if err != nil {
			lastErr = err
			continue
		}
		// Drain the pull stream. Pull failures such as "no matching manifest
		// for <platform>" are reported inside this JSON stream, so we must
		// inspect it rather than discarding the bytes blindly.
		err = drainPullStream(reader)
		reader.Close()
		if err != nil {
			lastErr = err
			continue
		}
		return nil
	}

	// Every remote pull failed - fall back to a locally present image (e.g. one
	// built locally for this host's architecture).
	if m.imageExistsLocally(ctx, imageName) {
		return nil
	}
	return fmt.Errorf("image %q not found remotely or locally: %w", imageName, lastErr)
}

// drainPullStream reads an ImagePull progress stream to completion and returns
// any error the daemon reports inside it. The daemon delivers errors such as
// "no matching manifest for linux/arm64" as a JSON object in the stream body
// rather than as a failure of ImagePull itself, so they would be lost if the
// stream were simply discarded.
func drainPullStream(reader io.Reader) error {
	dec := json.NewDecoder(reader)
	for {
		var msg struct {
			Error string `json:"error"`
		}
		if err := dec.Decode(&msg); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if msg.Error != "" {
			return fmt.Errorf("%s", msg.Error)
		}
	}
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
