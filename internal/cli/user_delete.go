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
	userDeleteNetwork   string
	userDeleteWallet    string
	userDeleteAlgorithm string
	userDeleteFee       string
)

var userDeleteCmd = &cobra.Command{
	Use:   "user-delete <contract-account>",
	Short: "Delete user data from a contract",
	Long: `Delete your data from a contract via ContractUserDelete transaction.

This recovers the reserves held for your contract data.

Examples:
  bedrock user-delete rContract123... --wallet sXXX...
  bedrock user-delete rContract123... --wallet sXXX... --network local`,
	Args: cobra.ExactArgs(1),
	RunE: runUserDelete,
}

func init() {
	rootCmd.AddCommand(userDeleteCmd)

	userDeleteCmd.Flags().StringVarP(&userDeleteNetwork, "network", "n", "alphanet", "Network")
	userDeleteCmd.Flags().StringVarP(&userDeleteWallet, "wallet", "w", "", "Wallet seed or name (required)")
	userDeleteCmd.Flags().StringVar(&userDeleteAlgorithm, "algorithm", "secp256k1", "Cryptographic algorithm")
	userDeleteCmd.Flags().StringVar(&userDeleteFee, "fee", "1000000", "Transaction fee in drops")

	userDeleteCmd.MarkFlagRequired("wallet")
}

func runUserDelete(cmd *cobra.Command, args []string) error {
	contractAccount := args[0]

	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'bedrock init' first)", err)
	}

	networkCfg, ok := cfg.Networks[userDeleteNetwork]
	if !ok {
		if userDeleteNetwork == "local" {
			networkCfg = config.NetworkConfig{URL: "ws://localhost:6006", NetworkID: 100}
		} else {
			return fmt.Errorf("network '%s' not found in config", userDeleteNetwork)
		}
	}

	color.Cyan("Deleting user data from contract\n")
	fmt.Printf("  Contract: %s\n", contractAccount)
	fmt.Printf("  Network:  %s\n", userDeleteNetwork)

	resolver, err := wallet.NewWalletResolver()
	if err != nil {
		return fmt.Errorf("failed to initialize wallet resolver: %w", err)
	}

	walletSeed, err := resolver.ResolveWallet(userDeleteWallet)
	if err != nil {
		return fmt.Errorf("failed to resolve wallet: %w", err)
	}

	d, err := deployer.NewDeployer(false)
	if err != nil {
		return fmt.Errorf("failed to initialize deployer: %w", err)
	}

	ctx := cmd.Context()
	result, err := d.UserDelete(ctx, deployer.UserDeleteConfig{
		ContractAccount: contractAccount,
		NetworkURL:      networkCfg.URL,
		WalletSeed:      walletSeed,
		Algorithm:       userDeleteAlgorithm,
		Fee:             userDeleteFee,
	})

	if err != nil {
		color.Red("\n✗ User data deletion failed: %v\n", err)
		return err
	}

	color.Green("\n✓ User data deleted successfully! Reserves recovered.\n")
	fmt.Printf("  Transaction Hash: %s\n", result.TxHash)
	fmt.Printf("  Validated: %v\n", result.Validated)

	return nil
}
