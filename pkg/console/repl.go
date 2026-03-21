package console

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/xrpl-commons/bedrock/pkg/abi"
	"github.com/xrpl-commons/bedrock/pkg/caller"
	"github.com/xrpl-commons/bedrock/pkg/chain"
	"github.com/xrpl-commons/bedrock/pkg/config"
)

// REPL provides an interactive console for contract interaction
type REPL struct {
	cfg             *config.Config
	client          *chain.Client
	contractAccount string
	walletSeed      string
	networkCfg      config.NetworkConfig
	abiData         *abi.ABI
	history         []string
}

// NewREPL creates a new interactive console
func NewREPL(cfg *config.Config, contractAccount string, walletSeed string, networkCfg config.NetworkConfig) *REPL {
	return &REPL{
		cfg:             cfg,
		client:          chain.NewClient(networkCfg.URL, networkCfg.GetRPCURL()),
		contractAccount: contractAccount,
		walletSeed:      walletSeed,
		networkCfg:      networkCfg,
	}
}

// Run starts the REPL loop
func (r *REPL) Run(ctx context.Context) error {
	// Try to load ABI for tab completion
	r.loadABI()

	fmt.Println("Bedrock Interactive Console")
	fmt.Printf("  Network: %s\n", r.networkCfg.URL)
	if r.contractAccount != "" {
		fmt.Printf("  Contract: %s\n", r.contractAccount)
	}
	if r.abiData != nil {
		fmt.Printf("  Functions: %d\n", len(r.abiData.Functions))
	}
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  call <function> [params-json]  - Call a contract function")
	fmt.Println("  info                           - Show contract info")
	fmt.Println("  balance [address]              - Check XRP balance")
	fmt.Println("  ledger                         - Current ledger info")
	fmt.Println("  rpc <method> [params-json]     - Raw RPC call")
	fmt.Println("  functions                      - List available functions")
	fmt.Println("  contract <address>             - Switch contract")
	fmt.Println("  history                        - Show command history")
	fmt.Println("  help                           - Show this help")
	fmt.Println("  exit                           - Exit console")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("bedrock> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		r.history = append(r.history, line)

		parts := strings.SplitN(line, " ", 3)
		cmd := parts[0]

		switch cmd {
		case "exit", "quit", "q":
			return nil
		case "help", "?":
			r.showHelp()
		case "call":
			r.handleCall(ctx, parts[1:])
		case "info":
			r.handleInfo(ctx)
		case "balance":
			r.handleBalance(ctx, parts[1:])
		case "ledger":
			r.handleLedger(ctx)
		case "rpc":
			r.handleRPC(ctx, parts[1:])
		case "functions", "funcs":
			r.handleFunctions()
		case "contract":
			r.handleContract(parts[1:])
		case "history":
			r.handleHistory()
		default:
			// Try to interpret as a function call
			if r.isFunction(cmd) {
				r.handleCall(ctx, parts)
			} else {
				fmt.Printf("Unknown command: %s (type 'help' for commands)\n", cmd)
			}
		}

		fmt.Println()
	}

	return nil
}

func (r *REPL) loadABI() {
	abiPath := "abi.json"
	if contracts := r.cfg.Contracts; contracts != nil {
		if main, ok := contracts["main"]; ok {
			abiPath = main.ABI
		}
	}

	data, err := os.ReadFile(abiPath)
	if err != nil {
		return
	}

	var a abi.ABI
	if err := json.Unmarshal(data, &a); err != nil {
		return
	}

	r.abiData = &a
}

func (r *REPL) isFunction(name string) bool {
	if r.abiData == nil {
		return false
	}
	for _, fn := range r.abiData.Functions {
		if fn.Name == name {
			return true
		}
	}
	return false
}

