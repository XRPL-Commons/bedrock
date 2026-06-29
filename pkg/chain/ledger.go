package chain

import (
	"context"
	"encoding/json"
	"fmt"
)

// LedgerInfo represents the result of a ledger RPC call
type LedgerInfo struct {
	Ledger      LedgerData `json:"ledger"`
	LedgerIndex int64      `json:"ledger_index"`
	Status      string     `json:"status"`
	Validated   bool       `json:"validated"`
}

// LedgerData contains ledger details
type LedgerData struct {
	LedgerIndex string `json:"ledger_index"`
	LedgerHash  string `json:"ledger_hash"`
	CloseTime   int64  `json:"close_time"`
	ParentHash  string `json:"parent_hash"`
	TotalCoins  string `json:"total_coins"`
	TxCount     int    `json:"transaction_count"`
	AccountHash string `json:"account_hash"`
	TxHash      string `json:"transaction_hash"`
}

// TransactionInfo represents the result of tx RPC call
type TransactionInfo struct {
	Hash        string                 `json:"hash"`
	Status      string                 `json:"status"`
	Validated   bool                   `json:"validated"`
	LedgerIndex int64                  `json:"ledger_index"`
	Meta        map[string]interface{} `json:"meta"`
	Tx          json.RawMessage        `json:"tx_json"`
}

// GetLedger retrieves the current or specific ledger
func (c *Client) GetLedger(ctx context.Context, ledgerIndex string) (*LedgerInfo, error) {
	params := map[string]interface{}{
		"ledger_index": ledgerIndex,
		"transactions": false,
	}

	var result LedgerInfo
	if err := c.CallTyped(ctx, &result, "ledger", params); err != nil {
		return nil, fmt.Errorf("ledger failed: %w", err)
	}

	return &result, nil
}

// GetTransaction retrieves a transaction by hash
func (c *Client) GetTransaction(ctx context.Context, hash string) (*TransactionInfo, error) {
	params := map[string]interface{}{
		"transaction": hash,
	}

	var result TransactionInfo
	if err := c.CallTyped(ctx, &result, "tx", params); err != nil {
		return nil, fmt.Errorf("tx failed: %w", err)
	}

	return &result, nil
}
