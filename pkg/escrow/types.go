package escrow

// DeployConfig configures escrow deployment
type DeployConfig struct {
	WasmPath    string
	Destination string
	Amount      string // in drops
	CancelAfter int64  // ripple epoch timestamp (optional)
	FinishAfter int64  // ripple epoch timestamp (optional)
	NetworkURL  string
	NetworkID   uint32
	WalletSeed  string
	FaucetURL   string
	Fee         string
	Algorithm   string
}

// DeployResult contains deployment results
type DeployResult struct {
	TxHash         string `json:"txHash"`
	WalletAddress  string `json:"walletAddress"`
	WalletSeed     string `json:"walletSeed"`
	EscrowSequence int    `json:"escrowSequence"`
	Validated      bool   `json:"validated"`
}

// FinishConfig configures an escrow finish operation
type FinishConfig struct {
	Owner          string
	EscrowSequence int
	NetworkURL     string
	NetworkID      uint32
	WalletSeed     string
	Fee            string
	Algorithm      string
}

// FinishResult contains finish results
type FinishResult struct {
	TxHash     string `json:"txHash"`
	ReturnCode int    `json:"returnCode"`
	Validated  bool   `json:"validated"`
}

// CancelConfig configures an escrow cancel operation
type CancelConfig struct {
	Owner          string
	EscrowSequence int
	NetworkURL     string
	NetworkID      uint32
	WalletSeed     string
	Fee            string
	Algorithm      string
}

// CancelResult contains cancel results
type CancelResult struct {
	TxHash    string `json:"txHash"`
	Validated bool   `json:"validated"`
}

// StatusConfig configures an escrow status query
type StatusConfig struct {
	Owner          string
	EscrowSequence int
	NetworkURL     string
}
