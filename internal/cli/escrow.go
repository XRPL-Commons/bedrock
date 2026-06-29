package cli

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/xrpl-commons/bedrock/pkg/builder"
	"github.com/xrpl-commons/bedrock/pkg/config"
	"github.com/xrpl-commons/bedrock/pkg/escrow"
	"github.com/xrpl-commons/bedrock/pkg/primitives"
	"github.com/xrpl-commons/bedrock/pkg/wallet"
)

var (
	escrowDestination string
	escrowAmount      string
	escrowCancelAfter int64
	escrowFinishAfter int64
	escrowWallet      string
	escrowNetwork     string
	escrowGas         string
	escrowFee         string
	escrowSkipBuild   bool
)

var escrowCmd = &cobra.Command{
	Use:   "escrow",
	Short: "Manage XRPL smart escrows",
	Long: `Create, finish, cancel and inspect XRPL smart escrows.

Smart escrows use WASM code to define custom release conditions.
The WASM finish() function returns 1 to release or 0 to keep locked.`,
}

var escrowDeployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Create a smart escrow with WASM condition",
	Long: `Build the escrow WASM and deploy it via EscrowCreate transaction.

The WASM code defines the finish() condition that determines when the escrow can be released.

Examples:
  bedrock escrow deploy --destination rXXX... --amount 1000000 --wallet sXXX... --network local
  bedrock escrow deploy --destination rXXX... --amount 1000000 --cancel-after 1234567 --wallet sXXX... --network alphanet`,
	RunE: runEscrowDeploy,
}

var escrowFinishCmd = &cobra.Command{
	Use:   "finish <owner> <sequence>",
	Short: "Finish (release) a smart escrow",
	Long: `Submit an EscrowFinish transaction. The on-chain WASM finish() function
will execute and determine whether the escrow should be released.`,
	Args: cobra.ExactArgs(2),
	RunE: runEscrowFinish,
}

var escrowCancelCmd = &cobra.Command{
	Use:   "cancel <owner> <sequence>",
	Short: "Cancel a smart escrow",
	Args:  cobra.ExactArgs(2),
	RunE:  runEscrowCancel,
}

var escrowStatusCmd = &cobra.Command{
	Use:   "status <owner> <sequence>",
	Short: "Query escrow status",
	Args:  cobra.ExactArgs(2),
	RunE:  runEscrowStatus,
}

func init() {
	rootCmd.AddCommand(escrowCmd)
	escrowCmd.AddCommand(escrowDeployCmd)
	escrowCmd.AddCommand(escrowFinishCmd)
	escrowCmd.AddCommand(escrowCancelCmd)
	escrowCmd.AddCommand(escrowStatusCmd)

	// Deploy flags
	escrowDeployCmd.Flags().StringVar(&escrowDestination, "destination", "", "Escrow beneficiary address (required)")
	escrowDeployCmd.Flags().StringVar(&escrowAmount, "amount", "", "Amount in drops (required)")
	escrowDeployCmd.Flags().Int64Var(&escrowCancelAfter, "cancel-after", 0, "Cancel after time (ripple epoch)")
	escrowDeployCmd.Flags().Int64Var(&escrowFinishAfter, "finish-after", 0, "Finish after time (ripple epoch)")
	escrowDeployCmd.Flags().StringVar(&escrowWallet, "wallet", "", "Wallet seed or name")
	escrowDeployCmd.Flags().StringVar(&escrowNetwork, "network", "local", "Network (local, alphanet)")
	escrowDeployCmd.Flags().StringVar(&escrowFee, "fee", "", "Transaction fee in drops")
	escrowDeployCmd.Flags().BoolVar(&escrowSkipBuild, "skip-build", false, "Skip building WASM")
	_ = escrowDeployCmd.MarkFlagRequired("destination")
	_ = escrowDeployCmd.MarkFlagRequired("amount")

	// Finish flags
	escrowFinishCmd.Flags().StringVar(&escrowWallet, "wallet", "", "Wallet seed or name (required)")
	escrowFinishCmd.Flags().StringVar(&escrowNetwork, "network", "local", "Network")
	escrowFinishCmd.Flags().StringVarP(&escrowGas, "gas", "g", "1000000", "Computation allowance")
	escrowFinishCmd.Flags().StringVar(&escrowFee, "fee", "", "Transaction fee in drops")
	_ = escrowFinishCmd.MarkFlagRequired("wallet")

	// Cancel flags
	escrowCancelCmd.Flags().StringVar(&escrowWallet, "wallet", "", "Wallet seed or name (required)")
	escrowCancelCmd.Flags().StringVar(&escrowNetwork, "network", "local", "Network")
	escrowCancelCmd.Flags().StringVar(&escrowFee, "fee", "", "Transaction fee in drops")
	_ = escrowCancelCmd.MarkFlagRequired("wallet")

	// Status flags
	escrowStatusCmd.Flags().StringVar(&escrowNetwork, "network", "local", "Network")
}

