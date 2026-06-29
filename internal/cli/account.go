package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/xrpl-commons/bedrock/pkg/chain"
	"github.com/xrpl-commons/bedrock/pkg/config"
)

var accountNetwork string

var accountCmd = &cobra.Command{
	Use:   "account <info|balance|objects|lines> <address>",
	Short: "Query account information",
	Long: `Query XRPL account information.

Commands:
  info     - Full account information (account_info RPC)
  balance  - XRP balance
  objects  - Account-owned objects (contracts, escrows, etc.)
  lines    - Trust lines

Examples:
  bedrock account info rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh
  bedrock account balance rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh
  bedrock account objects rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh --network local`,
	Args: cobra.ExactArgs(2),
	RunE: runAccount,
}

func init() {
	rootCmd.AddCommand(accountCmd)

	accountCmd.Flags().StringVarP(&accountNetwork, "network", "n", "alphanet", "Network to query (local, alphanet, testnet, mainnet)")
}

func runAccount(cmd *cobra.Command, args []string) error {
	subcommand := args[0]
	address := args[1]

	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'bedrock init' first)", err)
	}

	networkCfg, ok := cfg.Networks[accountNetwork]
	if !ok {
		if accountNetwork == "local" {
			networkCfg = config.NetworkConfig{
				URL:       "ws://localhost:6006",
				NetworkID: 100,
			}
		} else {
			return fmt.Errorf("network '%s' not found in config", accountNetwork)
		}
	}

	client := chain.NewClient(networkCfg.URL)
	ctx := cmd.Context()

	switch subcommand {
	case "info":
		return accountInfo(ctx, client, address)
	case "balance":
		return accountBalance(ctx, client, address)
	case "objects":
		return accountObjects(ctx, client, address)
	case "lines":
		return accountLines(ctx, client, address)
	default:
		return fmt.Errorf("unknown subcommand: %s (use: info, balance, objects, lines)", subcommand)
	}
}

func accountInfo(ctx context.Context, client *chain.Client, address string) error {
	color.Cyan("Account Info\n")
	fmt.Printf("  Address: %s\n\n", address)

	info, err := client.GetAccountInfo(ctx, address)
	if err != nil {
		color.Red("Failed to get account info: %v\n", err)
		return err
	}

	fmt.Printf("  Balance:      %s XRP (%s drops)\n", chain.DropsToXRP(info.AccountData.Balance), info.AccountData.Balance)
	fmt.Printf("  Sequence:     %d\n", info.AccountData.Sequence)
	fmt.Printf("  Owner Count:  %d\n", info.AccountData.OwnerCount)
	fmt.Printf("  Flags:        %d\n", info.AccountData.Flags)
	fmt.Printf("  Ledger Index: %d\n", info.LedgerIndex)

	if info.AccountData.PreviousTxnID != "" {
		fmt.Printf("  Last Tx:      %s\n", info.AccountData.PreviousTxnID)
	}

	return nil
}

func accountBalance(ctx context.Context, client *chain.Client, address string) error {
	balance, err := client.GetAccountBalance(ctx, address)
	if err != nil {
		color.Red("Failed to get balance: %v\n", err)
		return err
	}

	fmt.Printf("%s XRP\n", balance)
	return nil
}

func accountObjects(ctx context.Context, client *chain.Client, address string) error {
	color.Cyan("Account Objects\n")
	fmt.Printf("  Address: %s\n\n", address)

	objects, err := client.GetAccountObjects(ctx, address)
	if err != nil {
		color.Red("Failed to get objects: %v\n", err)
		return err
	}

	if len(objects.Objects) == 0 {
		fmt.Println("  No objects found")
		return nil
	}

	fmt.Printf("  Found %d objects:\n\n", len(objects.Objects))
	for i, obj := range objects.Objects {
		var parsed map[string]interface{}
		_ = json.Unmarshal(obj, &parsed)

		entryType, _ := parsed["LedgerEntryType"].(string)
		fmt.Printf("  [%d] %s\n", i+1, entryType)

		pretty, _ := json.MarshalIndent(parsed, "      ", "  ")
		fmt.Printf("      %s\n\n", string(pretty))
	}

	return nil
}

func accountLines(ctx context.Context, client *chain.Client, address string) error {
	color.Cyan("Trust Lines\n")
	fmt.Printf("  Address: %s\n\n", address)

	lines, err := client.GetAccountLines(ctx, address)
	if err != nil {
		color.Red("Failed to get trust lines: %v\n", err)
		return err
	}

	if len(lines.Lines) == 0 {
		fmt.Println("  No trust lines found")
		return nil
	}

	fmt.Printf("  Found %d trust lines:\n\n", len(lines.Lines))
	for _, line := range lines.Lines {
		fmt.Printf("  %s  Balance: %s  Limit: %s  Peer: %s\n",
			line.Currency, line.Balance, line.Limit, line.Account)
	}

	return nil
}
