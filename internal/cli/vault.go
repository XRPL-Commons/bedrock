package cli

import (
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/xrpl-commons/bedrock/pkg/builder"
	"github.com/xrpl-commons/bedrock/pkg/config"
	"github.com/xrpl-commons/bedrock/pkg/primitives"
	"github.com/xrpl-commons/bedrock/pkg/vault"
)

var (
	vaultAsset         string
	vaultIssuer        string
	vaultAssetsMaximum string
	vaultAmount        string
	vaultDestination   string
	vaultWallet        string
	vaultNetwork       string
	vaultFee           string
	vaultSkipBuild     bool
)

var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Manage XRPL smart vaults",
	Long: `Create, deposit, withdraw and inspect XRPL smart vaults.

Smart vaults use WASM code to define custom deposit/withdraw logic.
The WASM on_deposit() and on_withdraw() functions return 1 to allow or 0 to deny.`,
}

var vaultDeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Create a smart vault with WASM logic",
	Long: `Build the vault WASM and deploy it via VaultCreate transaction.

Examples:
  bedrock vault deploy --asset XRP --wallet sXXX... --network local
  bedrock vault deploy --asset USD --issuer rXXX... --wallet sXXX... --network alphanet`,
	RunE: runVaultDeploy,
}

var vaultDepositCmd = &cobra.Command{
	Use:   "deposit <vault-id>",
	Short: "Deposit into a smart vault",
	Args:  cobra.ExactArgs(1),
	RunE:  runVaultDeposit,
}

var vaultWithdrawCmd = &cobra.Command{
	Use:   "withdraw <vault-id>",
	Short: "Withdraw from a smart vault",
	Args:  cobra.ExactArgs(1),
	RunE:  runVaultWithdraw,
}

var vaultStatusCmd = &cobra.Command{
	Use:   "status <vault-id>",
	Short: "Query vault status",
	Args:  cobra.ExactArgs(1),
	RunE:  runVaultStatus,
}

func init() {
	rootCmd.AddCommand(vaultCmd)
	vaultCmd.AddCommand(vaultDeployCmd)
	vaultCmd.AddCommand(vaultDepositCmd)
	vaultCmd.AddCommand(vaultWithdrawCmd)
	vaultCmd.AddCommand(vaultStatusCmd)

	// Deploy flags
	vaultDeployCmd.Flags().StringVar(&vaultAsset, "asset", "XRP", "Asset currency code")
	vaultDeployCmd.Flags().StringVar(&vaultIssuer, "issuer", "", "Asset issuer (for non-XRP assets)")
	vaultDeployCmd.Flags().StringVar(&vaultAssetsMaximum, "max-capacity", "", "Maximum vault capacity")
	vaultDeployCmd.Flags().StringVar(&vaultWallet, "wallet", "", "Wallet seed or name")
	vaultDeployCmd.Flags().StringVar(&vaultNetwork, "network", "local", "Network (local, alphanet)")
	vaultDeployCmd.Flags().StringVar(&vaultFee, "fee", "", "Transaction fee in drops")
	vaultDeployCmd.Flags().BoolVar(&vaultSkipBuild, "skip-build", false, "Skip building WASM")

	// Deposit flags
	vaultDepositCmd.Flags().StringVar(&vaultAmount, "amount", "", "Amount in drops (required)")
	vaultDepositCmd.Flags().StringVar(&vaultWallet, "wallet", "", "Wallet seed or name (required)")
	vaultDepositCmd.Flags().StringVar(&vaultNetwork, "network", "local", "Network")
	vaultDepositCmd.Flags().StringVar(&vaultFee, "fee", "", "Transaction fee in drops")
	_ = vaultDepositCmd.MarkFlagRequired("amount")
	_ = vaultDepositCmd.MarkFlagRequired("wallet")

	// Withdraw flags
	vaultWithdrawCmd.Flags().StringVar(&vaultAmount, "amount", "", "Amount in drops (required)")
	vaultWithdrawCmd.Flags().StringVar(&vaultDestination, "destination", "", "Withdrawal destination (required)")
	vaultWithdrawCmd.Flags().StringVar(&vaultWallet, "wallet", "", "Wallet seed or name (required)")
	vaultWithdrawCmd.Flags().StringVar(&vaultNetwork, "network", "local", "Network")
	vaultWithdrawCmd.Flags().StringVar(&vaultFee, "fee", "", "Transaction fee in drops")
	_ = vaultWithdrawCmd.MarkFlagRequired("amount")
	_ = vaultWithdrawCmd.MarkFlagRequired("destination")
	_ = vaultWithdrawCmd.MarkFlagRequired("wallet")

	// Status flags
	vaultStatusCmd.Flags().StringVar(&vaultNetwork, "network", "local", "Network")
}

