package chain

import (
	"context"
	"encoding/json"
	"fmt"
)

// AccountInfo represents the result of account_info RPC
type AccountInfo struct {
	AccountData AccountData `json:"account_data"`
	LedgerIndex int64       `json:"ledger_current_index"`
	Status      string      `json:"status"`
	Validated   bool        `json:"validated"`
}

// AccountData contains the account's ledger data
type AccountData struct {
	Account       string `json:"Account"`
	Balance       string `json:"Balance"`
	Flags         int64  `json:"Flags"`
	LedgerIndex   string `json:"index"`
	OwnerCount    int    `json:"OwnerCount"`
	Sequence      int    `json:"Sequence"`
	PreviousTxnID string `json:"PreviousTxnID"`
}

// AccountObjects represents the result of account_objects RPC
type AccountObjects struct {
	Account     string            `json:"account"`
	Objects     []json.RawMessage `json:"account_objects"`
	LedgerIndex int64             `json:"ledger_current_index"`
	Status      string            `json:"status"`
}

// AccountLines represents the result of account_lines RPC
type AccountLines struct {
	Account string      `json:"account"`
	Lines   []TrustLine `json:"lines"`
	Status  string      `json:"status"`
}

// TrustLine represents an XRPL trust line
type TrustLine struct {
	Account   string `json:"account"`
	Balance   string `json:"balance"`
	Currency  string `json:"currency"`
	Limit     string `json:"limit"`
	LimitPeer string `json:"limit_peer"`
	NoRipple  bool   `json:"no_ripple"`
}

// GetAccountInfo retrieves account info from the ledger
func (c *Client) GetAccountInfo(ctx context.Context, address string) (*AccountInfo, error) {
	params := map[string]interface{}{
		"account":      address,
		"ledger_index": "current",
	}

	var result AccountInfo
	if err := c.CallTyped(ctx, &result, "account_info", params); err != nil {
		return nil, fmt.Errorf("account_info failed: %w", err)
	}

	return &result, nil
}

// GetAccountBalance retrieves the XRP balance for an account
func (c *Client) GetAccountBalance(ctx context.Context, address string) (string, error) {
	info, err := c.GetAccountInfo(ctx, address)
	if err != nil {
		return "", err
	}
	return DropsToXRP(info.AccountData.Balance), nil
}

// GetAccountObjects retrieves objects owned by an account
func (c *Client) GetAccountObjects(ctx context.Context, address string) (*AccountObjects, error) {
	params := map[string]interface{}{
		"account":      address,
		"ledger_index": "current",
	}

	var result AccountObjects
	if err := c.CallTyped(ctx, &result, "account_objects", params); err != nil {
		return nil, fmt.Errorf("account_objects failed: %w", err)
	}

	return &result, nil
}

// GetAccountLines retrieves trust lines for an account
func (c *Client) GetAccountLines(ctx context.Context, address string) (*AccountLines, error) {
	params := map[string]interface{}{
		"account":      address,
		"ledger_index": "current",
	}

	var result AccountLines
	if err := c.CallTyped(ctx, &result, "account_lines", params); err != nil {
		return nil, fmt.Errorf("account_lines failed: %w", err)
	}

	return &result, nil
}

// DropsToXRP converts drops (smallest XRP unit) to XRP
func DropsToXRP(drops string) string {
	var d int64
	_, _ = fmt.Sscanf(drops, "%d", &d)
	xrp := float64(d) / 1_000_000
	return fmt.Sprintf("%.6f", xrp)
}
