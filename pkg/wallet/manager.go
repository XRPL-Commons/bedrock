package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// NewWalletManager creates a new wallet manager
func NewWalletManager() (*WalletManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	walletsDir := filepath.Join(homeDir, ".config", "bedrock", "wallets")
	if err := os.MkdirAll(walletsDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create wallets directory: %w", err)
	}

	return &WalletManager{
		walletsDir: walletsDir,
	}, nil
}

// ValidateWalletName checks if wallet name is valid
func (wm *WalletManager) ValidateWalletName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("wallet name cannot be empty")
	}
	if len(name) > 15 {
		return fmt.Errorf("wallet name cannot exceed 15 characters")
	}

	// Only allow alphanumeric characters, hyphens, and underscores
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("wallet name can only contain letters, numbers, hyphens, and underscores")
	}

	return nil
}

// WalletExists checks if a wallet with the given name already exists
func (wm *WalletManager) WalletExists(name string) bool {
	keystorePath := filepath.Join(wm.walletsDir, name+".json")
	_, err := os.Stat(keystorePath)
	return err == nil
}

// ListWallets returns information about all stored wallets
func (wm *WalletManager) ListWallets() ([]WalletInfo, error) {
	files, err := os.ReadDir(wm.walletsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read wallets directory: %w", err)
	}

	var wallets []WalletInfo
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		keystorePath := filepath.Join(wm.walletsDir, file.Name())
		keystore, err := wm.loadKeystore(keystorePath)
		if err != nil {
			continue // Skip corrupted files
		}

		wallets = append(wallets, WalletInfo{
			Name:      keystore.Name,
			Address:   keystore.Address,
			CreatedAt: keystore.CreatedAt,
		})
	}

	return wallets, nil
}

// SaveWallet encrypts and saves a wallet to disk
func (wm *WalletManager) SaveWallet(wallet *Wallet, password string) error {
	if err := wm.ValidateWalletName(wallet.Name); err != nil {
		return err
	}

	if wm.WalletExists(wallet.Name) {
		return fmt.Errorf("wallet '%s' already exists", wallet.Name)
	}

	// Generate salt and derive key
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	key := pbkdf2.Key([]byte(password), salt, PBKDF2Iterations, 32, sha256.New)

	// Encrypt the seed
	encryptedSeed, nonce, err := wm.encrypt([]byte(wallet.Seed), key)
	if err != nil {
		return fmt.Errorf("failed to encrypt seed: %w", err)
	}

	// Create keystore
	keystore := &Keystore{
		Version:       KeystoreVersion,
		Name:          wallet.Name,
		Address:       wallet.Address,
		CreatedAt:     time.Now(),
		EncryptedSeed: hex.EncodeToString(encryptedSeed),
		Salt:          hex.EncodeToString(salt),
		Nonce:         hex.EncodeToString(nonce),
		Iterations:    PBKDF2Iterations,
	}

	// Save to disk
	keystorePath := filepath.Join(wm.walletsDir, wallet.Name+".json")
	return wm.saveKeystore(keystore, keystorePath)
}

// LoadWallet decrypts and loads a wallet from disk
func (wm *WalletManager) LoadWallet(name string, password string) (*Wallet, error) {
	keystorePath := filepath.Join(wm.walletsDir, name+".json")
	keystore, err := wm.loadKeystore(keystorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load wallet '%s': %w", name, err)
	}

	// Derive key from password and salt
	salt, err := hex.DecodeString(keystore.Salt)
	if err != nil {
		return nil, fmt.Errorf("invalid salt in keystore: %w", err)
	}

	iterations := keystore.Iterations
	if iterations <= 0 {
		// Pre-v2 keystores omit the field; they were always written with the
		// legacy iteration count.
		iterations = LegacyPBKDF2Iterations
	}
	key := pbkdf2.Key([]byte(password), salt, iterations, 32, sha256.New)

	// Decrypt seed
	encryptedSeed, err := hex.DecodeString(keystore.EncryptedSeed)
	if err != nil {
		return nil, fmt.Errorf("invalid encrypted seed: %w", err)
	}

	nonce, err := hex.DecodeString(keystore.Nonce)
	if err != nil {
		return nil, fmt.Errorf("invalid nonce: %w", err)
	}

	seedBytes, err := wm.decrypt(encryptedSeed, key, nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt seed (invalid password?): %w", err)
	}

	return &Wallet{
		Name:    keystore.Name,
		Address: keystore.Address,
		Seed:    string(seedBytes),
	}, nil
}

// RemoveWallet deletes a wallet from disk
func (wm *WalletManager) RemoveWallet(name string) error {
	keystorePath := filepath.Join(wm.walletsDir, name+".json")
	if _, err := os.Stat(keystorePath); os.IsNotExist(err) {
		return fmt.Errorf("wallet '%s' not found", name)
	}

	return os.Remove(keystorePath)
}

// encrypt encrypts data using AES-256-GCM
func (wm *WalletManager) encrypt(plaintext, key []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// decrypt decrypts data using AES-256-GCM
func (wm *WalletManager) decrypt(ciphertext, key, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// saveKeystore saves a keystore to disk
func (wm *WalletManager) saveKeystore(keystore *Keystore, path string) error {
	data, err := json.MarshalIndent(keystore, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keystore: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// loadKeystore loads a keystore from disk
func (wm *WalletManager) loadKeystore(path string) (*Keystore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read keystore file: %w", err)
	}

	var keystore Keystore
	if err := json.Unmarshal(data, &keystore); err != nil {
		return nil, fmt.Errorf("failed to parse keystore: %w", err)
	}

	return &keystore, nil
}
