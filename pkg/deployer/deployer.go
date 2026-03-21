package deployer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/xrpl-commons/bedrock/pkg/adapter"
)

// Deployer handles contract deployment via embedded Node.js module
type Deployer struct {
	executor *adapter.Executor
	verbose  bool
}

// NewDeployer creates a new deployer instance
func NewDeployer(verbose bool) (*Deployer, error) {
	executor, err := adapter.NewExecutor(verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	return &Deployer{
		executor: executor,
		verbose:  verbose,
	}, nil
}

// Deploy deploys a contract to the specified network
func (d *Deployer) Deploy(ctx context.Context, config DeploymentConfig) (*DeploymentResult, error) {
	// Build JSON config for deploy.js module
	jsConfig := map[string]interface{}{
		"wasm_path":   config.WasmPath,
		"abi_path":    config.ABIPath,
		"network_url": config.NetworkURL,
		"network_id":  config.NetworkID,
		"algorithm":   config.Algorithm,
		"verbose":     d.verbose,
	}

	if config.RPCURL != "" {
		jsConfig["rpc_url"] = config.RPCURL
	}

	if config.WalletSeed != "" {
		jsConfig["wallet_seed"] = config.WalletSeed
	}

	if config.FaucetURL != "" {
		jsConfig["faucet_url"] = config.FaucetURL
	}

	if config.Fee != "" {
		jsConfig["fee"] = config.Fee
	}

	if config.Immutable {
		jsConfig["immutable"] = true
	}
	if config.CodeImmutable {
		jsConfig["code_immutable"] = true
	}
	if config.ABIImmutable {
		jsConfig["abi_immutable"] = true
	}
	if config.Undeletable {
		jsConfig["undeletable"] = true
	}
	if config.ReuseCode != "" {
		jsConfig["reuse_code"] = config.ReuseCode
	}
	if config.Params != "" {
		jsConfig["params"] = config.Params
	}
	if config.Owner != "" {
		jsConfig["owner"] = config.Owner
	}

	// Execute deploy.js module
	result, err := d.executor.ExecuteModule(ctx, "deploy.js", jsConfig)
	if err != nil {
		return nil, fmt.Errorf("deployment failed: %w", err)
	}

	// Parse deployment result from module output
	var deployResult DeploymentResult
	if err := json.Unmarshal(result.Data, &deployResult); err != nil {
		return nil, fmt.Errorf("failed to parse deployment result: %w", err)
	}

	return &deployResult, nil
}
