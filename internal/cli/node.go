package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/xrpl-commons/bedrock/pkg/config"
	"github.com/xrpl-commons/bedrock/pkg/network"
)

const (
	// Default RPC URL for local node (uses HTTP RPC port)
	DefaultLocalRPCURL = "http://localhost:5005"
	// PID file for ledger daemon
	LedgerDaemonPIDFile = ".bedrock/ledger-daemon.pid"
)

var nodeCmd = &cobra.Command{
	Use:   "node <start|stop|status|logs>",
	Short: "Manage local XRPL node",
	Long: `Manage a local XRPL test node for development.

The node runs in a Docker container and provides a local network
for testing your smart contracts before deploying to testnet/mainnet.

Commands:
  start   - Start the local node
  stop    - Stop the local node
  status  - Check if node is running
  logs    - View node logs`,
	Args: cobra.ExactArgs(1),
	RunE: runNode,
}

func init() {
	rootCmd.AddCommand(nodeCmd)
}

func runNode(cmd *cobra.Command, args []string) error {
	subcommand := args[0]

	manager, err := network.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create network manager: %w", err)
	}

	ctx := cmd.Context()

	switch subcommand {
	case "start":
		return nodeStart(ctx, manager)
	case "stop":
		return nodeStop(ctx, manager)
	case "status":
		return nodeStatus(ctx, manager)
	case "logs":
		return nodeLogs(manager)
	default:
		return fmt.Errorf("unknown subcommand: %s (use: start, stop, status, logs)", subcommand)
	}
}

func nodeStart(ctx context.Context, manager *network.Manager) error {
	color.Cyan("Starting local XRPL node\n")
	fmt.Println()

	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		color.Red("✗ Failed to load config: %v\n", err)
		return err
	}

	// Convert ledger interval from milliseconds to duration
	ledgerInterval := time.Duration(cfg.LocalNode.LedgerInterval) * time.Millisecond

	// Resolve local network URLs from config
	localCfg, ok := cfg.Networks["local"]
	if !ok {
		localCfg = config.NetworkConfig{
			URL:    "ws://localhost:6006",
			RPCURL: DefaultLocalRPCURL,
		}
	}
	rpcURL := localCfg.GetRPCURL()

	opts := network.StartOptions{
		DockerImage:    cfg.LocalNode.DockerImage,
		ConfigDir:      cfg.LocalNode.ConfigDir,
		LedgerInterval: ledgerInterval,
		RPCURL:         rpcURL,
	}

	if err := manager.Start(ctx, opts); err != nil {
		color.Red("✗ Failed to start node: %v\n", err)
		return err
	}

	fmt.Println()
	color.Green("✓ Local node started successfully!\n")
	fmt.Println()

	// Start the ledger daemon in background
	if err := startLedgerDaemon(cfg.LocalNode.LedgerInterval, rpcURL); err != nil {
		color.Yellow("Warning: Failed to start ledger daemon: %v\n", err)
		color.Yellow("Ledgers will not advance automatically.\n")
	} else {
		color.Green("✓ Ledger daemon started\n")
	}

	fmt.Println()
	color.Cyan("Connection Details:\n")
	fmt.Printf("  WebSocket URL: %s\n", localCfg.URL)
	fmt.Printf("  RPC URL:       %s\n", rpcURL)
	fmt.Println()
	color.Cyan("Ledger Service:\n")
	fmt.Printf("  Auto-advance interval: %v\n", ledgerInterval)
	fmt.Println("  Ledgers will advance automatically in background")
	fmt.Println()
	color.Yellow("Tips:\n")
	color.Yellow("   Deploy with: bedrock deploy --network local\n")
	color.Yellow("   View logs with: bedrock node logs\n")
	color.Yellow("   Stop with: bedrock node stop\n")

	return nil
}

