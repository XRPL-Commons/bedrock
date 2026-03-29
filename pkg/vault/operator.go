package vault

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/xrpl-commons/bedrock/pkg/adapter"
)

// Operator handles vault operations via embedded JS modules
type Operator struct {
	executor *adapter.Executor
	verbose  bool
}

// NewOperator creates a new vault operator
func NewOperator(verbose bool) (*Operator, error) {
	executor, err := adapter.NewGroupExecutor("vault", verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault executor: %w", err)
	}
	return &Operator{executor: executor, verbose: verbose}, nil
}

// Deploy creates a smart vault with WASM logic
func (o *Operator) Deploy(ctx context.Context, cfg DeployConfig) (*DeployResult, error) {
	jsConfig := map[string]interface{}{
		"wasm_path":   cfg.WasmPath,
		"asset":       cfg.Asset,
		"network_url": cfg.NetworkURL,
		"network_id":  cfg.NetworkID,
		"algorithm":   cfg.Algorithm,
		"verbose":     o.verbose,
	}

	if cfg.WalletSeed != "" {
		jsConfig["wallet_seed"] = cfg.WalletSeed
	}
	if cfg.FaucetURL != "" {
		jsConfig["faucet_url"] = cfg.FaucetURL
	}
	if cfg.Fee != "" {
		jsConfig["fee"] = cfg.Fee
	}
	if cfg.AssetsMaximum != "" {
		jsConfig["assets_maximum"] = cfg.AssetsMaximum
	}

	result, err := o.executor.ExecuteModule(ctx, "vault_deploy.js", jsConfig)
	if err != nil {
		return nil, fmt.Errorf("vault deploy failed: %w", err)
	}

	var deployResult DeployResult
	if err := json.Unmarshal(result.Data, &deployResult); err != nil {
		return nil, fmt.Errorf("failed to parse deploy result: %w", err)
	}

	return &deployResult, nil
}

// Deposit deposits into a vault
func (o *Operator) Deposit(ctx context.Context, cfg DepositConfig) (*DepositResult, error) {
	jsConfig := map[string]interface{}{
		"vault_id":    cfg.VaultID,
		"amount":      cfg.Amount,
		"network_url": cfg.NetworkURL,
		"network_id":  cfg.NetworkID,
		"wallet_seed": cfg.WalletSeed,
		"algorithm":   cfg.Algorithm,
		"verbose":     o.verbose,
	}

	if cfg.Fee != "" {
		jsConfig["fee"] = cfg.Fee
	}

	result, err := o.executor.ExecuteModule(ctx, "vault_deposit.js", jsConfig)
	if err != nil {
		return nil, fmt.Errorf("vault deposit failed: %w", err)
	}

	var depositResult DepositResult
	if err := json.Unmarshal(result.Data, &depositResult); err != nil {
		return nil, fmt.Errorf("failed to parse deposit result: %w", err)
	}

	return &depositResult, nil
}

// Withdraw withdraws from a vault
func (o *Operator) Withdraw(ctx context.Context, cfg WithdrawConfig) (*WithdrawResult, error) {
	jsConfig := map[string]interface{}{
		"vault_id":    cfg.VaultID,
		"amount":      cfg.Amount,
		"destination": cfg.Destination,
		"network_url": cfg.NetworkURL,
		"network_id":  cfg.NetworkID,
		"wallet_seed": cfg.WalletSeed,
		"algorithm":   cfg.Algorithm,
		"verbose":     o.verbose,
	}

	if cfg.Fee != "" {
		jsConfig["fee"] = cfg.Fee
	}

	result, err := o.executor.ExecuteModule(ctx, "vault_withdraw.js", jsConfig)
	if err != nil {
		return nil, fmt.Errorf("vault withdraw failed: %w", err)
	}

	var withdrawResult WithdrawResult
	if err := json.Unmarshal(result.Data, &withdrawResult); err != nil {
		return nil, fmt.Errorf("failed to parse withdraw result: %w", err)
	}

	return &withdrawResult, nil
}

// Status queries a vault ledger entry
func (o *Operator) Status(ctx context.Context, cfg StatusConfig) (json.RawMessage, error) {
	jsConfig := map[string]interface{}{
		"vault_id":    cfg.VaultID,
		"network_url": cfg.NetworkURL,
		"verbose":     o.verbose,
	}

	result, err := o.executor.ExecuteModule(ctx, "vault_status.js", jsConfig)
	if err != nil {
		return nil, fmt.Errorf("vault status query failed: %w", err)
	}

	return result.Data, nil
}
