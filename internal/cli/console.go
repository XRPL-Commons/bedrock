package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/xrpl-commons/bedrock/pkg/config"
	"github.com/xrpl-commons/bedrock/pkg/console"
	"github.com/xrpl-commons/bedrock/pkg/wallet"
)

var (
	consoleNetwork  string
	consoleContract string
	consoleWallet   string
)

var consoleCmd = &cobra.Command{
	Use:   "console",
	Short: "Interactive contract console",
	Long: `Start an interactive console for contract interaction.

Connect to a network, execute contract calls, query state,
and send raw RPC requests interactively.

Examples:
  bedrock console
  bedrock console --network local --contract rContract123... --wallet sXXX...
  bedrock console --network alphanet`,
	RunE: runConsole,
}

func init() {
	rootCmd.AddCommand(consoleCmd)

	consoleCmd.Flags().StringVarP(&consoleNetwork, "network", "n", "local", "Network to connect to")
	consoleCmd.Flags().StringVarP(&consoleContract, "contract", "c", "", "Contract account to interact with")
	consoleCmd.Flags().StringVarP(&consoleWallet, "wallet", "w", "", "Wallet seed or name")
}

func runConsole(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'bedrock init' first)", err)
	}

	networkCfg, ok := cfg.Networks[consoleNetwork]
	if !ok {
		if consoleNetwork == "local" {
			networkCfg = config.NetworkConfig{
				URL:       "ws://localhost:6006",
				RPCURL:    "http://localhost:5005",
				NetworkID: 63456,
			}
		} else {
			return fmt.Errorf("network '%s' not found in config", consoleNetwork)
		}
	}

	// Resolve wallet if provided
	var walletSeed string
	if consoleWallet != "" {
		resolver, err := wallet.NewWalletResolver()
		if err != nil {
			return fmt.Errorf("failed to initialize wallet resolver: %w", err)
		}
		walletSeed, err = resolver.ResolveWallet(consoleWallet)
		if err != nil {
			return fmt.Errorf("failed to resolve wallet: %w", err)
		}
	}

	color.Cyan("Starting interactive console\n\n")

	repl := console.NewREPL(cfg, consoleContract, walletSeed, networkCfg)
	ctx := cmd.Context()

	return repl.Run(ctx)
}