func runVaultDeploy(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	netCfg, ok := cfg.Networks[vaultNetwork]
	if !ok {
		return fmt.Errorf("unknown network %q", vaultNetwork)
	}

	def := primitives.Registry[primitives.Vault]

	// Build WASM if needed
	var wasmPath string
	if !vaultSkipBuild {
		color.Cyan("Building vault WASM...\n")
		b := builder.New(".")
		result, err := b.Build(cmd.Context(), builder.BuildOptions{
			Release:   true,
			Target:    def.WasmTarget,
			SourceDir: def.SourceDir,
		})
		if err != nil {
			return fmt.Errorf("build failed: %w", err)
		}
		wasmPath = result.WasmPath
		color.Green("✓ Built: %s (%d bytes)\n\n", wasmPath, result.Size)
	} else {
		return fmt.Errorf("--skip-build requires a pre-built WASM file; build first with 'bedrock build --type vault'")
	}

	// Build asset object
	var asset interface{}
	if vaultAsset == "XRP" {
		asset = map[string]string{"currency": "XRP"}
	} else {
		if vaultIssuer == "" {
			return fmt.Errorf("--issuer is required for non-XRP assets")
		}
		asset = map[string]string{"currency": vaultAsset, "issuer": vaultIssuer}
	}

	walletSeed, err := resolveWalletFlag(vaultWallet)
	if err != nil {
		return err
	}

	color.Cyan("Deploying smart vault...\n")
	fmt.Printf("  Asset: %s\n", vaultAsset)
	if vaultIssuer != "" {
		fmt.Printf("  Issuer: %s\n", vaultIssuer)
	}
	fmt.Printf("  Network: %s (%s)\n", vaultNetwork, netCfg.URL)
	fmt.Println()

	op, err := vault.NewOperator(false)
	if err != nil {
		return err
	}

	result, err := op.Deploy(cmd.Context(), vault.DeployConfig{
		WasmPath:      wasmPath,
		Asset:         asset,
		AssetsMaximum: vaultAssetsMaximum,
		NetworkURL:    netCfg.URL,
		NetworkID:     netCfg.NetworkID,
		WalletSeed:    walletSeed,
		FaucetURL:     netCfg.FaucetURL,
		Fee:           vaultFee,
	})
	if err != nil {
		color.Red("\n✗ Vault deploy failed: %v\n", err)
		return err
	}

	color.Green("\n✓ Smart vault created!\n\n")
	fmt.Printf("  Tx Hash: %s\n", result.TxHash)
	fmt.Printf("  Wallet: %s\n", result.WalletAddress)
	fmt.Printf("  Seed: %s\n", result.WalletSeed)
	fmt.Printf("  Vault ID: %s\n", result.VaultID)
	fmt.Printf("  Validated: %v\n", result.Validated)

	return nil
}

func runVaultDeposit(cmd *cobra.Command, args []string) error {
	vaultID := args[0]

	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	netCfg, ok := cfg.Networks[vaultNetwork]
	if !ok {
		return fmt.Errorf("unknown network %q", vaultNetwork)
	}

	walletSeed, err := resolveWalletFlag(vaultWallet)
	if err != nil {
		return err
	}

	color.Cyan("Depositing %s drops into vault %s...\n", vaultAmount, vaultID)

	op, err := vault.NewOperator(false)
	if err != nil {
		return err
	}

	result, err := op.Deposit(cmd.Context(), vault.DepositConfig{
		VaultID:    vaultID,
		Amount:     vaultAmount,
		NetworkURL: netCfg.URL,
		NetworkID:  netCfg.NetworkID,
		WalletSeed: walletSeed,
		Fee:        vaultFee,
	})
	if err != nil {
		color.Red("\n✗ Vault deposit failed: %v\n", err)
		return err
	}

	color.Green("\n✓ Deposit successful!\n\n")
	fmt.Printf("  Tx Hash: %s\n", result.TxHash)
	fmt.Printf("  Validated: %v\n", result.Validated)

	return nil
}

func runVaultWithdraw(cmd *cobra.Command, args []string) error {
	vaultID := args[0]

	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	netCfg, ok := cfg.Networks[vaultNetwork]
	if !ok {
		return fmt.Errorf("unknown network %q", vaultNetwork)
	}

	walletSeed, err := resolveWalletFlag(vaultWallet)
	if err != nil {
		return err
	}

	color.Cyan("Withdrawing %s drops from vault %s...\n", vaultAmount, vaultID)

	op, err := vault.NewOperator(false)
	if err != nil {
		return err
	}

	result, err := op.Withdraw(cmd.Context(), vault.WithdrawConfig{
		VaultID:     vaultID,
		Amount:      vaultAmount,
		Destination: vaultDestination,
		NetworkURL:  netCfg.URL,
		NetworkID:   netCfg.NetworkID,
		WalletSeed:  walletSeed,
		Fee:         vaultFee,
	})
	if err != nil {
		color.Red("\n✗ Vault withdraw failed: %v\n", err)
		return err
	}

	color.Green("\n✓ Withdrawal successful!\n\n")
	fmt.Printf("  Tx Hash: %s\n", result.TxHash)
	fmt.Printf("  Validated: %v\n", result.Validated)

	return nil
}

func runVaultStatus(cmd *cobra.Command, args []string) error {
	vaultID := args[0]

	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	netCfg, ok := cfg.Networks[vaultNetwork]
	if !ok {
		return fmt.Errorf("unknown network %q", vaultNetwork)
	}

	op, err := vault.NewOperator(false)
	if err != nil {
		return err
	}

	data, err := op.Status(cmd.Context(), vault.StatusConfig{
		VaultID:    vaultID,
		NetworkURL: netCfg.URL,
	})
	if err != nil {
		color.Red("✗ Vault status query failed: %v\n", err)
		return err
	}

	var pretty json.RawMessage
	if err := json.Unmarshal(data, &pretty); err == nil {
		formatted, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println(string(formatted))
	} else {
		fmt.Println(string(data))
	}

	return nil
}
