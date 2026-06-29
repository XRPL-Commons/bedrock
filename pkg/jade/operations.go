package jade

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/Peersyst/xrpl-go/xrpl/queries/account"
	"github.com/Peersyst/xrpl-go/xrpl/queries/common"
	"github.com/Peersyst/xrpl-go/xrpl/queries/server"
	"github.com/Peersyst/xrpl-go/xrpl/rpc"
	rpctypes "github.com/Peersyst/xrpl-go/xrpl/rpc/types"
	"github.com/Peersyst/xrpl-go/xrpl/transaction"
	"github.com/Peersyst/xrpl-go/xrpl/transaction/types"
	"github.com/Peersyst/xrpl-go/xrpl/wallet"
)

// Operations handles XRPL network operations using pure Go
type Operations struct {
	verbose bool
}

// NewOperations creates a new Operations instance
func NewOperations(verbose bool) (*Operations, error) {
	return &Operations{
		verbose: verbose,
	}, nil
}

// BalanceResult represents the result of a balance query
type BalanceResult struct {
	Address      string `json:"address"`
	Balance      string `json:"balance"`
	BalanceDrops string `json:"balance_drops"`
	Funded       bool   `json:"funded,omitempty"`
}

// SendResult represents the result of a send operation
type SendResult struct {
	TxHash      string `json:"tx_hash"`
	From        string `json:"from"`
	To          string `json:"to"`
	Amount      string `json:"amount"`
	AmountDrops string `json:"amount_drops"`
	Result      string `json:"result"`
	Fee         string `json:"fee"`
	Sequence    int    `json:"sequence"`
	Validated   bool   `json:"validated"`
}

// TxResult represents the result of a transaction query
type TxResult struct {
	Hash            string      `json:"hash"`
	Type            string      `json:"type"`
	Account         string      `json:"account"`
	Result          string      `json:"result"`
	Fee             string      `json:"fee"`
	Sequence        int         `json:"sequence"`
	Validated       bool        `json:"validated"`
	LedgerIndex     int         `json:"ledger_index"`
	Date            int         `json:"date,omitempty"`
	Destination     string      `json:"destination,omitempty"`
	Amount          interface{} `json:"amount,omitempty"`
	DeliveredAmount interface{} `json:"delivered_amount,omitempty"`
	ContractAccount string      `json:"contract_account,omitempty"`
	FunctionName    string      `json:"function_name,omitempty"`
	ReturnValue     string      `json:"return_value,omitempty"`
	WasmSize        int         `json:"wasm_size,omitempty"`
}

// AccountInfoResult represents the result of an account_info query
type AccountInfoResult struct {
	Address           string      `json:"address"`
	Balance           interface{} `json:"balance"`
	BalanceDrops      string      `json:"balance_drops"`
	Sequence          int         `json:"sequence"`
	OwnerCount        int         `json:"owner_count"`
	PreviousTxnID     string      `json:"previous_txn_id"`
	PreviousTxnLgrSeq int         `json:"previous_txn_lgr_seq"`
	Flags             int         `json:"flags"`
	LedgerIndex       int         `json:"ledger_index"`
	Funded            bool        `json:"funded,omitempty"`
	Error             string      `json:"error,omitempty"`
}

