package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/xrpl-commons/bedrock/pkg/config"
	"github.com/xrpl-commons/bedrock/pkg/deployer"
	"github.com/xrpl-commons/bedrock/pkg/wallet"
)

var (
	clawbackNetwork   string
	clawbackWallet    string
	clawbackAmount    string
	clawbackAlgorithm string
	clawbackFee       string
)

var clawbackCmd = &cobra.Command{
	Use:   "clawback <contract-account>",
	Short: "Claw back tokens from a contract",
	Long: `Claw back tokens held by a contract via ContractClawback transaction.

This allows token issuers to reclaim tokens from a contract account.

Examples:
  bedrock clawback rContract123... --wallet sXXX... --amount "100/USD/rIssuer..."
  bedrock clawback rContract123... --wallet sXXX... --amount "1000000" --network local`,
	Args: cobra.ExactArgs(1),
	RunE: runClawback,
}

func init() {
	rootCmd.AddCommand(clawbackCmd)

	clawbackCmd.Flags().StringVarP(&clawbackNetwork, "network", "n", "alphanet", "Network")
	clawbackCmd.Flags().StringVarP(&clawbackWallet, "wallet", "w", "", "Wallet seed or name (required)")
	clawbackCmd.Flags().StringVar(&clawbackAmount, "amount", "", "Amount to claw back (e.g. '100/USD/rIssuer...' or drops)")
	clawbackCmd.Flags().StringVar(&clawbackAlgorithm, "algorithm", "secp256k1", "Cryptographic algorithm")
	clawbackCmd.Flags().StringVar(&clawbackFee, "fee", "1000000", "Transaction fee in drops")

	clawbackCmd.MarkFlagRequired("wallet")
	clawbackCmd.MarkFlagRequired("amount")
}

func runClawback(cmd *cobra.Command, args []string) error {
	contractAccount := args[0]

	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'bedrock init' first)", err)
	}

	networkCfg, ok := cfg.Networks[clawbackNetwork]
	if !ok {
		if clawbackNetwork == "local" {
			networkCfg = config.NetworkConfig{URL: "ws://localhost:6006", NetworkID: 100}
		} else {
			return fmt.Errorf("network '%s' not found in config", clawbackNetwork)
		}
	}

	color.Cyan("Clawing back tokens from contract\n")
	fmt.Printf("  Contract: %s\n", contractAccount)
	fmt.Printf("  Amount:   %s\n", clawbackAmount)
	fmt.Printf("  Network:  %s\n", clawbackNetwork)

	resolver, err := wallet.NewWalletResolver()
	if err != nil {
		return fmt.Errorf("failed to initialize wallet resolver: %w", err)
	}

	walletSeed, err := resolver.ResolveWallet(clawbackWallet)
	if err != nil {
		return fmt.Errorf("failed to resolve wallet: %w", err)
	}

	d, err := deployer.NewDeployer(false)
	if err != nil {
		return fmt.Errorf("failed to initialize deployer: %w", err)
	}

	ctx := cmd.Context()
	result, err := d.Clawback(ctx, deployer.ClawbackConfig{
		ContractAccount: contractAccount,
		Amount:          clawbackAmount,
		NetworkURL:      networkCfg.URL,
		WalletSeed:      walletSeed,
		Algorithm:       clawbackAlgorithm,
		Fee:             clawbackFee,
	})

	if err != nil {
		color.Red("\n  Clawback failed: %v\n", err)
		return err
	}

	color.Green("\n  Contract clawback successful!\n")
	fmt.Printf("  Transaction Hash: %s\n", result.TxHash)
	fmt.Printf("  Validated: %v\n", result.Validated)

	return nil
}
