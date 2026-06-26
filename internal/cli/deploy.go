package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/xrpl-commons/bedrock/pkg/abi"
	"github.com/xrpl-commons/bedrock/pkg/builder"
	"github.com/xrpl-commons/bedrock/pkg/config"
	"github.com/xrpl-commons/bedrock/pkg/deployer"
	"github.com/xrpl-commons/bedrock/pkg/wallet"
)

var (
	deployNetwork       string
	deployWallet        string
	deployABI           string
	deploySkipBuild     bool
	deploySkipABI       bool
	deployAlgorithm     string
	deployImmutable     bool
	deployCodeImmutable bool
	deployABIImmutable  bool
	deployUndeletable   bool
	deployReuseCode     string
	deployParams        string
	deployOwner         string
	deployFee           string
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy smart contract",
	Long: `Deploy your smart contract to the network.

This command automatically:
1. Builds the contract in release mode (if needed)
2. Generates the ABI (if needed)
3. Deploys to the specified network

Use --skip-build or --skip-abi to skip these steps.`,
	RunE: runDeploy,
}

func init() {
	rootCmd.AddCommand(deployCmd)

	deployCmd.Flags().StringVarP(&deployNetwork, "network", "n", "alphanet", "Network to deploy to (local, alphanet, testnet, mainnet)")
	deployCmd.Flags().StringVarP(&deployWallet, "wallet", "w", "", "Wallet seed or name (generates new if not provided)")
	deployCmd.Flags().StringVarP(&deployABI, "abi", "a", "abi.json", "Path to ABI file")
	deployCmd.Flags().BoolVar(&deploySkipBuild, "skip-build", false, "Skip building the contract")
	deployCmd.Flags().BoolVar(&deploySkipABI, "skip-abi", false, "Skip generating ABI")
	deployCmd.Flags().StringVar(&deployAlgorithm, "algorithm", "secp256k1", "Cryptographic algorithm (secp256k1, ed25519)")
	deployCmd.Flags().BoolVar(&deployImmutable, "immutable", false, "Set lsfImmutable flag (no modifications allowed)")
	deployCmd.Flags().BoolVar(&deployCodeImmutable, "code-immutable", false, "Set lsfCodeImmutable flag (code cannot be changed)")
	deployCmd.Flags().BoolVar(&deployABIImmutable, "abi-immutable", false, "Set lsfABIImmutable flag (ABI cannot be changed)")
	deployCmd.Flags().BoolVar(&deployUndeletable, "undeletable", false, "Set lsfUndeletable flag (contract cannot be deleted)")
	deployCmd.Flags().StringVar(&deployReuseCode, "reuse-code", "", "Reference existing ContractSource by hash")
	deployCmd.Flags().StringVar(&deployParams, "params", "", "Instance parameter values as JSON")
	deployCmd.Flags().StringVar(&deployOwner, "owner", "", "Contract owner address (defaults to deployer)")
	deployCmd.Flags().StringVar(&deployFee, "fee", "", "Transaction fee in drops")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'bedrock init' first)", err)
	}

	if !cfg.HasPrimitive("contract") {
		return fmt.Errorf("no contract primitive in this project\nUse 'bedrock escrow deploy' or 'bedrock vault deploy' instead")
	}

	color.Cyan("Deploying smart contract\n")
	fmt.Printf("   Network: %s\n", deployNetwork)

	// Get network configuration
	networkCfg, ok := cfg.Networks[deployNetwork]
	if !ok {
		if deployNetwork == "local" {
			networkCfg = config.NetworkConfig{
				URL:       "ws://localhost:6006",
				NetworkID: 100,
				FaucetURL: "http://localhost:8080/faucet",
			}
		} else {
			return fmt.Errorf("network '%s' not found in config", deployNetwork)
		}
	}

	fmt.Printf("   URL: %s\n", networkCfg.URL)

	// Step 1: Build the contract (unless skipped)
	buildMode := "release"
	contractDir := filepath.Dir(filepath.Dir(cfg.Build.Source))
	wasmPath := filepath.Join(contractDir, "target", "wasm32-unknown-unknown", buildMode, cfg.Project.Name+".wasm")

	if !deploySkipBuild {
		// Check if WASM exists
		wasmExists := false
		if _, err := os.Stat(wasmPath); err == nil {
			wasmExists = true
		}

		if !wasmExists {
			fmt.Println()
			color.Yellow("→ Building contract (release mode)...\n")

			b := builder.New(".")
			ctx := cmd.Context()
			result, err := b.Build(ctx, builder.BuildOptions{
				Release: true,
				Verbose: false,
			})

			if err != nil {
				color.Red("\n✗ Build failed: %v\n", err)
				return err
			}

			wasmPath = result.WasmPath // Use the actual output path
			color.Green("✓ Build completed\n")
		} else {
			color.Green("✓ WASM file found (skipping build)\n")
		}
	} else {
		fmt.Println()
		color.Yellow("⊙ Skipping build (--skip-build)\n")
	}

	// Verify WASM exists
	if _, err := os.Stat(wasmPath); err != nil {
		return fmt.Errorf("WASM file not found: %s (build failed or --skip-build used incorrectly)", wasmPath)
	}

	fmt.Printf("   WASM: %s\n", wasmPath)

	// Step 2: Generate ABI (unless skipped)
	abiPath := deployABI

	if !deploySkipABI {
		// Check if ABI exists
		abiExists := false
		if _, err := os.Stat(abiPath); err == nil {
			abiExists = true
		}

		if !abiExists {
			fmt.Println()
			color.Yellow("→ Generating ABI...\n")

			// Parser needs the source directory (contract/src)
			sourceDir := filepath.Dir(cfg.Build.Source)
			parser := abi.NewParser(sourceDir)
			abiData, err := parser.ParseContract(cfg.Project.Name)
			if err != nil {
				color.Red("\n✗ ABI generation failed: %v\n", err)
				return err
			}

			// Generator needs the output directory
			outputDir := filepath.Dir(abiPath)
			generator := abi.NewGenerator(outputDir)
			_, err = generator.Generate(abiData, filepath.Base(abiPath))
			if err != nil {
				color.Red("\n✗ Failed to write ABI: %v\n", err)
				return err
			}

			color.Green("✓ ABI generated\n")
		} else {
			color.Green("✓ ABI file found (skipping generation)\n")
		}
	} else {
		fmt.Println()
		color.Yellow("⊙ Skipping ABI generation (--skip-abi)\n")
	}

	fmt.Printf("   ABI: %s\n", abiPath)

	// Step 3: Deploy
	fmt.Println()
	color.Yellow("→ Deploying to network...\n")

	// Determine faucet URL
	faucetURL := networkCfg.FaucetURL

	// Resolve wallet seed
	var walletSeed string
	if deployWallet != "" {
		resolver, err := wallet.NewWalletResolver()
		if err != nil {
			color.Red("✗ Failed to initialize wallet resolver: %v\n", err)
			return err
		}

		walletSeed, err = resolver.ResolveWallet(deployWallet)
		if err != nil {
			color.Red("✗ Failed to resolve wallet: %v\n", err)
			return err
		}

		color.White("   Wallet: Using provided wallet/seed\n")
	} else {
		color.White("   Wallet: Generating new wallet\n")
	}
	fmt.Println()

	// Create deployer
	verbose, _ := cmd.Flags().GetBool("verbose")
	d, err := deployer.NewDeployer(verbose)
	if err != nil {
		color.Red("✗ Failed to initialize deployer: %v\n", err)
		return err
	}

	// Deploy contract
	ctx := cmd.Context()
	result, err := d.Deploy(ctx, deployer.DeploymentConfig{
		WasmPath:      wasmPath,
		ABIPath:       abiPath,
		NetworkURL:    networkCfg.URL,
		NetworkID:     networkCfg.NetworkID,
		WalletSeed:    walletSeed,
		Algorithm:     deployAlgorithm,
		FaucetURL:     faucetURL,
		Fee:           deployFee,
		Immutable:     deployImmutable,
		CodeImmutable: deployCodeImmutable,
		ABIImmutable:  deployABIImmutable,
		Undeletable:   deployUndeletable,
		ReuseCode:     deployReuseCode,
		Params:        deployParams,
		Owner:         deployOwner,
	})

	if err != nil {
		color.Red("\n✗ Deployment failed: %v\n", err)
		return err
	}

	// Display results
	fmt.Println()
	color.Green("✓ Contract deployed successfully!\n")
	fmt.Println()
	color.Cyan("Deployment Details:\n")
	fmt.Printf("  Transaction Hash: %s\n", result.TxHash)

	// Highlight the contract account (most important info)
	if result.ContractAccount != "" {
		color.Green("  Contract Account: %s\n", result.ContractAccount)
	} else {
		color.Yellow("  Contract Account: (not found - check explorer)\n")
	}

	if result.ContractIndex != "" {
		fmt.Printf("  Contract Index: %s\n", result.ContractIndex)
	}

	fmt.Printf("  Wallet Address: %s\n", result.WalletAddress)
	fmt.Printf("  Wallet Seed: %s\n", maskSeed(result.WalletSeed))

	fmt.Println()
	color.Yellow("💡 Tips:\n")
	color.Yellow("   • Use --verbose to see the full wallet seed\n")
	color.Yellow("   • Save the wallet seed to interact with the contract later\n")
	if result.ContractAccount != "" {
		color.Yellow("   • Call functions with: bedrock call %s <function-name> --wallet <seed>\n", result.ContractAccount)
	}

	return nil
}

// maskSeed masks a wallet seed, showing only the first 4 and last 4 characters
func maskSeed(seed string) string {
	if len(seed) <= 8 {
		return "****"
	}
	return seed[:4] + strings.Repeat("*", len(seed)-8) + seed[len(seed)-4:]
}
