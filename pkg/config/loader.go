package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// configFileName is the project manifest filename.
const configFileName = "bedrock.toml"

// Load reads the bedrock.toml configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply defaults for local node if not specified
	if cfg.LocalNode.ConfigDir == "" {
		cfg.LocalNode.ConfigDir = DefaultLocalNodeConfig().ConfigDir
	}
	if cfg.LocalNode.DockerImage == "" {
		cfg.LocalNode.DockerImage = DefaultLocalNodeConfig().DockerImage
	}

	// Backward compatibility: infer primitives from existing config sections
	if len(cfg.Primitives) == 0 {
		if len(cfg.Contracts) > 0 || cfg.Build.Source != "" {
			cfg.Primitives = append(cfg.Primitives, "contract")
		}
		if len(cfg.Escrows) > 0 {
			cfg.Primitives = append(cfg.Primitives, "escrow")
		}
		if len(cfg.Vaults) > 0 {
			cfg.Primitives = append(cfg.Primitives, "vault")
		}
	}

	return &cfg, nil
}

// LoadFromWorkingDir loads bedrock.toml by walking upward from the current
// working directory until the project root is found. This lets users invoke
// bedrock from within a subdirectory (e.g. contract/) the same way cargo and
// git work.
func LoadFromWorkingDir() (*Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	path, err := findConfigUpward(cwd)
	if err != nil {
		return nil, err
	}

	// Chdir to project root so relative paths in bedrock.toml resolve as
	// expected by the rest of the CLI (which assumes CWD == project root).
	root := filepath.Dir(path)
	if root != cwd {
		if err := os.Chdir(root); err != nil {
			return nil, fmt.Errorf("failed to chdir to project root %s: %w", root, err)
		}
	}

	return Load(configFileName)
}

// findConfigUpward walks from start toward the filesystem root looking for
// bedrock.toml. Returns an error mentioning the original directory so the
// user knows the search scope.
func findConfigUpward(start string) (string, error) {
	dir := start
	for {
		candidate := filepath.Join(dir, configFileName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s found in %s or any parent directory; run `bedrock init <name>` to create a project", configFileName, start)
		}
		dir = parent
	}
}

// Save writes the configuration to a file
func Save(cfg *Config, path string) error {
	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
