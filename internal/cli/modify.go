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
	modifyNetwork       string
	modifyWallet        string
	modifyWasm          string
	modifyABI           string
	modifyAlgorithm     string
	modifyFee           string
	modifyOwner         string
	modifyHash          string
	modifyImmutable     bool
	modifyCodeImmutable bool
	modifyABIImmutable  bool
	modifyUndeletable   bool
)

var modifyCmd = &cobra.Command{
	Use:   "modify <contract-account>",
	Short: "Modify a deployed contract",
	Long: `Update a deployed contract's code, ABI, owner, or flags via ContractModify transaction.

Examples:
  bedrock modify rContract123... --wallet sXXX... --wasm contract.wasm
  bedrock modify rContract123... --wallet sXXX... --abi abi.json
  bedrock modify rContract123... --wallet sXXX... --owner rNewOwner...
  bedrock modify rContract123... --wallet sXXX... --immutable
  bedrock modify rContract123... --wallet sXXX... --hash ABCDEF1234...`,
	Args: cobra.ExactArgs(1),
	RunE: runModify,
}

func init() {
	rootCmd.AddCommand(modifyCmd)

	modifyCmd.Flags().StringVarP(&modifyNetwork, "network", "n", "alphanet", "Network")
	modifyCmd.Flags().StringVarP(&modifyWallet, "wallet", "w", "", "Wallet seed or name (required)")
	modifyCmd.Flags().StringVar(&modifyWasm, "wasm", "", "New WASM file path")
	modifyCmd.Flags().StringVar(&modifyABI, "abi", "", "New ABI file path")
	modifyCmd.Flags().StringVar(&modifyAlgorithm, "algorithm", "secp256k1", "Cryptographic algorithm")
	modifyCmd.Flags().StringVar(&modifyFee, "fee", "10000000", "Transaction fee in drops")
	modifyCmd.Flags().StringVar(&modifyOwner, "owner", "", "New contract owner address")
	modifyCmd.Flags().StringVar(&modifyHash, "hash", "", "Reference existing ContractSource by hash")
	modifyCmd.Flags().BoolVar(&modifyImmutable, "immutable", false, "Set lsfImmutable flag")
	modifyCmd.Flags().BoolVar(&modifyCodeImmutable, "code-immutable", false, "Set lsfCodeImmutable flag")
	modifyCmd.Flags().BoolVar(&modifyABIImmutable, "abi-immutable", false, "Set lsfABIImmutable flag")
	modifyCmd.Flags().BoolVar(&modifyUndeletable, "undeletable", false, "Set lsfUndeletable flag")

	modifyCmd.MarkFlagRequired("wallet")
}

func runModify(cmd *cobra.Command, args []string) error {
	contractAccount := args[0]

	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'bedrock init' first)", err)
	}

	if !cfg.HasPrimitive("contract") {
		return fmt.Errorf("modify is only available for smart contract projects")
	}

	networkCfg, ok := cfg.Networks[modifyNetwork]
	if !ok {
		if modifyNetwork == "local" {
			networkCfg = config.NetworkConfig{URL: "ws://localhost:6006", NetworkID: 100}
		} else {
			return fmt.Errorf("network '%s' not found in config", modifyNetwork)
		}
	}

	hasFlags := modifyImmutable || modifyCodeImmutable || modifyABIImmutable || modifyUndeletable
	if modifyWasm == "" && modifyABI == "" && modifyOwner == "" && modifyHash == "" && !hasFlags {
		return fmt.Errorf("at least one of --wasm, --abi, --owner, --hash, or a flag must be provided")
	}

	color.Cyan("Modifying contract\n")
	fmt.Printf("  Contract: %s\n", contractAccount)
	fmt.Printf("  Network:  %s\n", modifyNetwork)

	resolver, err := wallet.NewWalletResolver()
	if err != nil {
		return fmt.Errorf("failed to initialize wallet resolver: %w", err)
	}

	walletSeed, err := resolver.ResolveWallet(modifyWallet)
	if err != nil {
		return fmt.Errorf("failed to resolve wallet: %w", err)
	}

	d, err := deployer.NewDeployer(false)
	if err != nil {
		return fmt.Errorf("failed to initialize deployer: %w", err)
	}

	ctx := cmd.Context()
	result, err := d.Modify(ctx, deployer.ModifyConfig{
		ContractAccount: contractAccount,
		NetworkURL:      networkCfg.URL,
		WalletSeed:      walletSeed,
		Algorithm:       modifyAlgorithm,
		WasmPath:        modifyWasm,
		ABIPath:         modifyABI,
		Fee:             modifyFee,
		Owner:           modifyOwner,
		ContractHash:    modifyHash,
		Immutable:       modifyImmutable,
		CodeImmutable:   modifyCodeImmutable,
		ABIImmutable:    modifyABIImmutable,
		Undeletable:     modifyUndeletable,
	})

	if err != nil {
		color.Red("\n✗ Modification failed: %v\n", err)
		return err
	}

	color.Green("\n✓ Contract modified successfully!\n")
	fmt.Printf("  Transaction Hash: %s\n", result.TxHash)
	fmt.Printf("  Validated: %v\n", result.Validated)

	return nil
}
