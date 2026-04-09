package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/xrpl-commons/bedrock/pkg/config"
	"github.com/xrpl-commons/bedrock/pkg/primitives"
	"github.com/xrpl-commons/bedrock/pkg/templates"
)

var addTemplate string

var addCmd = &cobra.Command{
	Use:   "add <primitive>",
	Short: "Add a primitive to this project",
	Long: `Add a smart contract, escrow, or vault capability to an existing project.

This creates the necessary source directory, Cargo.toml, template code,
and updates bedrock.toml.

Examples:
  bedrock add escrow
  bedrock add vault --template vault-whitelist
  bedrock add contract --template token`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"contract", "escrow", "vault"},
	RunE:      runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringVarP(&addTemplate, "template", "t", "", "Template for the new primitive")
}

func runAdd(cmd *cobra.Command, args []string) error {
	p := args[0]

	if !primitives.IsValid(p) {
		return fmt.Errorf("unknown primitive %q (available: %s)", p, strings.Join(primitives.All(), ", "))
	}

	// Load existing config
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'bedrock init' first)", err)
	}

	// Check if already present
	if cfg.HasPrimitive(p) {
		return fmt.Errorf("primitive %q is already configured in this project", p)
	}

	def := primitives.Registry[p]

	// Resolve template
	tmplName := addTemplate
	if tmplName == "" {
		tmplName = templates.DefaultTemplate(p)
	}

	available := templates.AvailableForPrimitive(p)
	tmpl, ok := available[tmplName]
	if !ok {
		var names []string
		for k := range available {
			names = append(names, k)
		}
		return fmt.Errorf("unknown template %q for %s (available: %s)", tmplName, p, strings.Join(names, ", "))
	}

	color.Cyan("Adding %s to project...\n", def.DisplayName)
	fmt.Printf("  Template: %s - %s\n", tmpl.Name, tmpl.Description)

	// Create source directory
	srcDir := filepath.Join(def.SourceDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s: %w", srcDir, err)
	}

	// Write lib.rs
	libPath := filepath.Join(def.SourceDir, "src", "lib.rs")
	if err := os.WriteFile(libPath, []byte(tmpl.LibRS), 0644); err != nil {
		return fmt.Errorf("failed to create lib.rs: %w", err)
	}

	// Write Cargo.toml
	projectName := cfg.Project.Name
	cargoContent := templates.CargoToml(p, projectName+"-"+def.SourceDir)
	cargoPath := filepath.Join(def.SourceDir, "Cargo.toml")
	if err := os.WriteFile(cargoPath, []byte(cargoContent), 0644); err != nil {
		return fmt.Errorf("failed to create Cargo.toml: %w", err)
	}

	// Update bedrock.toml
	cfg.Primitives = append(cfg.Primitives, p)

	switch p {
	case primitives.Contract:
		if cfg.Contracts == nil {
			cfg.Contracts = make(map[string]config.ContractConfig)
		}
		cfg.Contracts["main"] = config.ContractConfig{
			Source: def.SourceDir + "/src/lib.rs",
			ABI:    def.SourceDir + "/build/abi.json",
		}
		if cfg.Build.Source == "" {
			cfg.Build = config.BuildConfig{
				Source: def.SourceDir + "/src/lib.rs",
				Output: def.SourceDir + "/target/wasm32-unknown-unknown/release",
				Target: "wasm32-unknown-unknown",
			}
		}
	case primitives.Escrow:
		if cfg.Escrows == nil {
			cfg.Escrows = make(map[string]config.EscrowConfig)
		}
		cfg.Escrows["main"] = config.EscrowConfig{
			Source: def.SourceDir + "/src/lib.rs",
			Output: def.SourceDir + "/target/wasm32v1-none/release",
		}
	case primitives.Vault:
		if cfg.Vaults == nil {
			cfg.Vaults = make(map[string]config.VaultConfig)
		}
		cfg.Vaults["main"] = config.VaultConfig{
			Source: def.SourceDir + "/src/lib.rs",
			Output: def.SourceDir + "/target/wasm32v1-none/release",
		}
	}

	// Update xrpld.cfg to unified config with all features
	xrpldPath := filepath.Join(".bedrock", "node-config", "xrpld.cfg")
	if err := os.WriteFile(xrpldPath, []byte(xrpldCfgTemplate), 0644); err != nil {
		color.Yellow("Warning: failed to update xrpld.cfg: %v\n", err)
	}
	cfg.LocalNode.DockerImage = def.DockerImage

	if err := config.Save(cfg, "bedrock.toml"); err != nil {
		return fmt.Errorf("failed to update bedrock.toml: %w", err)
	}

	color.Green("\n✓ %s added successfully!\n\n", def.DisplayName)

	// Next steps
	fmt.Println("Next steps:")
	switch p {
	case primitives.Contract:
		fmt.Println("  bedrock build --type contract")
		fmt.Println("  bedrock deploy --network local")
	case primitives.Escrow:
		fmt.Println("  bedrock build --type escrow")
		fmt.Println("  bedrock escrow deploy --destination <addr> --amount <drops> --wallet <seed> --network local")
	case primitives.Vault:
		fmt.Println("  bedrock build --type vault")
		fmt.Println("  bedrock vault deploy --asset XRP --wallet <seed> --network local")
	}

	return nil
}
