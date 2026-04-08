package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/xrpl-commons/bedrock/pkg/chain"
	"github.com/xrpl-commons/bedrock/pkg/config"
)

var rpcNetwork string

var rpcCmd = &cobra.Command{
	Use:   "rpc <method> [params-json]",
	Short: "Send raw RPC request",
	Long: `Send an arbitrary JSON-RPC request to the XRPL node.

The first argument is the method name.
The second optional argument is a JSON object of parameters.
Output is always JSON for scripting.

Examples:
  bedrock rpc server_info
  bedrock rpc account_info '{"account":"rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh"}'
  bedrock rpc ledger '{"ledger_index":"current"}' --network local`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runRPC,
}

func init() {
	rootCmd.AddCommand(rpcCmd)

	rpcCmd.Flags().StringVarP(&rpcNetwork, "network", "n", "alphanet", "Network to send RPC to")
}

func runRPC(cmd *cobra.Command, args []string) error {
	method := args[0]

	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'bedrock init' first)", err)
	}

	networkCfg, ok := cfg.Networks[rpcNetwork]
	if !ok {
		if rpcNetwork == "local" {
			networkCfg = config.NetworkConfig{
				URL:       "ws://localhost:6006",
				NetworkID: 100,
			}
		} else {
			return fmt.Errorf("network '%s' not found in config", rpcNetwork)
		}
	}

	client := chain.NewClient(networkCfg.URL)
	ctx := cmd.Context()

	var params []interface{}
	if len(args) > 1 {
		paramStr := strings.TrimSpace(args[1])
		var parsed interface{}
		if err := json.Unmarshal([]byte(paramStr), &parsed); err != nil {
			color.Red("Invalid JSON parameters: %v\n", err)
			return err
		}
		params = append(params, parsed)
	}

	result, err := client.Call(ctx, method, params...)
	if err != nil {
		color.Red("RPC error: %v\n", err)
		return err
	}

	// Pretty-print JSON output
	var parsed interface{}
	if err := json.Unmarshal(result, &parsed); err == nil {
		pretty, _ := json.MarshalIndent(parsed, "", "  ")
		fmt.Println(string(pretty))
	} else {
		fmt.Println(string(result))
	}

	return nil
}
