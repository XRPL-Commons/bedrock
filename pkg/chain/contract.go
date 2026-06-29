package chain

import (
	"context"
	"encoding/json"
	"fmt"
)

// ContractInfo represents contract details from the ledger
type ContractInfo struct {
	Account         string                 `json:"Account"`
	Owner           string                 `json:"Owner"`
	LedgerEntryType string                 `json:"LedgerEntryType"`
	Flags           int64                  `json:"Flags"`
	ContractData    json.RawMessage        `json:"ContractData"`
	ABI             json.RawMessage        `json:"ABI"`
	WasmHash        string                 `json:"WasmHash"`
	Extra           map[string]interface{} `json:"-"`
}

// GetContractInfo retrieves contract information from the ledger
func (c *Client) GetContractInfo(ctx context.Context, account string) (*ContractInfo, error) {
	// Try account_objects first to find Contract ledger entries
	params := map[string]interface{}{
		"account":      account,
		"ledger_index": "current",
		"type":         "contract",
	}

	var result struct {
		AccountObjects []json.RawMessage `json:"account_objects"`
		Status         string            `json:"status"`
	}

	if err := c.CallTyped(ctx, &result, "account_objects", params); err != nil {
		return nil, fmt.Errorf("failed to get contract info: %w", err)
	}

	if len(result.AccountObjects) == 0 {
		// Try ledger_entry directly
		return c.getContractByLedgerEntry(ctx, account)
	}

	var info ContractInfo
	if err := json.Unmarshal(result.AccountObjects[0], &info); err != nil {
		return nil, fmt.Errorf("failed to parse contract info: %w", err)
	}

	// Capture all fields
	_ = json.Unmarshal(result.AccountObjects[0], &info.Extra)

	return &info, nil
}

func (c *Client) getContractByLedgerEntry(ctx context.Context, account string) (*ContractInfo, error) {
	params := map[string]interface{}{
		"contract":     account,
		"ledger_index": "current",
	}

	raw, err := c.Call(ctx, "ledger_entry", params)
	if err != nil {
		return nil, fmt.Errorf("ledger_entry failed: %w", err)
	}

	var wrapper struct {
		Node json.RawMessage `json:"node"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse ledger_entry: %w", err)
	}

	var info ContractInfo
	if err := json.Unmarshal(wrapper.Node, &info); err != nil {
		return nil, fmt.Errorf("failed to parse contract node: %w", err)
	}

	_ = json.Unmarshal(wrapper.Node, &info.Extra)
	return &info, nil
}

// GetContractData retrieves contract state data
func (c *Client) GetContractData(ctx context.Context, account string, user string) (json.RawMessage, error) {
	params := map[string]interface{}{
		"account":      account,
		"ledger_index": "current",
		"type":         "contract_data",
	}

	if user != "" {
		params["account"] = user
	}

	var result struct {
		AccountObjects []json.RawMessage `json:"account_objects"`
	}

	if err := c.CallTyped(ctx, &result, "account_objects", params); err != nil {
		return nil, fmt.Errorf("failed to get contract data: %w", err)
	}

	if len(result.AccountObjects) == 0 {
		return nil, nil
	}

	return result.AccountObjects[0], nil
}