// startLedgerDaemon spawns the ledger daemon as a background process
func startLedgerDaemon(intervalMs int, rpcURL string) error {
	// Get the path to the current executable
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Create the .bedrock directory if it doesn't exist
	if err := os.MkdirAll(".bedrock", 0755); err != nil {
		return fmt.Errorf("failed to create .bedrock directory: %w", err)
	}

	// Open log file for daemon output
	logFile, err := os.OpenFile(".bedrock/ledger-daemon.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	// Start the daemon process
	cmd := exec.Command(executable, "_ledger-daemon",
		"--rpc-url", rpcURL,
		"--interval", strconv.Itoa(intervalMs),
	)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create new process group so it survives parent exit
	}

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Write PID file
	pidFile := filepath.Join(".bedrock", "ledger-daemon.pid")
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
		// Kill the process if we can't write PID file
		cmd.Process.Kill()
		logFile.Close()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Don't wait for the process - let it run in background
	go func() {
		cmd.Wait()
		logFile.Close()
	}()

	return nil
}

func nodeStop(ctx context.Context, manager *network.Manager) error {
	color.Cyan("Stopping local XRPL node\n")
	fmt.Println()

	// Stop the ledger daemon first
	if err := stopLedgerDaemon(); err != nil {
		color.Yellow("Warning: %v\n", err)
	} else {
		color.Green("✓ Ledger daemon stopped\n")
	}

	if err := manager.Stop(ctx); err != nil {
		color.Red("✗ Failed to stop node: %v\n", err)
		return err
	}

	fmt.Println()
	color.Green("✓ Local node stopped\n")

	return nil
}

// stopLedgerDaemon stops the ledger daemon process
func stopLedgerDaemon() error {
	pidFile := filepath.Join(".bedrock", "ledger-daemon.pid")

	// Read PID file
	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No daemon running
		}
		return fmt.Errorf("failed to read PID file: %w", err)
	}

	pid, err := strconv.Atoi(string(pidBytes))
	if err != nil {
		os.Remove(pidFile)
		return fmt.Errorf("invalid PID in file: %w", err)
	}

	// Find and kill the process
	process, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(pidFile)
		return nil // Process not found
	}

	// Send SIGTERM for graceful shutdown
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Process might already be dead
		os.Remove(pidFile)
		return nil
	}

	// Wait a moment for graceful shutdown
	time.Sleep(500 * time.Millisecond)

	// Clean up PID file
	os.Remove(pidFile)

	return nil
}

func nodeStatus(ctx context.Context, manager *network.Manager) error {
	color.Cyan("Checking local node status\n")
	fmt.Println()

	status, err := manager.Status(ctx)
	if err != nil {
		color.Red("✗ Failed to check status: %v\n", err)
		return err
	}

	if status.Running {
		// Resolve local network URLs from config
		cfg, _ := config.LoadFromWorkingDir()
		localCfg := config.NetworkConfig{
			URL:    "ws://localhost:6006",
			RPCURL: DefaultLocalRPCURL,
		}
		if cfg != nil {
			if lc, ok := cfg.Networks["local"]; ok {
				localCfg = lc
			}
		}

		color.Green("✓ Local node is running\n")
		fmt.Println()
		color.Cyan("Connection Details:\n")
		fmt.Printf("  WebSocket URL: %s\n", localCfg.URL)
		fmt.Printf("  RPC URL:       %s\n", localCfg.GetRPCURL())
		fmt.Println()

		// Show ledger daemon status
		color.Cyan("Ledger Daemon:\n")
		if isLedgerDaemonRunning() {
			color.Green("  Status: Running\n")
			fmt.Println("  Log:    .bedrock/ledger-daemon.log")
		} else {
			color.Yellow("  Status: Not running\n")
			fmt.Println("  Restart node to start ledger daemon")
		}
	} else {
		color.Yellow("⊙ Local node is not running\n")
		fmt.Println()
		color.White("  Start with: bedrock node start\n")
	}

	return nil
}

// isLedgerDaemonRunning checks if the ledger daemon process is running
func isLedgerDaemonRunning() bool {
	pidFile := filepath.Join(".bedrock", "ledger-daemon.pid")

	pidBytes, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(string(pidBytes))
	if err != nil {
		return false
	}

	// Check if process exists by sending signal 0
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Signal 0 doesn't actually send a signal, just checks if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func nodeLogs(manager *network.Manager) error {
	color.Cyan("Fetching local node logs (not yet implemented)\n")
	fmt.Println()

	color.Yellow("Log viewing coming soon. Use 'docker logs bedrock-xrpl-node' for now.\n")

	return nil
}