// GetBalanceString returns the balance as a string
func (a *AccountInfoResult) GetBalanceString() string {
	switch v := a.Balance.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.6f", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ServerInfoResult represents the result of a server_info query
type ServerInfoResult struct {
	BuildVersion    string                 `json:"build_version"`
	CompleteLedgers string                 `json:"complete_ledgers"`
	HostID          string                 `json:"hostid"`
	NetworkID       int                    `json:"network_id"`
	Peers           int                    `json:"peers"`
	PubkeyNode      string                 `json:"pubkey_node"`
	ServerState     string                 `json:"server_state"`
	Uptime          int                    `json:"uptime"`
	ValidatedLedger map[string]interface{} `json:"validated_ledger,omitempty"`
}

// createRPCClient creates an RPC client for the given network URL
func createRPCClient(networkURL string) (*rpc.Client, error) {
	// Convert WebSocket URL to HTTP URL for RPC
	rpcURL := networkURL
	if strings.HasPrefix(networkURL, "ws://") {
		rpcURL = strings.Replace(networkURL, "ws://", "http://", 1)
	} else if strings.HasPrefix(networkURL, "wss://") {
		rpcURL = strings.Replace(networkURL, "wss://", "https://", 1)
	}

	cfg, err := rpc.NewClientConfig(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC client config: %w", err)
	}

	return rpc.NewClient(cfg), nil
}

// dropsToXRP converts drops to XRP string using integer arithmetic for precision
func dropsToXRP(drops string) string {
	dropsBig, ok := new(big.Int).SetString(drops, 10)
	if !ok {
		return "0"
	}

	million := big.NewInt(1_000_000)
	whole := new(big.Int).Div(dropsBig, million)
	remainder := new(big.Int).Mod(dropsBig, million)

	if remainder.Sign() == 0 {
		return whole.String()
	}

	remainderStr := fmt.Sprintf("%06d", remainder.Int64())
	remainderStr = strings.TrimRight(remainderStr, "0")
	return fmt.Sprintf("%s.%s", whole.String(), remainderStr)
}

// xrpToDrops converts XRP string to drops string using big.Float for precision
func xrpToDrops(xrp string) (string, error) {
	xrpBig, _, err := big.ParseFloat(xrp, 10, 128, big.ToNearestEven)
	if err != nil {
		return "", fmt.Errorf("invalid XRP amount: %w", err)
	}

	if xrpBig.Sign() <= 0 {
		return "", fmt.Errorf("amount must be positive")
	}

	multiplier := big.NewFloat(1_000_000)
	dropsBig := new(big.Float).Mul(xrpBig, multiplier)

	dropsInt, _ := dropsBig.Int(nil)
	return dropsInt.String(), nil
}

// GetBalance retrieves the XRP balance for an address
func (o *Operations) GetBalance(networkURL string, address string) (*BalanceResult, error) {
	client, err := createRPCClient(networkURL)
	if err != nil {
		return nil, err
	}

	if o.verbose {
		fmt.Printf("Getting balance for %s from %s...\n", address, networkURL)
	}

	req := &account.InfoRequest{
		Account: types.Address(address),
	}

	resp, err := client.Request(req)
	if err != nil {
		// Check if account not found
		if strings.Contains(err.Error(), "actNotFound") || strings.Contains(err.Error(), "Account not found") {
			return &BalanceResult{
				Address:      address,
				Balance:      "0",
				BalanceDrops: "0",
				Funded:       false,
			}, nil
		}
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}

	var infoResp account.InfoResponse
	if err := resp.GetResult(&infoResp); err != nil {
		// Check if account not found in response
		if strings.Contains(err.Error(), "actNotFound") {
			return &BalanceResult{
				Address:      address,
				Balance:      "0",
				BalanceDrops: "0",
				Funded:       false,
			}, nil
		}
		return nil, fmt.Errorf("failed to parse account info: %w", err)
	}

	balanceDrops := infoResp.AccountData.Balance.String()
	balanceXRP := dropsToXRP(balanceDrops)

	return &BalanceResult{
		Address:      address,
		Balance:      balanceXRP,
		BalanceDrops: balanceDrops,
		Funded:       true,
	}, nil
}

// Send transfers XRP to a destination address
func (o *Operations) Send(networkURL string, walletSeed, destination, amount, algorithm string) (*SendResult, error) {
	client, err := createRPCClient(networkURL)
	if err != nil {
		return nil, err
	}

	// Create wallet from seed
	w, err := wallet.FromSeed(walletSeed, algorithm)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet from seed: %w", err)
	}

	if o.verbose {
		fmt.Printf("Sending %s XRP from %s to %s...\n", amount, w.ClassicAddress, destination)
	}

	// Convert XRP to drops
	amountDrops, err := xrpToDrops(amount)
	if err != nil {
		return nil, err
	}

	// Create payment transaction
	payment := transaction.FlatTransaction{
		"TransactionType": "Payment",
		"Account":         string(w.ClassicAddress),
		"Destination":     destination,
		"Amount":          amountDrops,
	}

	// Submit and wait for validation
	opts := &rpctypes.SubmitOptions{
		Autofill: true,
		Wallet:   &w,
		FailHard: false,
	}

	txResp, err := client.SubmitTxAndWait(payment, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to submit transaction: %w", err)
	}

	// Extract result from meta
	result := "tesSUCCESS"
	if meta, ok := txResp.Meta.(map[string]interface{}); ok {
		if txResult, ok := meta["TransactionResult"].(string); ok {
			result = txResult
		}
	}

	// Get fee from transaction
	fee := ""
	if feeVal, ok := txResp.Tx["Fee"]; ok {
		fee = fmt.Sprintf("%v", feeVal)
	}

	// Get sequence from transaction
	sequence := 0
	if seqVal, ok := txResp.Tx["Sequence"]; ok {
		switch v := seqVal.(type) {
		case float64:
			sequence = int(v)
		case int:
			sequence = v
		}
	}

	return &SendResult{
		TxHash:      string(txResp.Hash),
		From:        string(w.ClassicAddress),
		To:          destination,
		Amount:      amount,
		AmountDrops: amountDrops,
		Result:      result,
		Fee:         fee,
		Sequence:    sequence,
		Validated:   txResp.Validated,
	}, nil
}

