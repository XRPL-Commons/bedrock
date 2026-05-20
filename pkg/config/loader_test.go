package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, configFileName)
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadAppliesLocalNodeDefaults(t *testing.T) {
	tmp := t.TempDir()
	path := writeConfig(t, tmp, `
[project]
name = "demo"
version = "0.1.0"

[networks.local]
url = "ws://localhost:6006"
network_id = 100
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.LocalNode.ConfigDir == "" {
		t.Error("expected ConfigDir default to be applied")
	}
	if cfg.LocalNode.DockerImage == "" {
		t.Error("expected DockerImage default to be applied")
	}
}

func TestLoadInfersPrimitivesFromLegacyConfig(t *testing.T) {
	tmp := t.TempDir()
	path := writeConfig(t, tmp, `
[project]
name = "demo"
version = "0.1.0"

[build]
source = "contract/src/lib.rs"

[contracts.main]
source = "contract/src/lib.rs"
abi = "contract/build/abi.json"

[escrows.main]
source = "escrow/src/lib.rs"

[vaults.main]
source = "vault/src/lib.rs"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := map[string]bool{"contract": true, "escrow": true, "vault": true}
	if len(cfg.Primitives) != len(want) {
		t.Errorf("Primitives length: got %d want %d (%v)", len(cfg.Primitives), len(want), cfg.Primitives)
	}
	for _, p := range cfg.Primitives {
		if !want[p] {
			t.Errorf("unexpected inferred primitive %q", p)
		}
	}
}

// TestLoadFromWorkingDirWalksUpward proves the upward search added to fix
// "running bedrock build from contract/ used to fail with a confusing
// 'config not found' error".
func TestLoadFromWorkingDirWalksUpward(t *testing.T) {
	tmp := t.TempDir()
	writeConfig(t, tmp, `
[project]
name = "demo"
version = "0.1.0"

[networks.local]
url = "ws://localhost:6006"
network_id = 100
`)
	sub := filepath.Join(tmp, "contract", "src")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}

	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromWorkingDir()
	if err != nil {
		t.Fatalf("LoadFromWorkingDir: %v", err)
	}
	if cfg.Project.Name != "demo" {
		t.Errorf("Project.Name: got %q want %q", cfg.Project.Name, "demo")
	}

	// LoadFromWorkingDir should also have chdir'd to the project root.
	cwd, _ := os.Getwd()
	// On macOS /var is a symlink to /private/var; compare resolved paths.
	wantRoot, _ := filepath.EvalSymlinks(tmp)
	gotRoot, _ := filepath.EvalSymlinks(cwd)
	if wantRoot != gotRoot {
		t.Errorf("cwd after Load: got %q want %q", gotRoot, wantRoot)
	}
}

func TestLoadFromWorkingDirNoConfigErr(t *testing.T) {
	tmp := t.TempDir()
	prev, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(prev) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFromWorkingDir(); err == nil {
		t.Fatal("expected error when no bedrock.toml exists anywhere upward")
	}
}

func TestHasPrimitive(t *testing.T) {
	c := &Config{Primitives: []string{"contract", "vault"}}
	if !c.HasPrimitive("contract") {
		t.Error("expected HasPrimitive(contract) true")
	}
	if c.HasPrimitive("escrow") {
		t.Error("expected HasPrimitive(escrow) false")
	}
}
