package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

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

// LoadFromWorkingDir loads bedrock.toml from the current working directory
func LoadFromWorkingDir() (*Config, error) {
	return Load("bedrock.toml")
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