// GetTransaction retrieves transaction details by hash.
// Uses raw JSON-RPC instead of xrpl-go's typed parser because xrpl-go
// does not have parsers for contract-related transaction types
// (ContractCreate, ContractCall, ContractModify, ContractDelete).
func (o *Operations) GetTransaction(networkURL string, hash string) (*TxResult, error) {
	if o.verbose {
		fmt.Printf("Fetching transaction %s...\n", hash)
	}

	// Use raw HTTP JSON-RPC to avoid xrpl-go parser limitations
	rpcURL := networkURL
	if strings.HasPrefix(networkURL, "ws://") {
		rpcURL = strings.Replace(networkURL, "ws://", "http://", 1)
	} else if strings.HasPrefix(networkURL, "wss://") {
		rpcURL = strings.Replace(networkURL, "wss://", "https://", 1)
	}

	reqBody := map[string]interface{}{
		"method": "tx",
		"params": []map[string]interface{}{
			{
				"transaction": hash,
				"binary":      false,
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpResp, err := http.Post(rpcURL, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to send tx request: %w", err)
	}
	defer httpResp.Body.Close()

	var rpcResp struct {
		Result map[string]interface{} `json:"result"`
	}
	if err := json.NewDecoder(httpResp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("failed to decode tx response: %w", err)
	}

	res := rpcResp.Result
	if errMsg, ok := res["error"].(string); ok {
		return nil, fmt.Errorf("tx lookup failed: %s", errMsg)
	}

	getString := func(m map[string]interface{}, key string) string {
		if v, ok := m[key]; ok {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}

	getInt := func(m map[string]interface{}, key string) int {
		if v, ok := m[key].(float64); ok {
			return int(v)
		}
		return 0
	}

	txType := getString(res, "TransactionType")

	txResult := &TxResult{
		Hash:        getString(res, "hash"),
		Type:        txType,
		Account:     getString(res, "Account"),
		Fee:         getString(res, "Fee"),
		Sequence:    getInt(res, "Sequence"),
		LedgerIndex: getInt(res, "ledger_index"),
		Date:        getInt(res, "date"),
	}

	if v, ok := res["validated"].(bool); ok {
		txResult.Validated = v
	}

	// Extract result from meta
	txResult.Result = "tesSUCCESS"
	if meta, ok := res["meta"].(map[string]interface{}); ok {
		if r, ok := meta["TransactionResult"].(string); ok {
			txResult.Result = r
		}
	}

	// Type-specific fields
	switch txType {
	case "Payment":
		txResult.Destination = getString(res, "Destination")
		if amt, ok := res["Amount"]; ok {
			txResult.Amount = amt
		}
		if meta, ok := res["meta"].(map[string]interface{}); ok {
			if delivered, ok := meta["delivered_amount"]; ok {
				txResult.DeliveredAmount = delivered
			}
		}

	case "ContractCreate":
		if meta, ok := res["meta"].(map[string]interface{}); ok {
			if contractAcc, ok := meta["ContractAccount"].(string); ok {
				txResult.ContractAccount = contractAcc
			}
		}
		if wasmHex, ok := res["ContractCode"].(string); ok {
			txResult.WasmSize = len(wasmHex) / 2
		}

	case "ContractCall":
		txResult.ContractAccount = getString(res, "ContractAccount")
		txResult.FunctionName = getString(res, "FunctionName")
		if meta, ok := res["meta"].(map[string]interface{}); ok {
			if returnVal, ok := meta["HookReturnString"].(string); ok {
				txResult.ReturnValue = returnVal
			}
		}

	case "ContractModify":
		txResult.ContractAccount = getString(res, "ContractAccount")

	case "ContractDelete":
		txResult.ContractAccount = getString(res, "ContractAccount")
	}

	return txResult, nil
}

// GetAccountInfo retrieves detailed account information
func (o *Operations) GetAccountInfo(networkURL string, address string) (*AccountInfoResult, error) {
	client, err := createRPCClient(networkURL)
	if err != nil {
		return nil, err
	}

	if o.verbose {
		fmt.Printf("Getting account info for %s...\n", address)
	}

	req := &account.InfoRequest{
		Account:     types.Address(address),
		LedgerIndex: common.Validated,
	}

	resp, err := client.Request(req)
	if err != nil {
		// Check if account not found
		if strings.Contains(err.Error(), "actNotFound") || strings.Contains(err.Error(), "Account not found") {
			return &AccountInfoResult{
				Address: address,
				Funded:  false,
				Error:   "Account not found (not funded)",
			}, nil
		}
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}

	var infoResp account.InfoResponse
	if err := resp.GetResult(&infoResp); err != nil {
		if strings.Contains(err.Error(), "actNotFound") {
			return &AccountInfoResult{
				Address: address,
				Funded:  false,
				Error:   "Account not found (not funded)",
			}, nil
		}
		return nil, fmt.Errorf("failed to parse account info: %w", err)
	}

	balanceDrops := infoResp.AccountData.Balance.String()
	balanceXRP := dropsToXRP(balanceDrops)

	return &AccountInfoResult{
		Address:           address,
		Balance:           balanceXRP,
		BalanceDrops:      balanceDrops,
		Sequence:          int(infoResp.AccountData.Sequence),
		OwnerCount:        int(infoResp.AccountData.OwnerCount),
		PreviousTxnID:     string(infoResp.AccountData.PreviousTxnID),
		PreviousTxnLgrSeq: int(infoResp.AccountData.PreviousTxnLgrSeq),
		Flags:             int(infoResp.AccountData.Flags),
		LedgerIndex:       int(infoResp.LedgerIndex),
		Funded:            true,
	}, nil
}

// GetServerInfo retrieves XRPL server information
func (o *Operations) GetServerInfo(networkURL string) (*ServerInfoResult, error) {
	client, err := createRPCClient(networkURL)
	if err != nil {
		return nil, err
	}

	if o.verbose {
		fmt.Printf("Getting server info from %s...\n", networkURL)
	}

	req := &server.InfoRequest{}

	resp, err := client.Request(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}

	var infoResp server.InfoResponse
	if err := resp.GetResult(&infoResp); err != nil {
		return nil, fmt.Errorf("failed to parse server info: %w", err)
	}

	result := &ServerInfoResult{
		BuildVersion:    infoResp.Info.BuildVersion,
		CompleteLedgers: infoResp.Info.CompleteLedgers,
		HostID:          infoResp.Info.HostID,
		NetworkID:       int(infoResp.Info.NetworkID),
		Peers:           int(infoResp.Info.Peers),
		PubkeyNode:      infoResp.Info.PubkeyNode,
		ServerState:     infoResp.Info.ServerState,
		Uptime:          int(infoResp.Info.Uptime),
	}

	// Add validated ledger info if available
	if infoResp.Info.ValidatedLedger.Seq > 0 {
		result.ValidatedLedger = map[string]interface{}{
			"hash": string(infoResp.Info.ValidatedLedger.Hash),
			"seq":  float64(infoResp.Info.ValidatedLedger.Seq),
			"age":  float64(infoResp.Info.ValidatedLedger.Age),
		}
	}

	return result, nil
}
