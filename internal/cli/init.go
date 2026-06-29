package cli

import (
	_ "embed"
	"bufio"
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

var (
	initTemplate   string
	initPrimitives string
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new XRPL project",
	Long: `Creates a new XRPL project with the necessary structure and configuration files.

Primitives: contract, escrow, vault

Examples:
  bedrock init my-project
  bedrock init my-project --primitives escrow
  bedrock init my-project -p vault
  bedrock init my-project -p vault --template vault-whitelist`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

//go:embed templates/genesis.json
var genesisTemplate string

//go:embed templates/xrpld.cfg
var xrpldCfgTemplate string

//go:embed templates/validators.txt
var validatorsTemplate string

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVarP(&initTemplate, "template", "t", "", "Project template (use --help to see available templates per primitive)")
	initCmd.Flags().StringVarP(&initPrimitives, "primitives", "p", "", "Comma-separated primitives (contract,escrow,vault)")
}

func runInit(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	// Determine primitives
	selectedPrimitives, err := resolvePrimitives()
	if err != nil {
		return err
	}

	color.Cyan("Initializing project: %s\n", projectName)
	for _, p := range selectedPrimitives {
		def := primitives.Registry[p]
		fmt.Printf("  Primitive: %s - %s\n", def.DisplayName, def.Description)
	}

	// Create project directory
	if err := os.MkdirAll(projectName, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Create common directories
	commonDirs := []string{
		filepath.Join(projectName, ".wallets"),
		filepath.Join(projectName, ".bedrock", "node-config"),
	}
	for _, dir := range commonDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Scaffold each primitive
	for _, p := range selectedPrimitives {
		if err := scaffoldPrimitive(projectName, p); err != nil {
			return err
		}
	}

	// Generate bedrock.toml
	if err := generateConfig(projectName, selectedPrimitives); err != nil {
		return err
	}

	// Write node config templates
	if err := writeNodeConfig(projectName, selectedPrimitives); err != nil {
		return err
	}

	// Create .gitignore
	if err := writeGitignore(projectName, selectedPrimitives); err != nil {
		return err
	}

	// Create README.md
	if err := writeReadme(projectName, selectedPrimitives); err != nil {
		return err
	}

	// Success message
	color.Green("\n✓ Project initialized successfully!\n\n")
	printNextSteps(projectName, selectedPrimitives)

	return nil
}

func resolvePrimitives() ([]string, error) {
	if initPrimitives != "" {
		p := strings.TrimSpace(initPrimitives)
		if !primitives.IsValid(p) {
			return nil, fmt.Errorf("unknown primitive %q (available: %s)", p, strings.Join(primitives.All(), ", "))
		}
		return []string{p}, nil
	}

	// Check if stdin is a terminal
	fi, _ := os.Stdin.Stat()
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		// Non-interactive: default to contract
		return []string{primitives.Contract}, nil
	}

	// Interactive menu
	fmt.Println("\nSelect a primitive:")
	fmt.Println()
	for i, kind := range primitives.All() {
		def := primitives.Registry[kind]
		fmt.Printf("  %d. %-16s - %s\n", i+1, def.DisplayName, def.Description)
	}
	fmt.Println()
	fmt.Print("Choice [1]: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())

	if input == "" {
		return []string{primitives.Contract}, nil
	}

	allKinds := primitives.All()
	idx := 0
	if _, err := fmt.Sscanf(input, "%d", &idx); err != nil || idx < 1 || idx > len(allKinds) {
		return nil, fmt.Errorf("invalid selection %q (enter a number 1-%d)", input, len(allKinds))
	}

	return []string{allKinds[idx-1]}, nil
}

func scaffoldPrimitive(projectName, p string) error {
	def := primitives.Registry[p]

	// Create source directories
	srcDir := filepath.Join(projectName, def.SourceDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", def.SourceDir, err)
	}

	// Resolve template
	tmplName := initTemplate
	if tmplName == "" {
		tmplName = templates.DefaultTemplate(p)
	}

	available := templates.AvailableForPrimitive(p)
	tmpl, ok := available[tmplName]
	if !ok {
		// If user specified a template for a different primitive, try the default
		tmpl, ok = available[templates.DefaultTemplate(p)]
		if !ok {
			var names []string
			for k := range available {
				names = append(names, k)
			}
			return fmt.Errorf("no template %q for %s (available: %s)", tmplName, p, strings.Join(names, ", "))
		}
	}

	fmt.Printf("  Template: %s - %s\n", tmpl.Name, tmpl.Description)

	// Write lib.rs
	libPath := filepath.Join(projectName, def.SourceDir, "src", "lib.rs")
	if err := os.WriteFile(libPath, []byte(tmpl.LibRS), 0644); err != nil {
		return fmt.Errorf("failed to create lib.rs for %s: %w", p, err)
	}

	// Write Cargo.toml
	cargoPath := filepath.Join(projectName, def.SourceDir, "Cargo.toml")
	cargoContent := templates.CargoToml(p, projectName+"-"+def.SourceDir)
	if err := os.WriteFile(cargoPath, []byte(cargoContent), 0644); err != nil {
		return fmt.Errorf("failed to create Cargo.toml for %s: %w", p, err)
	}

	return nil
}

func generateConfig(projectName string, selected []string) error {
	var buf strings.Builder

	// `primitives` is a root-level key in bedrock.toml: it must precede the
	// first table header. Emitting it under [project] would make TOML parse it
	// as `project.primitives`, which Config never reads (cfg.Primitives loads
	// empty and is only repopulated by the section-inference fallback in Load).
	buf.WriteString("primitives = [")
	for i, p := range selected {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf(`"%s"`, p))
	}
	buf.WriteString("]\n\n")

	buf.WriteString(fmt.Sprintf(`[project]
name = "%s"
version = "0.1.0"
authors = ["Your Name"]

`, projectName))

	// Build section (only for contract)
	for _, p := range selected {
		if p == primitives.Contract {
			buf.WriteString(`[build]
source = "contract/src/lib.rs"
output = "contract/target/wasm32-unknown-unknown/release"
target = "wasm32-unknown-unknown"

`)
			break
		}
	}

	// Use arch-aware default image (x86 for amd64, arm64 for Apple Silicon)
	dockerImage := config.DefaultLocalNodeConfig().DockerImage

	buf.WriteString(fmt.Sprintf(`[local_node]
config_dir = ".bedrock/node-config"
docker_image = "%s"
ledger_interval = 1000

`, dockerImage))

	buf.WriteString(`[networks.local]
url = "ws://localhost:6006"
network_id = 100
faucet_url = "http://localhost:8080/faucet"

[networks.alphanet]
url = "wss://alphanet.nerdnest.xyz"
network_id = 21465
faucet_url = "https://alphanet.faucet.nerdnest.xyz/accounts"

`)

	// Per-primitive config sections
	for _, p := range selected {
		def := primitives.Registry[p]
		switch p {
		case primitives.Contract:
			buf.WriteString(fmt.Sprintf(`[contracts.main]
source = "%s/src/lib.rs"
abi = "%s/build/abi.json"

`, def.SourceDir, def.SourceDir))
		case primitives.Escrow:
			buf.WriteString(fmt.Sprintf(`[escrows.main]
source = "%s/src/lib.rs"
output = "%s/target/wasm32v1-none/release"

`, def.SourceDir, def.SourceDir))
		case primitives.Vault:
			buf.WriteString(fmt.Sprintf(`[vaults.main]
source = "%s/src/lib.rs"
output = "%s/target/wasm32v1-none/release"

`, def.SourceDir, def.SourceDir))
		}
	}

	buf.WriteString(`[wallets]
keystore = ".wallets/keystore.json"
`)

	configPath := filepath.Join(projectName, "bedrock.toml")
	return os.WriteFile(configPath, []byte(buf.String()), 0644)
}

func writeNodeConfig(projectName string, selected []string) error {
	// Write unified xrpld.cfg (supports all features: contracts, escrows, vaults)
	configs := map[string]string{
		"xrpld.cfg": xrpldCfgTemplate,
	}

	// Contract projects also need genesis.json for ledger initialization
	hasContract := false
	for _, p := range selected {
		if p == primitives.Contract {
			hasContract = true
			break
		}
	}
	if hasContract {
		configs["genesis.json"] = genesisTemplate
	}

	for name, content := range configs {
		path := filepath.Join(projectName, ".bedrock", "node-config", name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create %s: %w", name, err)
		}
	}

	return nil
}

func writeGitignore(projectName string, selected []string) error {
	var buf strings.Builder
	buf.WriteString("# Build outputs\n")
	for _, p := range selected {
		def := primitives.Registry[p]
		buf.WriteString(fmt.Sprintf("%s/target/\n", def.SourceDir))
	}
	buf.WriteString("*.wasm\n\n")
	buf.WriteString(`# Wallets (keep private!)
.wallets/

# Bedrock internal
.bedrock/

# OS files
.DS_Store
`)
	path := filepath.Join(projectName, ".gitignore")
	return os.WriteFile(path, []byte(buf.String()), 0644)
}

func writeReadme(projectName string, selected []string) error {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("# %s\n\n", projectName))

	primNames := make([]string, len(selected))
	for i, p := range selected {
		primNames[i] = primitives.Registry[p].DisplayName
	}
	buf.WriteString(fmt.Sprintf("XRPL %s project\n\n", strings.Join(primNames, " + ")))

	buf.WriteString("## Getting Started\n\n")

	for _, p := range selected {
		def := primitives.Registry[p]
		buf.WriteString(fmt.Sprintf("### %s\n\n", def.DisplayName))

		switch p {
		case primitives.Contract:
			buf.WriteString("```bash\nbedrock build          # Build contract\nbedrock deploy --network local  # Deploy\nbedrock call <addr> <fn> --wallet <seed> --network local\n```\n\n")
		case primitives.Escrow:
			buf.WriteString("```bash\nbedrock build --type escrow  # Build escrow WASM\nbedrock escrow deploy --destination <addr> --amount <drops> --wallet <seed> --network local\nbedrock escrow finish <owner> <seq> --wallet <seed> --network local\n```\n\n")
		case primitives.Vault:
			buf.WriteString("```bash\nbedrock build --type vault  # Build vault WASM\nbedrock vault deploy --asset XRP --wallet <seed> --network local\nbedrock vault deposit <vault-id> --amount <drops> --wallet <seed> --network local\nbedrock vault withdraw <vault-id> --amount <drops> --wallet <seed> --network local\n```\n\n")
		}
	}

	buf.WriteString("## Project Structure\n\n")
	for _, p := range selected {
		def := primitives.Registry[p]
		buf.WriteString(fmt.Sprintf("- `%s/` - %s source code\n", def.SourceDir, def.DisplayName))
	}
	buf.WriteString("- `bedrock.toml` - Project configuration\n")
	buf.WriteString("- `.wallets/` - Local wallet storage (git-ignored)\n")

	path := filepath.Join(projectName, "README.md")
	return os.WriteFile(path, []byte(buf.String()), 0644)
}

func printNextSteps(projectName string, selected []string) {
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", projectName)

	hasContract := false
	hasEscrow := false
	hasVault := false
	for _, p := range selected {
		switch p {
		case primitives.Contract:
			hasContract = true
		case primitives.Escrow:
			hasEscrow = true
		case primitives.Vault:
			hasVault = true
		}
	}

	if hasContract {
		fmt.Println("  bedrock build            # Build contract")
	}
	if hasEscrow {
		fmt.Println("  bedrock build --type escrow   # Build escrow WASM")
	}
	if hasVault {
		fmt.Println("  bedrock build --type vault    # Build vault WASM")
	}
	fmt.Println("  bedrock node start       # Start local node")
	if hasContract {
		fmt.Println("  bedrock deploy --network local")
	}
	if hasEscrow {
		fmt.Println("  bedrock escrow deploy --destination <addr> --amount <drops> --wallet <seed> --network local")
	}
	if hasVault {
		fmt.Println("  bedrock vault deploy --asset XRP --wallet <seed> --network local")
	}
}
