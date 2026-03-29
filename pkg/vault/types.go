package vault

// DeployConfig configures vault deployment
type DeployConfig struct {
	WasmPath      string
	Asset         interface{} // e.g. { currency: "XRP" } or { currency: "USD", issuer: "..." }
	AssetsMaximum string      // optional max capacity
	NetworkURL    string
	NetworkID     uint32
	WalletSeed    string
	FaucetURL     string
	Fee           string
	Algorithm     string
}

// DeployResult contains vault creation results
type DeployResult struct {
	TxHash        string `json:"txHash"`
	WalletAddress string `json:"walletAddress"`
	WalletSeed    string `json:"walletSeed"`
	VaultID       string `json:"vaultId"`
	Validated     bool   `json:"validated"`
}

// DepositConfig configures a vault deposit
type DepositConfig struct {
	VaultID    string
	Amount     string // in drops for XRP
	NetworkURL string
	NetworkID  uint32
	WalletSeed string
	Fee        string
	Algorithm  string
}

// DepositResult contains deposit results
type DepositResult struct {
	TxHash    string `json:"txHash"`
	Validated bool   `json:"validated"`
}

// WithdrawConfig configures a vault withdrawal
type WithdrawConfig struct {
	VaultID     string
	Amount      string
	Destination string
	NetworkURL  string
	NetworkID   uint32
	WalletSeed  string
	Fee         string
	Algorithm   string
}

// WithdrawResult contains withdrawal results
type WithdrawResult struct {
	TxHash    string `json:"txHash"`
	Validated bool   `json:"validated"`
}

// StatusConfig configures a vault status query
type StatusConfig struct {
	VaultID    string
	NetworkURL string
}
