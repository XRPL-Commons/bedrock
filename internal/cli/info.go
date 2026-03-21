package cli

import (
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/xrpl-commons/bedrock/pkg/chain"
	"github.com/xrpl-commons/bedrock/pkg/config"
)

var (
	infoNetwork string
	infoUser    string
)

var infoCmd = &cobra.Command{
	Use:   "info <contract-account>",
	Short: "Display contract information",
	Long: `Fetch and display details about a deployed smart contract.

Shows ABI, functions, owner, immutability flags, and state data.
Use --user to show per-user contract state.

Examples:
  bedrock info rContract123...
  bedrock info rContract123... --user rUser456...
  bedrock info rContract123... --network local`,
	Args: cobra.ExactArgs(1),
	RunE: runInfo,
}

func init() {
	rootCmd.AddCommand(infoCmd)

	infoCmd.Flags().StringVarP(&infoNetwork, "network", "n", "alphanet", "Network to query")
	infoCmd.Flags().StringVar(&infoUser, "user", "", "Show per-user state for this address")
}

func runInfo(cmd *cobra.Command, args []string) error {
	contractAccount := args[0]

	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'bedrock init' first)", err)
	}

	networkCfg, ok := cfg.Networks[infoNetwork]
	if !ok {
		if infoNetwork == "local" {
			networkCfg = config.NetworkConfig{
				URL:       "ws://localhost:6006",
				RPCURL:    "http://localhost:5005",
				NetworkID: 63456,
			}
		} else {
			return fmt.Errorf("network '%s' not found in config", infoNetwork)
		}
	}

	client := chain.NewClient(networkCfg.URL, networkCfg.GetRPCURL())
	ctx := cmd.Context()

	color.Cyan("Contract Info\n")
	fmt.Printf("  Account: %s\n", contractAccount)
	fmt.Printf("  Network: %s\n\n", infoNetwork)

	info, err := client.GetContractInfo(ctx, contractAccount)
	if err != nil {
		color.Red("Failed to get contract info: %v\n", err)
		return err
	}

	if info.Owner != "" {
		fmt.Printf("  Owner:   %s\n", info.Owner)
	}
	fmt.Printf("  Flags:   %d\n", info.Flags)
	displayContractFlags(info.Flags)

	if info.WasmHash != "" {
		fmt.Printf("  WASM Hash: %s\n", info.WasmHash)
	}

	if info.ABI != nil {
		fmt.Println()
		color.Cyan("ABI:\n")
		var abiParsed interface{}
		if err := json.Unmarshal(info.ABI, &abiParsed); err == nil {
			pretty, _ := json.MarshalIndent(abiParsed, "  ", "  ")
			fmt.Printf("  %s\n", string(pretty))
		}
	}

	if info.ContractData != nil {
		fmt.Println()
		color.Cyan("Global State:\n")
		var stateParsed interface{}
		if err := json.Unmarshal(info.ContractData, &stateParsed); err == nil {
			pretty, _ := json.MarshalIndent(stateParsed, "  ", "  ")
			fmt.Printf("  %s\n", string(pretty))
		}
	}

	if infoUser != "" {
		fmt.Println()
		color.Cyan("User State (%s):\n", infoUser)
		userData, err := client.GetContractData(ctx, contractAccount, infoUser)
		if err != nil {
			color.Red("  Failed to get user state: %v\n", err)
		} else if userData == nil {
			fmt.Println("  No user state found")
		} else {
			var parsed interface{}
			if err := json.Unmarshal(userData, &parsed); err == nil {
				pretty, _ := json.MarshalIndent(parsed, "  ", "  ")
				fmt.Printf("  %s\n", string(pretty))
			}
		}
	}

	return nil
}

func displayContractFlags(flags int64) {
	const (
		lsfImmutable     = 0x00000001
		lsfCodeImmutable = 0x00000002
		lsfABIImmutable  = 0x00000004
		lsfUndeletable   = 0x00000008
	)

	if flags == 0 {
		return
	}

	fmt.Print("           ")
	if flags&lsfImmutable != 0 {
		color.Yellow("Immutable ")
	}
	if flags&lsfCodeImmutable != 0 {
		color.Yellow("CodeImmutable ")
	}
	if flags&lsfABIImmutable != 0 {
		color.Yellow("ABIImmutable ")
	}
	if flags&lsfUndeletable != 0 {
		color.Yellow("Undeletable ")
	}
	fmt.Println()
}