func (r *REPL) handleCall(ctx context.Context, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: call <function> [params-json]")
		return
	}

	if r.contractAccount == "" {
		fmt.Println("No contract set. Use 'contract <address>' first.")
		return
	}

	if r.walletSeed == "" {
		fmt.Println("No wallet set. Start console with --wallet flag.")
		return
	}

	functionName := args[0]
	var params map[string]interface{}

	if len(args) > 1 {
		paramsStr := strings.Join(args[1:], " ")
		if err := json.Unmarshal([]byte(paramsStr), &params); err != nil {
			fmt.Printf("Invalid JSON parameters: %v\n", err)
			return
		}
	}

	c, err := caller.NewCaller(false)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	result, err := c.Call(ctx, caller.CallConfig{
		ContractAccount:      r.contractAccount,
		FunctionName:         functionName,
		NetworkURL:           r.networkCfg.URL,
		NetworkID:            r.networkCfg.NetworkID,
		WalletSeed:           r.walletSeed,
		Algorithm:            "secp256k1",
		ABIPath:              "abi.json",
		Parameters:           params,
		ComputationAllowance: "1000000",
		Fee:                  "1000000",
	})

	if err != nil {
		fmt.Printf("Call failed: %v\n", err)
		return
	}

	fmt.Printf("  Tx: %s\n", result.TxHash)
	fmt.Printf("  Return code: %d\n", result.ReturnCode)
	if result.ReturnValue != "" {
		fmt.Printf("  Return value: %s\n", result.ReturnValue)
	}
	if result.GasUsed > 0 {
		fmt.Printf("  Gas used: %d\n", result.GasUsed)
	}
}

func (r *REPL) handleInfo(ctx context.Context) {
	if r.contractAccount == "" {
		fmt.Println("No contract set. Use 'contract <address>' first.")
		return
	}

	info, err := r.client.GetAccountInfo(ctx, r.contractAccount)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("  Account: %s\n", r.contractAccount)
	fmt.Printf("  Balance: %s XRP\n", chain.DropsToXRP(info.AccountData.Balance))
	fmt.Printf("  Sequence: %d\n", info.AccountData.Sequence)
}

func (r *REPL) handleBalance(ctx context.Context, args []string) {
	address := r.contractAccount
	if len(args) > 0 {
		address = args[0]
	}
	if address == "" {
		fmt.Println("Provide an address or set a contract first.")
		return
	}

	balance, err := r.client.GetAccountBalance(ctx, address)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("  %s XRP\n", balance)
}

func (r *REPL) handleLedger(ctx context.Context) {
	info, err := r.client.GetLedger(ctx, "current")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("  Ledger: %s\n", info.Ledger.LedgerIndex)
	fmt.Printf("  Hash: %s\n", info.Ledger.LedgerHash)
}

func (r *REPL) handleRPC(ctx context.Context, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: rpc <method> [params-json]")
		return
	}

	method := args[0]
	var params []interface{}

	if len(args) > 1 {
		var parsed interface{}
		if err := json.Unmarshal([]byte(args[1]), &parsed); err != nil {
			fmt.Printf("Invalid JSON: %v\n", err)
			return
		}
		params = append(params, parsed)
	}

	result, err := r.client.Call(ctx, method, params...)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	var parsed interface{}
	json.Unmarshal(result, &parsed)
	pretty, _ := json.MarshalIndent(parsed, "  ", "  ")
	fmt.Printf("  %s\n", string(pretty))
}

func (r *REPL) handleFunctions() {
	if r.abiData == nil {
		fmt.Println("No ABI loaded. Build your contract first.")
		return
	}

	for _, fn := range r.abiData.Functions {
		var paramTypes []string
		for _, p := range fn.Parameters {
			paramTypes = append(paramTypes, fmt.Sprintf("%s: %s", p.Name, p.Type))
		}
		retStr := ""
		if fn.Returns != nil {
			retStr = fmt.Sprintf(" -> %s", fn.Returns.Type)
		}
		fmt.Printf("  %s(%s)%s\n", fn.Name, strings.Join(paramTypes, ", "), retStr)
	}
}

func (r *REPL) handleContract(args []string) {
	if len(args) < 1 {
		if r.contractAccount != "" {
			fmt.Printf("  Current: %s\n", r.contractAccount)
		} else {
			fmt.Println("Usage: contract <address>")
		}
		return
	}
	r.contractAccount = args[0]
	fmt.Printf("  Contract set to: %s\n", r.contractAccount)
}

func (r *REPL) handleHistory() {
	for i, cmd := range r.history {
		fmt.Printf("  [%d] %s\n", i+1, cmd)
	}
}

func (r *REPL) showHelp() {
	fmt.Println("Commands:")
	fmt.Println("  call <function> [params-json]  - Call a contract function")
	fmt.Println("  info                           - Show contract info")
	fmt.Println("  balance [address]              - Check XRP balance")
	fmt.Println("  ledger                         - Current ledger info")
	fmt.Println("  rpc <method> [params-json]     - Raw RPC call")
	fmt.Println("  functions                      - List available functions")
	fmt.Println("  contract <address>             - Switch contract")
	fmt.Println("  history                        - Show command history")
	fmt.Println("  exit                           - Exit console")
}
