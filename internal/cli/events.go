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
	eventsNetwork    string
	eventsType       string
	eventsFromLedger int64
	eventsToLedger   int64
	eventsLimit      int
)

var eventsCmd = &cobra.Command{
	Use:   "events <contract-account>",
	Short: "Query contract events",
	Long: `Query event history for a deployed smart contract.

Examples:
  bedrock events rContract123...
  bedrock events rContract123... --type Transfer
  bedrock events rContract123... --from-ledger 1000 --to-ledger 2000
  bedrock events rContract123... --network local`,
	Args: cobra.ExactArgs(1),
	RunE: runEvents,
}

func init() {
	rootCmd.AddCommand(eventsCmd)

	eventsCmd.Flags().StringVarP(&eventsNetwork, "network", "n", "alphanet", "Network to query")
	eventsCmd.Flags().StringVar(&eventsType, "type", "", "Filter by event type")
	eventsCmd.Flags().Int64Var(&eventsFromLedger, "from-ledger", 0, "Start ledger index")
	eventsCmd.Flags().Int64Var(&eventsToLedger, "to-ledger", 0, "End ledger index")
	eventsCmd.Flags().IntVar(&eventsLimit, "limit", 20, "Maximum number of events to return")
}

func runEvents(cmd *cobra.Command, args []string) error {
	contractAccount := args[0]

	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'bedrock init' first)", err)
	}

	networkCfg, ok := cfg.Networks[eventsNetwork]
	if !ok {
		if eventsNetwork == "local" {
			networkCfg = config.NetworkConfig{
				URL:       "ws://localhost:6006",
				RPCURL:    "http://localhost:5005",
				NetworkID: 63456,
			}
		} else {
			return fmt.Errorf("network '%s' not found in config", eventsNetwork)
		}
	}

	client := chain.NewClient(networkCfg.URL, networkCfg.GetRPCURL())
	ctx := cmd.Context()

	color.Cyan("Contract Events\n")
	fmt.Printf("  Contract: %s\n", contractAccount)
	if eventsType != "" {
		fmt.Printf("  Filter: %s\n", eventsType)
	}
	fmt.Println()

	history, err := client.GetEventHistory(ctx, contractAccount, chain.EventQueryOpts{
		EventType:  eventsType,
		FromLedger: eventsFromLedger,
		ToLedger:   eventsToLedger,
		Limit:      eventsLimit,
	})
	if err != nil {
		color.Red("Failed to query events: %v\n", err)
		return err
	}

	if len(history.Events) == 0 {
		fmt.Println("  No events found")
		return nil
	}

	fmt.Printf("  Found %d events:\n\n", len(history.Events))
	for i, event := range history.Events {
		fmt.Printf("  [%d] Type: %s | Ledger: %d | Tx: %s\n",
			i+1, event.Type, event.LedgerIndex, event.TxHash)

		if event.Data != nil {
			var data interface{}
			if err := json.Unmarshal(event.Data, &data); err == nil {
				pretty, _ := json.MarshalIndent(data, "      ", "  ")
				fmt.Printf("      %s\n", string(pretty))
			}
		}
		fmt.Println()
	}

	return nil
}
