package cli

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/xrpl-commons/bedrock/pkg/templates"
)

var initTemplate string

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new XRPL smart contract project",
	Long: `Creates a new XRPL smart contract project with the necessary structure and configuration files.

Templates: basic, token, nft, escrow, counter

Examples:
  bedrock init my-project
  bedrock init my-project --template token
  bedrock init my-project --template nft`,
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

	initCmd.Flags().StringVarP(&initTemplate, "template", "t", "basic", "Project template (basic, token, nft, escrow, counter)")
}

func runInit(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	// Validate template
	available := templates.Available()
	tmpl, ok := available[initTemplate]
	if !ok {
		var names []string
		for k := range available {
			names = append(names, k)
		}
		return fmt.Errorf("unknown template '%s' (available: %s)", initTemplate, strings.Join(names, ", "))
	}

	color.Cyan("Initializing project: %s\n", projectName)
	if initTemplate != "basic" {
		fmt.Printf("  Template: %s - %s\n", tmpl.Name, tmpl.Description)
	}
	_ = tmpl

	// Create project directory
	if err := os.MkdirAll(projectName, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Create directory structure
	dirs := []string{
		filepath.Join(projectName, "contract", "src"),
		filepath.Join(projectName, ".wallets"),
		filepath.Join(projectName, ".bedrock", "node-config"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create bedrock.toml
	configContent := fmt.Sprintf(`[project]
name = "%s"
version = "0.1.0"
authors = ["Your Name"]

[build]
source = "contract/src/lib.rs"
output = "contract/target/wasm32-unknown-unknown/release"
target = "wasm32-unknown-unknown"

[local_node]
config_dir = ".bedrock/node-config"
docker_image = "transia/cluster:f5d78179c9d1fbaf8bff8b77a052e263df90faa1"
ledger_interval = 1000

[networks.local]
url = "ws://localhost:6006"
rpc_url = "http://localhost:5005"
network_id = 63456
faucet_url = "http://localhost:8080/faucet"

[networks.alphanet]
url = "wss://alphanet.nerdnest.xyz"
network_id = 21465
faucet_url = "https://alphanet.faucet.nerdnest.xyz/accounts"

[contracts.main]
source = "contract/src/lib.rs"
abi = "contract/build/abi.json"

[wallets]
keystore = ".wallets/keystore.json"
`, projectName)

	if err := os.WriteFile(filepath.Join(projectName, "bedrock.toml"), []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to create bedrock.toml: %w", err)
	}

	// Create Cargo.toml
	cargoContent := fmt.Sprintf(`[package]
name = "%s"
version = "0.1.0"
edition = "2021"

[lib]
crate-type = ["cdylib"]

[dependencies]
xrpl-wasm-std = { git = "https://github.com/Transia-RnD/craft.git", branch = "dangell/smart-contracts", package = "xrpl-wasm-std" }
xrpl-wasm-macros= { git = "https://github.com/Transia-RnD/craft.git", branch = "dangell/smart-contracts", package = "xrpl-wasm-macros" }

[profile.release]
opt-level = "z"
lto = true
strip = true
panic = "abort"
`, projectName)

	if err := os.WriteFile(filepath.Join(projectName, "contract", "Cargo.toml"), []byte(cargoContent), 0644); err != nil {
		return fmt.Errorf("failed to create Cargo.toml: %w", err)
	}

	// Create lib.rs from template
	libContent := tmpl.LibRS
	if err := os.WriteFile(filepath.Join(projectName, "contract", "src", "lib.rs"), []byte(libContent), 0644); err != nil {
		return fmt.Errorf("failed to create lib.rs: %w", err)
	}

	// Write embedded genesis.json template
	genesisPath := filepath.Join(projectName, ".bedrock", "node-config", "genesis.json")
	if err := os.WriteFile(genesisPath, []byte(genesisTemplate), 0644); err != nil {
		return fmt.Errorf("failed to create genesis.json: %w", err)
	}

	// Write embedded xrpld.cfg template
	xrpldCfgPath := filepath.Join(projectName, ".bedrock", "node-config", "xrpld.cfg")
	if err := os.WriteFile(xrpldCfgPath, []byte(xrpldCfgTemplate), 0644); err != nil {
		return fmt.Errorf("failed to create xrpld.cfg: %w", err)
	}

	// Write embedded validators.txt template
	validatorsPath := filepath.Join(projectName, ".bedrock", "node-config", "validators.txt")
	if err := os.WriteFile(validatorsPath, []byte(validatorsTemplate), 0644); err != nil {
		return fmt.Errorf("failed to create validators.txt: %w", err)
	}

	// Create .gitignore
	gitignoreContent := `# Build outputs
contract/target/
*.wasm

# Wallets (keep private!)
.wallets/

# Bedrock internal
.bedrock/

# OS files
.DS_Store
`

	if err := os.WriteFile(filepath.Join(projectName, ".gitignore"), []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	// Create README.md
	readmeContent := fmt.Sprintf("# %s\n\nXRPL Smart Contract project\n\n## Getting Started\n\n### Build the contract\n```bash\nbedrock build --release\n```\n\n### Start local node\n```bash\nbedrock node start\n```\n\n### Deploy\n```bash\nbedrock deploy --network local\n```\n\n## Project Structure\n\n- `contract/` - Smart contract source code\n- `bedrock.toml` - Project configuration\n- `.wallets/` - Local wallet storage (git-ignored)\n", projectName)

	if err := os.WriteFile(filepath.Join(projectName, "README.md"), []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to create README.md: %w", err)
	}

	// Success message
	color.Green("\n✓ Project initialized successfully!\n\n")
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", projectName)
	fmt.Println("  bedrock build --release  # Build your contract")
	fmt.Println("  bedrock node start       # Start local node")
	fmt.Println("  bedrock deploy --network local  # Deploy your contract")

	return nil
}
