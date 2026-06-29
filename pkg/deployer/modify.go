package deployer

import (
	"context"
	"encoding/json"
	"fmt"
)

// ModifyConfig holds configuration for modifying a contract
type ModifyConfig struct {
	ContractAccount string
	NetworkURL      string
	WalletSeed      string
	Algorithm       string
	WasmPath        string // Optional: new WASM code
	ABIPath         string // Optional: new ABI
	Fee             string
	Owner           string // Optional: new contract owner
	ContractHash    string // Optional: reference existing ContractSource by hash
	Immutable       bool
	CodeImmutable   bool
	ABIImmutable    bool
	Undeletable     bool
}

// ModifyResult represents the result of a contract modification
type ModifyResult struct {
	TxHash    string                 `json:"txHash"`
	Validated bool                   `json:"validated"`
	Meta      map[string]interface{} `json:"meta"`
}

// DeleteConfig holds configuration for deleting a contract
type DeleteConfig struct {
	ContractAccount string
	NetworkURL      string
	WalletSeed      string
	Algorithm       string
	Fee             string
}

// DeleteResult represents the result of a contract deletion
type DeleteResult struct {
	TxHash    string                 `json:"txHash"`
	Validated bool                   `json:"validated"`
	Meta      map[string]interface{} `json:"meta"`
}

// UserDeleteConfig holds configuration for deleting user data from a contract
type UserDeleteConfig struct {
	ContractAccount      string
	FunctionName         string
	ComputationAllowance string
	NetworkURL           string
	WalletSeed           string
	Algorithm            string
	Fee                  string
}

// ClawbackConfig holds configuration for clawing back tokens from a contract
type ClawbackConfig struct {
	ContractAccount string
	Amount          string
	NetworkURL      string
	WalletSeed      string
	Algorithm       string
	Fee             string
}

// ClawbackResult represents the result of a contract clawback
type ClawbackResult struct {
	TxHash    string                 `json:"txHash"`
	Validated bool                   `json:"validated"`
	Meta      map[string]interface{} `json:"meta"`
}

// Modify updates a deployed contract's code or ABI
func (d *Deployer) Modify(ctx context.Context, config ModifyConfig) (*ModifyResult, error) {
	jsConfig := map[string]interface{}{
		"contract_account": config.ContractAccount,
		"network_url":      config.NetworkURL,
		"wallet_seed":      config.WalletSeed,
		"algorithm":        config.Algorithm,
		"fee":              config.Fee,
		"verbose":          d.verbose,
	}

	if config.WasmPath != "" {
		jsConfig["wasm_path"] = config.WasmPath
	}
	if config.ABIPath != "" {
		jsConfig["abi_path"] = config.ABIPath
	}
	if config.Owner != "" {
		jsConfig["owner"] = config.Owner
	}
	if config.ContractHash != "" {
		jsConfig["contract_hash"] = config.ContractHash
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

	result, err := d.executor.ExecuteModule(ctx, "modify.js", jsConfig)
	if err != nil {
		return nil, fmt.Errorf("contract modification failed: %w", err)
	}

	var modifyResult ModifyResult
	if err := json.Unmarshal(result.Data, &modifyResult); err != nil {
		return nil, fmt.Errorf("failed to parse modify result: %w", err)
	}

	return &modifyResult, nil
}

// Delete removes a deployed contract from the ledger
func (d *Deployer) Delete(ctx context.Context, config DeleteConfig) (*DeleteResult, error) {
	jsConfig := map[string]interface{}{
		"contract_account": config.ContractAccount,
		"network_url":      config.NetworkURL,
		"wallet_seed":      config.WalletSeed,
		"algorithm":        config.Algorithm,
		"fee":              config.Fee,
		"verbose":          d.verbose,
	}

	result, err := d.executor.ExecuteModule(ctx, "delete.js", jsConfig)
	if err != nil {
		return nil, fmt.Errorf("contract deletion failed: %w", err)
	}

	var deleteResult DeleteResult
	if err := json.Unmarshal(result.Data, &deleteResult); err != nil {
		return nil, fmt.Errorf("failed to parse delete result: %w", err)
	}

	return &deleteResult, nil
}

// Clawback reclaims tokens from a contract (issuer only)
func (d *Deployer) Clawback(ctx context.Context, config ClawbackConfig) (*ClawbackResult, error) {
	jsConfig := map[string]interface{}{
		"contract_account": config.ContractAccount,
		"amount":           config.Amount,
		"network_url":      config.NetworkURL,
		"wallet_seed":      config.WalletSeed,
		"algorithm":        config.Algorithm,
		"fee":              config.Fee,
		"verbose":          d.verbose,
	}

	result, err := d.executor.ExecuteModule(ctx, "clawback.js", jsConfig)
	if err != nil {
		return nil, fmt.Errorf("contract clawback failed: %w", err)
	}

	var clawbackResult ClawbackResult
	if err := json.Unmarshal(result.Data, &clawbackResult); err != nil {
		return nil, fmt.Errorf("failed to parse clawback result: %w", err)
	}

	return &clawbackResult, nil
}

// UserDelete removes user's data from a contract and recovers reserves
func (d *Deployer) UserDelete(ctx context.Context, config UserDeleteConfig) (*DeleteResult, error) {
	jsConfig := map[string]interface{}{
		"contract_account":      config.ContractAccount,
		"function_name":         config.FunctionName,
		"computation_allowance": config.ComputationAllowance,
		"network_url":           config.NetworkURL,
		"wallet_seed":           config.WalletSeed,
		"algorithm":             config.Algorithm,
		"verbose":               d.verbose,
	}
	// Forward an explicit fee only when set; otherwise user_delete.js derives a
	// gas-aware default (computation_allowance + 0.1 XRP) to avoid telINSUF_FEE_P.
	if config.Fee != "" {
		jsConfig["fee"] = config.Fee
	}

	result, err := d.executor.ExecuteModule(ctx, "user_delete.js", jsConfig)
	if err != nil {
		return nil, fmt.Errorf("user data deletion failed: %w", err)
	}

	var deleteResult DeleteResult
	if err := json.Unmarshal(result.Data, &deleteResult); err != nil {
		return nil, fmt.Errorf("failed to parse user delete result: %w", err)
	}

	return &deleteResult, nil
}
