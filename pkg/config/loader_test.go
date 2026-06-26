package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTOML writes content to a temp bedrock.toml and returns its path.
func writeTOML(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bedrock.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	return path
}

func TestLoadParsesProjectAndNetworks(t *testing.T) {
	path := writeTOML(t, `
[project]
name = "demo"
version = "0.1.0"

primitives = ["contract"]

[networks.local]
url = "ws://localhost:6006"
network_id = 100
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Project.Name != "demo" {
		t.Errorf("Project.Name = %q, want demo", cfg.Project.Name)
	}
	net, ok := cfg.Networks["local"]
	if !ok {
		t.Fatal("networks.local missing")
	}
	if net.URL != "ws://localhost:6006" {
		t.Errorf("local.URL = %q", net.URL)
	}
	if net.NetworkID != 100 {
		t.Errorf("local.NetworkID = %d, want 100", net.NetworkID)
	}
}

// Load fills in local-node defaults when the section is absent.
func TestLoadAppliesLocalNodeDefaults(t *testing.T) {
	path := writeTOML(t, `
[project]
name = "demo"
primitives = ["contract"]
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	def := DefaultLocalNodeConfig()
	if cfg.LocalNode.ConfigDir != def.ConfigDir {
		t.Errorf("LocalNode.ConfigDir = %q, want default %q", cfg.LocalNode.ConfigDir, def.ConfigDir)
	}
	if cfg.LocalNode.DockerImage != def.DockerImage {
		t.Errorf("LocalNode.DockerImage = %q, want default %q", cfg.LocalNode.DockerImage, def.DockerImage)
	}
}

// When `primitives` is omitted, Load infers it from the present sections
// (backward compatibility with older bedrock.toml files).
func TestLoadInfersPrimitivesFromSections(t *testing.T) {
	path := writeTOML(t, `
[project]
name = "legacy"

[contracts.main]
source = "contract/src/lib.rs"
abi = "abi.json"

[escrows.main]
source = "escrow/src/lib.rs"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if !cfg.HasPrimitive("contract") {
		t.Error("expected inferred primitive 'contract' from [contracts.main]")
	}
	if !cfg.HasPrimitive("escrow") {
		t.Error("expected inferred primitive 'escrow' from [escrows.main]")
	}
	if cfg.HasPrimitive("vault") {
		t.Error("did not expect 'vault' to be inferred")
	}
}

func TestLoadMissingFileErrors(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "does-not-exist.toml")); err == nil {
		t.Fatal("Load() of a missing file should return an error")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	original := &Config{
		Project:    ProjectConfig{Name: "rt", Version: "1.2.3"},
		Primitives: []string{"contract", "vault"},
		Networks: map[string]NetworkConfig{
			"local": {URL: "ws://localhost:6006", NetworkID: 100},
		},
	}

	path := filepath.Join(t.TempDir(), "bedrock.toml")
	if err := Save(original, path); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.Project.Name != "rt" || loaded.Project.Version != "1.2.3" {
		t.Errorf("project mismatch after round trip: %+v", loaded.Project)
	}
	if !loaded.HasPrimitive("contract") || !loaded.HasPrimitive("vault") {
		t.Errorf("primitives lost in round trip: %v", loaded.Primitives)
	}
	if loaded.Networks["local"].NetworkID != 100 {
		t.Errorf("network_id lost in round trip: %d", loaded.Networks["local"].NetworkID)
	}
}

func TestHasPrimitive(t *testing.T) {
	cfg := &Config{Primitives: []string{"contract", "escrow"}}
	if !cfg.HasPrimitive("contract") {
		t.Error("HasPrimitive(contract) = false, want true")
	}
	if cfg.HasPrimitive("vault") {
		t.Error("HasPrimitive(vault) = true, want false")
	}
}
