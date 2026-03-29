package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/xrpl-commons/bedrock/pkg/builder"
	"github.com/xrpl-commons/bedrock/pkg/config"
	"github.com/xrpl-commons/bedrock/pkg/primitives"
)

var (
	buildRelease bool
	buildWatch   bool
	buildType    string
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build project WASM binaries",
	Long: `Build the project's Rust source to WASM. Defaults to release mode.

By default, builds all primitives in the project. Use --type to build a specific one.

Examples:
  bedrock build                    # Build all primitives
  bedrock build --type contract    # Build only contract
  bedrock build --type escrow      # Build only escrow
  bedrock build --type vault       # Build only vault`,
	RunE: runBuild,
}

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().BoolVarP(&buildRelease, "release", "r", true, "Build in release mode (optimized)")
	buildCmd.Flags().BoolVarP(&buildWatch, "watch", "w", false, "Watch for changes and rebuild")
	buildCmd.Flags().StringVar(&buildType, "type", "", "Build specific primitive (contract, escrow, vault)")
}

func runBuild(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'bedrock init' first)", err)
	}

	if buildWatch {
		color.Yellow("Watch mode not yet implemented\n")
		return fmt.Errorf("watch mode coming soon")
	}

	// Determine which primitives to build
	toBuild := cfg.Primitives
	if buildType != "" {
		if !primitives.IsValid(buildType) {
			return fmt.Errorf("unknown primitive type %q (available: contract, escrow, vault)", buildType)
		}
		if !cfg.HasPrimitive(buildType) {
			return fmt.Errorf("primitive %q is not configured in this project (configured: %v)", buildType, cfg.Primitives)
		}
		toBuild = []string{buildType}
	}

	if len(toBuild) == 0 {
		return fmt.Errorf("no primitives configured (run 'bedrock init' first)")
	}

	b := builder.New(".")
	ctx := cmd.Context()

	for _, p := range toBuild {
		def := primitives.Registry[p]

		mode := "debug"
		if buildRelease {
			mode = "release"
		}

		color.Cyan("Building %s\n", def.DisplayName)
		fmt.Printf("   Mode: %s\n", mode)
		fmt.Printf("   Target: %s\n", def.WasmTarget)
		fmt.Printf("   Source: %s/\n", def.SourceDir)
		fmt.Println()

		result, err := b.Build(ctx, builder.BuildOptions{
			Release:   buildRelease,
			Verbose:   false,
			Target:    def.WasmTarget,
			SourceDir: def.SourceDir,
		})

		if err != nil {
			color.Red("\n✗ Build failed for %s: %v\n", def.DisplayName, err)
			return err
		}

		color.Green("✓ %s built successfully!\n", def.DisplayName)
		fmt.Printf("   Output: %s\n", result.WasmPath)
		fmt.Printf("   Size: %d bytes\n", result.Size)
		fmt.Printf("   Duration: %v\n", result.Duration)
		fmt.Println()
	}

	return nil
}
