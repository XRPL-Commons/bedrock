package builder

import "time"

// BuildOptions configures the build process
type BuildOptions struct {
	Release   bool   // Use --release flag
	Verbose   bool   // Show verbose output
	Target    string // WASM target (e.g. "wasm32-unknown-unknown" or "wasm32v1-none")
	SourceDir string // Source directory (e.g. "contract", "escrow", "vault")
}

// BuildResult contains information about the build
type BuildResult struct {
	WasmPath  string        // Path to the compiled WASM file
	Size      int64         // Size in bytes
	Duration  time.Duration // Build time
	Optimized bool          // Whether release optimizations were applied
}