func runEscrowDeploy(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	netCfg, ok := cfg.Networks[escrowNetwork]
	if !ok {
		return fmt.Errorf("unknown network %q", escrowNetwork)
	}

	def := primitives.Registry[primitives.Escrow]

	// Build WASM if needed
	var wasmPath string
	if !escrowSkipBuild {
		color.Cyan("Building escrow WASM...\n")
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
		// Find existing WASM
		// TODO: locate a pre-built WASM in the output dir and use it.
		return fmt.Errorf("--skip-build requires a pre-built WASM file; build first with 'bedrock build --type escrow'")
	}

	color.Cyan("Deploying smart escrow...\n")
	fmt.Printf("  Destination: %s\n", escrowDestination)
	fmt.Printf("  Amount: %s drops\n", escrowAmount)
	fmt.Printf("  Network: %s (%s)\n", escrowNetwork, netCfg.URL)
	fmt.Println()

	walletSeed, err := resolveWalletFlag(escrowWallet)
	if err != nil {
		return err
	}

	isVerbose := Verbose()
	op, err := escrow.NewOperator(isVerbose)
	if err != nil {
		return err
	}

	result, err := op.Deploy(cmd.Context(), escrow.DeployConfig{
		WasmPath:    wasmPath,
		Destination: escrowDestination,
		Amount:      escrowAmount,
		CancelAfter: escrowCancelAfter,
		FinishAfter: escrowFinishAfter,
		NetworkURL:  netCfg.URL,
		NetworkID:   netCfg.NetworkID,
		WalletSeed:  walletSeed,
		FaucetURL:   netCfg.FaucetURL,
		Fee:         escrowFee,
	})
	if err != nil {
		color.Red("\n✗ Escrow deploy failed: %v\n", err)
		return err
	}

	color.Green("\n✓ Smart escrow created!\n\n")
	fmt.Printf("  Tx Hash: %s\n", result.TxHash)
	fmt.Printf("  Wallet: %s\n", result.WalletAddress)
	fmt.Printf("  Seed: %s\n", maskSeed(result.WalletSeed))
	fmt.Printf("  Escrow Sequence: %d\n", result.EscrowSequence)
	fmt.Printf("  Validated: %v\n", result.Validated)

	return nil
}

