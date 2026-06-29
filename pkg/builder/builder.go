package builder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Builder handles contract compilation
type Builder struct {
	projectRoot string
}

// New creates a new Builder
func New(projectRoot string) *Builder {
	return &Builder{projectRoot: projectRoot}
}

// Build compiles Rust source to WASM
func (b *Builder) Build(ctx context.Context, opts BuildOptions) (*BuildResult, error) {
	// Verify cargo is installed
	if err := b.verifyToolchain(); err != nil {
		return nil, err
	}

	// Determine target and source dir (defaults for backward compat)
	target := opts.Target
	if target == "" {
		target = "wasm32-unknown-unknown"
	}
	sourceDir := opts.SourceDir
	if sourceDir == "" {
		sourceDir = "contract"
	}

	// Ensure WASM target is installed
	if err := b.ensureWasmTarget(ctx, target); err != nil {
		return nil, fmt.Errorf("failed to add %s target: %w", target, err)
	}

	buildDir := filepath.Join(b.projectRoot, sourceDir)

	// Check if Cargo.toml exists
	if _, err := os.Stat(filepath.Join(buildDir, "Cargo.toml")); os.IsNotExist(err) {
		return nil, fmt.Errorf("no Cargo.toml found in %s", buildDir)
	}

	// Build command
	args := []string{"build", "--target", target}
	if opts.Release {
		args = append(args, "--release")
	}
	if opts.Verbose {
		args = append(args, "--verbose")
	}

	cmd := exec.CommandContext(ctx, "cargo", args...)
	cmd.Dir = buildDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	startTime := time.Now()

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cargo build failed: %w", err)
	}

	duration := time.Since(startTime)

	// Determine output path
	buildType := "debug"
	if opts.Release {
		buildType = "release"
	}

	wasmPath := filepath.Join(buildDir, "target", target, buildType)

	// Find the .wasm file
	wasmFile, size, err := b.findWasmFile(wasmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find WASM output: %w", err)
	}

	return &BuildResult{
		WasmPath:  filepath.Join(wasmPath, wasmFile),
		Size:      size,
		Duration:  duration,
		Optimized: opts.Release,
	}, nil
}

// CleanDir removes build artifacts for a specific source directory
func (b *Builder) CleanDir(ctx context.Context, sourceDir string) error {
	dir := filepath.Join(b.projectRoot, sourceDir)

	if _, err := os.Stat(filepath.Join(dir, "Cargo.toml")); os.IsNotExist(err) {
		return nil // nothing to clean
	}

	cmd := exec.CommandContext(ctx, "cargo", "clean")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cargo clean failed in %s: %w", sourceDir, err)
	}

	return nil
}

// Clean removes build artifacts (backward compat -- cleans contract dir)
func (b *Builder) Clean(ctx context.Context) error {
	return b.CleanDir(ctx, "contract")
}

// verifyToolchain checks if cargo and rustc are installed
func (b *Builder) verifyToolchain() error {
	if _, err := exec.LookPath("cargo"); err != nil {
		return fmt.Errorf("cargo not found: please install Rust from https://rustup.rs")
	}
	if _, err := exec.LookPath("rustc"); err != nil {
		return fmt.Errorf("rustc not found: please install Rust from https://rustup.rs")
	}
	return nil
}

// ensureWasmTarget adds the given WASM target if not present
func (b *Builder) ensureWasmTarget(ctx context.Context, target string) error {
	cmd := exec.CommandContext(ctx, "rustup", "target", "add", target)
	if err := cmd.Run(); err != nil {
		// Not fatal if rustup fails (target might already be installed)
		return nil
	}
	return nil
}

// findWasmFile finds the first .wasm file in the directory
func (b *Builder) findWasmFile(dir string) (string, int64, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", 0, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".wasm" {
			info, err := entry.Info()
			if err != nil {
				return "", 0, err
			}
			return entry.Name(), info.Size(), nil
		}
	}

	return "", 0, fmt.Errorf("no .wasm file found in %s", dir)
}
