package escrow

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/xrpl-commons/bedrock/pkg/adapter"
)

// Operator handles escrow operations via embedded JS modules
type Operator struct {
	executor *adapter.Executor
	verbose  bool
}

// NewOperator creates a new escrow operator
func NewOperator(verbose bool) (*Operator, error) {
	executor, err := adapter.NewGroupExecutor("escrow", verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to create escrow executor: %w", err)
	}
	return &Operator{executor: executor, verbose: verbose}, nil
}

// Deploy creates a smart escrow with WASM condition
func (o *Operator) Deploy(ctx context.Context, cfg DeployConfig) (*DeployResult, error) {
	jsConfig := map[string]interface{}{
		"wasm_path":   cfg.WasmPath,
		"destination": cfg.Destination,
		"amount":      cfg.Amount,
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
	if cfg.CancelAfter > 0 {
		jsConfig["cancel_after"] = cfg.CancelAfter
	}
	if cfg.FinishAfter > 0 {
		jsConfig["finish_after"] = cfg.FinishAfter
	}

	result, err := o.executor.ExecuteModule(ctx, "escrow_deploy.js", jsConfig)
	if err != nil {
		return nil, fmt.Errorf("escrow deploy failed: %w", err)
	}

	var deployResult DeployResult
	if err := json.Unmarshal(result.Data, &deployResult); err != nil {
		return nil, fmt.Errorf("failed to parse deploy result: %w", err)
	}

	return &deployResult, nil
}

// Finish finishes (releases) a smart escrow
func (o *Operator) Finish(ctx context.Context, cfg FinishConfig) (*FinishResult, error) {
	jsConfig := map[string]interface{}{
		"owner":           cfg.Owner,
		"escrow_sequence": cfg.EscrowSequence,
		"network_url":     cfg.NetworkURL,
		"network_id":      cfg.NetworkID,
		"wallet_seed":     cfg.WalletSeed,
		"algorithm":       cfg.Algorithm,
		"verbose":         o.verbose,
	}

	if cfg.Fee != "" {
		jsConfig["fee"] = cfg.Fee
	}

	result, err := o.executor.ExecuteModule(ctx, "escrow_finish.js", jsConfig)
	if err != nil {
		return nil, fmt.Errorf("escrow finish failed: %w", err)
	}

	var finishResult FinishResult
	if err := json.Unmarshal(result.Data, &finishResult); err != nil {
		return nil, fmt.Errorf("failed to parse finish result: %w", err)
	}

	return &finishResult, nil
}

// Cancel cancels a smart escrow
func (o *Operator) Cancel(ctx context.Context, cfg CancelConfig) (*CancelResult, error) {
	jsConfig := map[string]interface{}{
		"owner":           cfg.Owner,
		"escrow_sequence": cfg.EscrowSequence,
		"network_url":     cfg.NetworkURL,
		"network_id":      cfg.NetworkID,
		"wallet_seed":     cfg.WalletSeed,
		"algorithm":       cfg.Algorithm,
		"verbose":         o.verbose,
	}

	if cfg.Fee != "" {
		jsConfig["fee"] = cfg.Fee
	}

	result, err := o.executor.ExecuteModule(ctx, "escrow_cancel.js", jsConfig)
	if err != nil {
		return nil, fmt.Errorf("escrow cancel failed: %w", err)
	}

	var cancelResult CancelResult
	if err := json.Unmarshal(result.Data, &cancelResult); err != nil {
		return nil, fmt.Errorf("failed to parse cancel result: %w", err)
	}

	return &cancelResult, nil
}

// Status queries an escrow ledger entry
func (o *Operator) Status(ctx context.Context, cfg StatusConfig) (json.RawMessage, error) {
	jsConfig := map[string]interface{}{
		"owner":           cfg.Owner,
		"escrow_sequence": cfg.EscrowSequence,
		"network_url":     cfg.NetworkURL,
		"verbose":         o.verbose,
	}

	result, err := o.executor.ExecuteModule(ctx, "escrow_status.js", jsConfig)
	if err != nil {
		return nil, fmt.Errorf("escrow status query failed: %w", err)
	}

	return result.Data, nil
}
