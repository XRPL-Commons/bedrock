package config

import (
	"runtime"
	"time"
)

// Config represents the bedrock.toml configuration
type Config struct {
	Project     ProjectConfig             `toml:"project"`
	Primitives  []string                  `toml:"primitives,omitempty"`
	Build       BuildConfig               `toml:"build"`
	Networks    map[string]NetworkConfig  `toml:"networks"`
	Contracts   map[string]ContractConfig `toml:"contracts"`
	Escrows     map[string]EscrowConfig   `toml:"escrows,omitempty"`
	Vaults      map[string]VaultConfig    `toml:"vaults,omitempty"`
	Deployments map[string]DeploymentInfo `toml:"deployments"`
	Wallets     WalletsConfig             `toml:"wallets"`
	LocalNode   LocalNodeConfig           `toml:"local_node"`
	Test        TestConfig                `toml:"test"`
	Snapshot    SnapshotConfig            `toml:"snapshot"`
	Doc         DocConfig                 `toml:"doc"`
}

// HasPrimitive returns true if the project includes the given primitive type
func (c *Config) HasPrimitive(name string) bool {
	for _, p := range c.Primitives {
		if p == name {
			return true
		}
	}
	return false
}

type ProjectConfig struct {
	Name    string   `toml:"name"`
	Version string   `toml:"version"`
	Authors []string `toml:"authors,omitempty"`
}

type BuildConfig struct {
	Source string `toml:"source"`
	Output string `toml:"output"`
	Target string `toml:"target"`
}

type NetworkConfig struct {
	URL       string `toml:"url"`
	NetworkID uint32 `toml:"network_id"`
	FaucetURL string `toml:"faucet_url,omitempty"`
	Explorer  string `toml:"explorer,omitempty"`
}

type ContractConfig struct {
	Source string `toml:"source"`
	ABI    string `toml:"abi"`
}

type DeploymentInfo struct {
	ContractAccount string    `toml:"contract_account"`
	ContractID      string    `toml:"contract_id"`
	TxHash          string    `toml:"tx_hash"`
	DeployedAt      time.Time `toml:"deployed_at"`
	Network         string    `toml:"network"`
}

type WalletsConfig struct {
	Default  string `toml:"default,omitempty"`
	Keystore string `toml:"keystore"`
}

// EscrowConfig represents a smart escrow source in the project
type EscrowConfig struct {
	Source string `toml:"source"`
	Output string `toml:"output,omitempty"`
}

// VaultConfig represents a smart vault source in the project
type VaultConfig struct {
	Source string `toml:"source"`
	Output string `toml:"output,omitempty"`
}

// LocalNodeConfig points to the directory containing rippled config files
type LocalNodeConfig struct {
	ConfigDir      string   `toml:"config_dir"`
	DockerImage    string   `toml:"docker_image"`
	Entrypoint     []string `toml:"entrypoint,omitempty"`
	Cmd            []string `toml:"cmd,omitempty"`
	Binds          []string `toml:"binds,omitempty"`
	LedgerInterval int      `toml:"ledger_interval"`
}

// TestConfig configures the test runner
type TestConfig struct {
	FuzzRuns           int    `toml:"fuzz_runs"`
	FuzzSeed           int64  `toml:"fuzz_seed"`
	IntegrationNetwork string `toml:"integration_network"`
	FixturesDir        string `toml:"fixtures_dir"`
}

// SnapshotConfig configures gas snapshots
type SnapshotConfig struct {
	File string `toml:"file"`
}

// DocConfig configures documentation generation
type DocConfig struct {
	Output string `toml:"output"`
}

const (
	// DefaultDockerImageAMD64 is the upstream amd64 image from Transia
	DefaultDockerImageAMD64 = "transia/cluster:f5d78179c9d1fbaf8bff8b77a052e263df90faa1"
	// DefaultDockerImageARM64 is a locally-built native arm64 image.
	// Build with: ./docker/build-arm64.sh
	DefaultDockerImageARM64 = "bedrock-xrpld:arm64-local"
)

// DefaultLocalNodeConfig returns default local node configuration.
// On arm64 (Apple Silicon), it defaults to the native arm64 image to avoid
// WASM execution hangs under Rosetta/QEMU emulation.
func DefaultLocalNodeConfig() LocalNodeConfig {
	image := DefaultDockerImageAMD64
	if runtime.GOARCH == "arm64" {
		image = DefaultDockerImageARM64
	}
	return LocalNodeConfig{
		ConfigDir:      ".bedrock/node-config",
		DockerImage:    image,
		LedgerInterval: 1000, // 1 second default
	}
}

// DefaultTestConfig returns default test configuration
func DefaultTestConfig() TestConfig {
	return TestConfig{
		FuzzRuns:           256,
		FuzzSeed:           0,
		IntegrationNetwork: "local",
		FixturesDir:        "tests/fixtures",
	}
}

// DefaultSnapshotConfig returns default snapshot configuration
func DefaultSnapshotConfig() SnapshotConfig {
	return SnapshotConfig{
		File: ".gas-snapshot",
	}
}

// DefaultDocConfig returns default documentation configuration
func DefaultDocConfig() DocConfig {
	return DocConfig{
		Output: "docs/api",
	}
}
