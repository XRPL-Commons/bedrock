package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/xrpl-commons/bedrock/pkg/caller"
	"github.com/xrpl-commons/bedrock/pkg/config"
	"github.com/xrpl-commons/bedrock/pkg/wallet"
)

var (
	callNetwork    string
	callWallet     string
	callABI        string
	callParams     string
	callParamsFile string
	callGas        string
	callFee        string
	callAlgorithm  string
)

var callCmd = &cobra.Command{
	Use:   "call <contract-account> <function-name>",
	Short: "Call a contract function",
	Long: `Call a function on a deployed smart contract.

Examples:
  bedrock call rContract123... hello
  bedrock call rContract123... register --params '{"name":"alice","age":25}'
  bedrock call rContract123... transfer --params-file params.json --wallet sXXX...`,
	Args: cobra.ExactArgs(2),
	RunE: runCall,
}

func init() {
	rootCmd.AddCommand(callCmd)

	callCmd.Flags().StringVarP(&callNetwork, "network", "n", "alphanet", "Network to call on (local, alphanet, testnet, mainnet)")
	callCmd.Flags().StringVarP(&callWallet, "wallet", "w", "", "Wallet seed or name (required)")
	callCmd.Flags().StringVarP(&callABI, "abi", "a", "abi.json", "Path to ABI file")
	callCmd.Flags().StringVarP(&callParams, "params", "p", "", "Parameters as JSON string")
	callCmd.Flags().StringVarP(&callParamsFile, "params-file", "f", "", "Parameters from JSON file")
	callCmd.Flags().StringVarP(&callGas, "gas", "g", "1000000", "Computation allowance")
	callCmd.Flags().StringVar(&callFee, "fee", "1000000", "Transaction fee in drops")
	callCmd.Flags().StringVar(&callAlgorithm, "algorithm", "secp256k1", "Cryptographic algorithm (secp256k1, ed25519)")

	callCmd.MarkFlagRequired("wallet")
}

func runCall(cmd *cobra.Command, args []string) error {
	contractAccount := args[0]
	functionName := args[1]

	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'bedrock init' first)", err)
	}

	if !cfg.HasPrimitive("contract") {
		return fmt.Errorf("no contract primitive in this project\nUse 'bedrock escrow finish' or 'bedrock vault deposit/withdraw' instead")
	}

	color.Cyan("Calling smart contract function\n")
	fmt.Printf("   Network: %s\n", callNetwork)
	fmt.Printf("   Contract: %s\n", contractAccount)
	fmt.Printf("   Function: %s\n", functionName)

	// Get network configuration
	networkCfg, ok := cfg.Networks[callNetwork]
	if !ok {
		if callNetwork == "local" {
			networkCfg = config.NetworkConfig{
				URL:       "ws://localhost:6006",
				NetworkID: 0, // Local network uses network ID 0
			}
		} else {
			return fmt.Errorf("network '%s' not found in config", callNetwork)
		}
	}

	fmt.Printf("   URL: %s\n", networkCfg.URL)
	fmt.Printf("   ABI: %s\n", callABI)

	// Parse parameters
	var params map[string]interface{}
	if callParamsFile != "" {
		// Load from file
		data, err := os.ReadFile(callParamsFile)
		if err != nil {
			return fmt.Errorf("failed to read params file: %w", err)
		}
		if err := json.Unmarshal(data, &params); err != nil {
			return fmt.Errorf("invalid JSON in params file: %w", err)
		}
		fmt.Printf("   Parameters: (from %s)\n", callParamsFile)
	} else if callParams != "" {
		// Parse JSON string
		if err := json.Unmarshal([]byte(callParams), &params); err != nil {
			return fmt.Errorf("invalid parameters JSON: %w", err)
		}
		fmt.Printf("   Parameters: %s\n", callParams)
	} else {
		fmt.Printf("   Parameters: (none)\n")
	}

	// Resolve wallet seed
	resolver, err := wallet.NewWalletResolver()
	if err != nil {
		color.Red("✗ Failed to initialize wallet resolver: %v\n", err)
		return err
	}

	walletSeed, err := resolver.ResolveWallet(callWallet)
	if err != nil {
		color.Red("✗ Failed to resolve wallet: %v\n", err)
		return err
	}

	fmt.Println()

	// Create caller
	verbose := false // TODO: get from global flag
	c, err := caller.NewCaller(verbose)
	if err != nil {
		color.Red("✗ Failed to initialize caller: %v\n", err)
		return err
	}

	// Call contract
	color.Yellow("→ Executing contract call...\n\n")

	ctx := cmd.Context()
	result, err := c.Call(ctx, caller.CallConfig{
		ContractAccount:      contractAccount,
		FunctionName:         functionName,
		NetworkURL:           networkCfg.URL,
		NetworkID:            networkCfg.NetworkID,
		WalletSeed:           walletSeed,
		Algorithm:            callAlgorithm,
		ABIPath:              callABI,
		Parameters:           params,
		ComputationAllowance: callGas,
		Fee:                  callFee,
	})

	if err != nil {
		color.Red("✗ Contract call failed: %v\n", err)
		return err
	}

	// Display results
	color.Green("✓ Contract function called successfully!\n")
	fmt.Println()
	color.Cyan("Call Results:\n")
	fmt.Printf("  Transaction Hash: %s\n", result.TxHash)
	fmt.Printf("  Return Code: %d", result.ReturnCode)
	if result.ReturnCode == 0 {
		color.Green(" (SUCCESS)\n")
	} else {
		color.Red(" (ERROR)\n")
	}

	if result.ReturnValue != "" {
		fmt.Printf("  Return Value: %s\n", result.ReturnValue)
		if result.ReturnValue != "" {
			// Try to show as decimal if it's a number
			fmt.Printf("  Return Value (decimal): %d\n", hexToInt(result.ReturnValue))
		}
	}

	if result.GasUsed > 0 {
		fmt.Printf("  Gas Used: %d / %s\n", result.GasUsed, callGas)
	}

	fmt.Printf("  Validated: %v\n", result.Validated)

	return nil
}

// hexToInt converts hex string to int (best effort)
func hexToInt(hex string) int64 {
	var val int64
	fmt.Sscanf(hex, "%x", &val)
	return val
}
