package cli

import (
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/xrpl-commons/bedrock/pkg/chain"
	"github.com/xrpl-commons/bedrock/pkg/config"
)

var ledgerNetwork string

var ledgerCmd = &cobra.Command{
	Use:   "ledger [sequence]",
	Short: "Query ledger information",
	Long: `Query XRPL ledger information.

Without arguments, shows the current ledger.
With a sequence number, shows that specific ledger.

Examples:
  bedrock ledger
  bedrock ledger 12345
  bedrock ledger --network local`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLedger,
}

var txCmd = &cobra.Command{
	Use:   "tx <hash>",
	Short: "Look up a transaction",
	Long: `Look up a transaction by its hash.

Examples:
  bedrock tx ABC123...
  bedrock tx ABC123... --network local`,
	Args: cobra.ExactArgs(1),
	RunE: runTx,
}

var txNetwork string

func init() {
	rootCmd.AddCommand(ledgerCmd)
	rootCmd.AddCommand(txCmd)

	ledgerCmd.Flags().StringVarP(&ledgerNetwork, "network", "n", "alphanet", "Network to query")
	txCmd.Flags().StringVarP(&txNetwork, "network", "n", "alphanet", "Network to query")
}

func runLedger(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'bedrock init' first)", err)
	}

	networkCfg, ok := cfg.Networks[ledgerNetwork]
	if !ok {
		if ledgerNetwork == "local" {
			networkCfg = config.NetworkConfig{
				URL:       "ws://localhost:6006",
				RPCURL:    "http://localhost:5005",
				NetworkID: 63456,
			}
		} else {
			return fmt.Errorf("network '%s' not found in config", ledgerNetwork)
		}
	}

	client := chain.NewClient(networkCfg.URL, networkCfg.GetRPCURL())
	ctx := cmd.Context()

	ledgerIndex := "current"
	if len(args) > 0 {
		ledgerIndex = args[0]
	}

	color.Cyan("Ledger Info\n")
	fmt.Println()

	info, err := client.GetLedger(ctx, ledgerIndex)
	if err != nil {
		color.Red("Failed to get ledger: %v\n", err)
		return err
	}

	fmt.Printf("  Ledger Index: %s\n", info.Ledger.LedgerIndex)
	fmt.Printf("  Ledger Hash:  %s\n", info.Ledger.LedgerHash)
	fmt.Printf("  Parent Hash:  %s\n", info.Ledger.ParentHash)
	fmt.Printf("  Close Time:   %d\n", info.Ledger.CloseTime)
	fmt.Printf("  Total Coins:  %s\n", info.Ledger.TotalCoins)
	fmt.Printf("  Validated:    %v\n", info.Validated)

	return nil
}

func runTx(cmd *cobra.Command, args []string) error {
	hash := args[0]

	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'bedrock init' first)", err)
	}

	networkCfg, ok := cfg.Networks[txNetwork]
	if !ok {
		if txNetwork == "local" {
			networkCfg = config.NetworkConfig{
				URL:       "ws://localhost:6006",
				RPCURL:    "http://localhost:5005",
				NetworkID: 63456,
			}
		} else {
			return fmt.Errorf("network '%s' not found in config", txNetwork)
		}
	}

	client := chain.NewClient(networkCfg.URL, networkCfg.GetRPCURL())
	ctx := cmd.Context()

	color.Cyan("Transaction Details\n")
	fmt.Printf("  Hash: %s\n\n", hash)

	tx, err := client.GetTransaction(ctx, hash)
	if err != nil {
		color.Red("Failed to get transaction: %v\n", err)
		return err
	}

	fmt.Printf("  Ledger Index: %d\n", tx.LedgerIndex)
	fmt.Printf("  Validated:    %v\n", tx.Validated)

	if tx.Tx != nil {
		fmt.Println()
		color.Cyan("Transaction JSON:\n")
		var parsed interface{}
		if err := json.Unmarshal(tx.Tx, &parsed); err == nil {
			pretty, _ := json.MarshalIndent(parsed, "  ", "  ")
			fmt.Printf("  %s\n", string(pretty))
		}
	}

	if tx.Meta != nil {
		fmt.Println()
		color.Cyan("Metadata:\n")
		pretty, _ := json.MarshalIndent(tx.Meta, "  ", "  ")
		fmt.Printf("  %s\n", string(pretty))
	}

	return nil
}
