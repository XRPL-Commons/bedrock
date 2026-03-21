package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/xrpl-commons/bedrock/pkg/config"
	"github.com/xrpl-commons/bedrock/pkg/faucet"
)

var (
	faucetNetwork   string
	faucetWallet    string
	faucetAddress   string
	faucetAlgorithm string
)

var faucetCmd = &cobra.Command{
	Use:   "faucet",
	Short: "Request funds from faucet",
	Long: `Request testnet funds from the XRPL faucet.

You can provide:
- A wallet seed with --wallet
- An address with --address
- Nothing (generates a new wallet)

The faucet will send testnet XRP to the specified or generated address.`,
	RunE: runFaucet,
}

func init() {
	rootCmd.AddCommand(faucetCmd)

	faucetCmd.Flags().StringVarP(&faucetNetwork, "network", "n", "alphanet", "Network to use (local, alphanet, testnet, mainnet)")
	faucetCmd.Flags().StringVarP(&faucetWallet, "wallet", "w", "", "Wallet seed (optional)")
	faucetCmd.Flags().StringVarP(&faucetAddress, "address", "a", "", "Wallet address (optional)")
	faucetCmd.Flags().StringVar(&faucetAlgorithm, "algorithm", "secp256k1", "Cryptographic algorithm (secp256k1, ed25519)")
}

func runFaucet(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'bedrock init' first)", err)
	}

	color.Cyan("Requesting funds from faucet\n")
	fmt.Printf("   Network: %s\n", faucetNetwork)

	// Get network configuration
	networkCfg, ok := cfg.Networks[faucetNetwork]
	isLocal := faucetNetwork == "local"
	if !ok {
		if isLocal {
			networkCfg = config.NetworkConfig{
				URL:       "ws://localhost:6006",
				RPCURL:    "http://localhost:5005",
				NetworkID: 0, // Local network uses network ID 0
			}
		} else {
			return fmt.Errorf("network '%s' not found in config", faucetNetwork)
		}
	}

	if !isLocal && networkCfg.FaucetURL == "" {
		return fmt.Errorf("no faucet URL configured for network '%s'", faucetNetwork)
	}

	if isLocal {
		fmt.Printf("   Using genesis account for local funding\n")
	} else {
		fmt.Printf("   Faucet URL: %s\n", networkCfg.FaucetURL)
	}

	// Validate flags
	if faucetWallet != "" && faucetAddress != "" {
		return fmt.Errorf("cannot specify both --wallet and --address")
	}

	// Determine what we're doing
	if faucetWallet != "" {
		color.White("   Using provided wallet seed\n")
	} else if faucetAddress != "" {
		color.White("   Using provided address: %s\n", faucetAddress)
	} else {
		color.White("   Generating new wallet\n")
	}

	fmt.Println()

	// Create faucet client
	verbose, _ := cmd.Flags().GetBool("verbose")
	f, err := faucet.NewFaucet(verbose)
	if err != nil {
		color.Red("✗ Failed to initialize faucet: %v\n", err)
		return err
	}

	// Request from faucet
	ctx := cmd.Context()
	result, err := f.Request(ctx, faucet.FaucetConfig{
		FaucetURL:     networkCfg.FaucetURL,
		WalletSeed:    faucetWallet,
		WalletAddress: faucetAddress,
		Algorithm:     faucetAlgorithm,
		NetworkURL:    networkCfg.URL,
		IsLocal:       isLocal,
	})

	if err != nil {
		color.Red("✗ Faucet request failed: %v\n", err)
		return err
	}

	// Display results
	fmt.Println()
	color.Green("✓ Funds received successfully!\n")
	fmt.Println()
	color.Cyan("Details:\n")
	fmt.Printf("  Wallet Address: %s\n", result.WalletAddress)

	if result.WalletSeed != "" {
		color.Yellow("  Wallet Seed: %s\n", result.WalletSeed)
	}

	if result.Balance != "" {
		fmt.Printf("  Balance: %s XRP\n", result.Balance)
	}

	if result.FaucetAmount != "" && result.FaucetAmount != "unknown" {
		fmt.Printf("  Faucet Amount: %s XRP\n", result.FaucetAmount)
	}

	fmt.Printf("  Transaction Hash: %s\n", result.TxHash)

	if result.WalletSeed != "" {
		fmt.Println()
		color.Yellow("💡 Tips:\n")
		color.Yellow("   • Save this wallet seed securely to use for deployments and contract calls\n")
		color.Yellow("   • Use --wallet <your-seed> with deploy and call commands\n")
	}

	return nil
}
