package caller

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/xrpl-commons/bedrock/pkg/adapter"
)

// Caller handles contract function calls via embedded Node.js module
type Caller struct {
	executor *adapter.Executor
	verbose  bool
}

// NewCaller creates a new caller instance
func NewCaller(verbose bool) (*Caller, error) {
	executor, err := adapter.NewExecutor(verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	return &Caller{
		executor: executor,
		verbose:  verbose,
	}, nil
}

// Call invokes a contract function
func (c *Caller) Call(ctx context.Context, config CallConfig) (*CallResult, error) {
	// Build JSON config for call.js module
	jsConfig := map[string]interface{}{
		"contract_account": config.ContractAccount,
		"function_name":    config.FunctionName,
		"network_url":      config.NetworkURL,
		"network_id":       config.NetworkID,
		"wallet_seed":      config.WalletSeed,
		"algorithm":        config.Algorithm,
		"verbose":          c.verbose,
	}

	if config.RPCURL != "" {
		jsConfig["rpc_url"] = config.RPCURL
	}

	if config.ABIPath != "" {
		jsConfig["abi_path"] = config.ABIPath
	}

	if config.Parameters != nil {
		jsConfig["parameters"] = config.Parameters
	}

	if config.ComputationAllowance != "" {
		jsConfig["computation_allowance"] = config.ComputationAllowance
	}

	if config.Fee != "" {
		jsConfig["fee"] = config.Fee
	}

	// Execute call.js module
	result, err := c.executor.ExecuteModule(ctx, "call.js", jsConfig)
	if err != nil {
		return nil, fmt.Errorf("contract call failed: %w", err)
	}

	// Parse call result from module output
	var callResult CallResult
	if err := json.Unmarshal(result.Data, &callResult); err != nil {
		return nil, fmt.Errorf("failed to parse call result: %w", err)
	}

	return &callResult, nil
}
