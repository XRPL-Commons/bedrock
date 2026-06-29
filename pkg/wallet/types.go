package wallet

import (
	"time"
)

// Keystore represents the encrypted wallet data stored on disk.
// Iterations is the PBKDF2 iteration count used to derive the encryption key;
// older keystores written before this field existed are treated as the legacy
// default (LegacyPBKDF2Iterations).
type Keystore struct {
	Version       int       `json:"version"`
	Name          string    `json:"name"`
	Address       string    `json:"address"`
	CreatedAt     time.Time `json:"created_at"`
	EncryptedSeed string    `json:"encrypted_seed"`
	Salt          string    `json:"salt"`
	Nonce         string    `json:"nonce"`
	Iterations    int       `json:"iterations,omitempty"`
}

// WalletInfo represents wallet information without sensitive data
type WalletInfo struct {
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"created_at"`
}

// Wallet represents a decrypted wallet in memory
type Wallet struct {
	Name    string
	Address string
	Seed    string
}

// WalletManager handles wallet operations
type WalletManager struct {
	walletsDir string
}

const (
	// KeystoreVersion is the current on-disk schema version. Bumped when
	// fields are added or KDF parameters change in an incompatible way.
	KeystoreVersion = 2

	// PBKDF2Iterations is the current iteration count for new keystores.
	// Aligned with OWASP 2023 baseline for PBKDF2-HMAC-SHA256.
	PBKDF2Iterations = 600_000

	// LegacyPBKDF2Iterations is the count used by v1 keystores (no iterations
	// field on disk). Decrypt falls back to this value when Iterations is 0.
	LegacyPBKDF2Iterations = 100_000
)
