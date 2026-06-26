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
	deleteNetwork   string
	deleteWallet    string
	deleteAlgorithm string
	deleteFee       string
)

var deleteCmd = &cobra.Command{
	Use:   "delete <contract-account>",
	Short: "Delete a deployed contract",
	Long: `Delete a deployed contract via ContractDelete transaction.

The contract must not have the lsfUndeletable flag set.

Examples:
  bedrock delete rContract123... --wallet sXXX...
  bedrock delete rContract123... --wallet sXXX... --network local`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().StringVarP(&deleteNetwork, "network", "n", "alphanet", "Network")
	deleteCmd.Flags().StringVarP(&deleteWallet, "wallet", "w", "", "Wallet seed or name (required)")
	deleteCmd.Flags().StringVar(&deleteAlgorithm, "algorithm", "secp256k1", "Cryptographic algorithm")
	deleteCmd.Flags().StringVar(&deleteFee, "fee", "1000000", "Transaction fee in drops")

	deleteCmd.MarkFlagRequired("wallet")
}

func runDelete(cmd *cobra.Command, args []string) error {
	contractAccount := args[0]

	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'bedrock init' first)", err)
	}

	networkCfg, ok := cfg.Networks[deleteNetwork]
	if !ok {
		if deleteNetwork == "local" {
			networkCfg = config.NetworkConfig{URL: "ws://localhost:6006", NetworkID: 100}
		} else {
			return fmt.Errorf("network '%s' not found in config", deleteNetwork)
		}
	}

	color.Cyan("Deleting contract\n")
	fmt.Printf("  Contract: %s\n", contractAccount)
	fmt.Printf("  Network:  %s\n", deleteNetwork)

	resolver, err := wallet.NewWalletResolver()
	if err != nil {
		return fmt.Errorf("failed to initialize wallet resolver: %w", err)
	}

	walletSeed, err := resolver.ResolveWallet(deleteWallet)
	if err != nil {
		return fmt.Errorf("failed to resolve wallet: %w", err)
	}

	verbose, _ := cmd.Flags().GetBool("verbose")
	d, err := deployer.NewDeployer(verbose)
	if err != nil {
		return fmt.Errorf("failed to initialize deployer: %w", err)
	}

	ctx := cmd.Context()
	result, err := d.Delete(ctx, deployer.DeleteConfig{
		ContractAccount: contractAccount,
		NetworkURL:      networkCfg.URL,
		WalletSeed:      walletSeed,
		Algorithm:       deleteAlgorithm,
		Fee:             deleteFee,
	})

	if err != nil {
		color.Red("\n✗ Deletion failed: %v\n", err)
		return err
	}

	color.Green("\n✓ Contract deleted successfully!\n")
	fmt.Printf("  Transaction Hash: %s\n", result.TxHash)
	fmt.Printf("  Validated: %v\n", result.Validated)

	return nil
}