func runEscrowFinish(cmd *cobra.Command, args []string) error {
	owner := args[0]
	seq, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid sequence number: %w", err)
	}

	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	netCfg, ok := cfg.Networks[escrowNetwork]
	if !ok {
		return fmt.Errorf("unknown network %q", escrowNetwork)
	}

	walletSeed, err := resolveWalletFlag(escrowWallet)
	if err != nil {
		return err
	}

	color.Cyan("Finishing escrow %s seq %d...\n", owner, seq)

	isVerbose := Verbose()
	op, err := escrow.NewOperator(isVerbose)
	if err != nil {
		return err
	}

	result, err := op.Finish(cmd.Context(), escrow.FinishConfig{
		Owner:                owner,
		EscrowSequence:       seq,
		NetworkURL:           netCfg.URL,
		NetworkID:            netCfg.NetworkID,
		WalletSeed:           walletSeed,
		ComputationAllowance: escrowGas,
		Fee:                  escrowFee,
	})
	if err != nil {
		color.Red("\n✗ Escrow finish failed: %v\n", err)
		return err
	}

	color.Green("\n✓ Escrow finish submitted!\n\n")
	fmt.Printf("  Tx Hash: %s\n", result.TxHash)
	fmt.Printf("  Return Code: %v\n", result.ReturnCode)
	fmt.Printf("  Validated: %v\n", result.Validated)

	// Return code can be int (WASM return) or string (transaction result)
	switch v := result.ReturnCode.(type) {
	case float64:
		if v > 0 {
			color.Green("  Result: Escrow released\n")
		} else {
			color.Yellow("  Result: Escrow kept locked (condition not met)\n")
		}
	case string:
		if v == "tesSUCCESS" {
			color.Green("  Result: Escrow released\n")
		} else {
			color.Yellow("  Result: %s\n", v)
		}
	default:
		color.Yellow("  Result: Unknown return code type\n")
	}

	return nil
}

func runEscrowCancel(cmd *cobra.Command, args []string) error {
	owner := args[0]
	seq, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid sequence number: %w", err)
	}

	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	netCfg, ok := cfg.Networks[escrowNetwork]
	if !ok {
		return fmt.Errorf("unknown network %q", escrowNetwork)
	}

	walletSeed, err := resolveWalletFlag(escrowWallet)
	if err != nil {
		return err
	}

	color.Cyan("Cancelling escrow %s seq %d...\n", owner, seq)

	isVerbose := Verbose()
	op, err := escrow.NewOperator(isVerbose)
	if err != nil {
		return err
	}

	result, err := op.Cancel(cmd.Context(), escrow.CancelConfig{
		Owner:          owner,
		EscrowSequence: seq,
		NetworkURL:     netCfg.URL,
		NetworkID:      netCfg.NetworkID,
		WalletSeed:     walletSeed,
		Fee:            escrowFee,
	})
	if err != nil {
		color.Red("\n✗ Escrow cancel failed: %v\n", err)
		return err
	}

	color.Green("\n✓ Escrow cancelled!\n\n")
	fmt.Printf("  Tx Hash: %s\n", result.TxHash)
	fmt.Printf("  Validated: %v\n", result.Validated)

	return nil
}

func runEscrowStatus(cmd *cobra.Command, args []string) error {
	owner := args[0]
	seq, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid sequence number: %w", err)
	}

	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	netCfg, ok := cfg.Networks[escrowNetwork]
	if !ok {
		return fmt.Errorf("unknown network %q", escrowNetwork)
	}

	isVerbose := Verbose()
	op, err := escrow.NewOperator(isVerbose)
	if err != nil {
		return err
	}

	data, err := op.Status(cmd.Context(), escrow.StatusConfig{
		Owner:          owner,
		EscrowSequence: seq,
		NetworkURL:     netCfg.URL,
	})
	if err != nil {
		color.Red("✗ Escrow status query failed: %v\n", err)
		return err
	}

	// Pretty-print JSON
	var pretty json.RawMessage
	if err := json.Unmarshal(data, &pretty); err == nil {
		formatted, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println(string(formatted))
	} else {
		fmt.Println(string(data))
	}

	return nil
}

// resolveWalletFlag resolves a wallet flag value (name or raw seed) via jade keystore
func resolveWalletFlag(input string) (string, error) {
	if input == "" {
		return "", nil
	}
	resolver, err := wallet.NewWalletResolver()
	if err != nil {
		return "", fmt.Errorf("failed to initialize wallet resolver: %w", err)
	}
	seed, err := resolver.ResolveWallet(input)
	if err != nil {
		return "", fmt.Errorf("failed to resolve wallet: %w", err)
	}
	return seed, nil
}
