package wallet

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWalletManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bedrock-wallets-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Temporarily override user home dir for testing
	oldHomeDir := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHomeDir)

	wm, err := NewWalletManager()
	require.NoError(t, err)

	expectedWalletsDir := filepath.Join(tempDir, ".config", "bedrock", "wallets")
	assert.Equal(t, expectedWalletsDir, wm.walletsDir)
	_, err = os.Stat(expectedWalletsDir)
	assert.NoError(t, err, "wallets directory should be created")
}

func TestWalletManager_ValidateWalletName(t *testing.T) {
	wm := &WalletManager{}

	testCases := []struct {
		name      string
		expectErr bool
	}{
		{"valid-name", false},
		{"invalid name", true},
		{"", true},
		{"this-name-is-way-too-long", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := wm.ValidateWalletName(tc.name)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWalletManager_WalletLifecycle(t *testing.T) {
	// Setup temporary directory for wallets
	tempDir, err := os.MkdirTemp("", "test-wallets-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	wm := &WalletManager{
		walletsDir: tempDir,
	}

	// Test data
	wallet := &Wallet{
		Name:    "test-wallet",
		Address: "0x123",
		Seed:    "test-seed",
	}
	password := "password123"

	// 1. Save wallet
	err = wm.SaveWallet(wallet, password)
	require.NoError(t, err)

	// 2. Check if wallet exists
	assert.True(t, wm.WalletExists(wallet.Name))

	// 3. Load wallet
	loadedWallet, err := wm.LoadWallet(wallet.Name, password)
	require.NoError(t, err)
	assert.Equal(t, wallet.Name, loadedWallet.Name)
	assert.Equal(t, wallet.Address, loadedWallet.Address)
	assert.Equal(t, wallet.Seed, loadedWallet.Seed)

	// 4. List wallets
	wallets, err := wm.ListWallets()
	require.NoError(t, err)
	require.Len(t, wallets, 1)
	assert.Equal(t, wallet.Name, wallets[0].Name)
	assert.Equal(t, wallet.Address, wallets[0].Address)
	assert.WithinDuration(t, time.Now(), wallets[0].CreatedAt, 5*time.Second)

	// 5. Remove wallet
	err = wm.RemoveWallet(wallet.Name)
	require.NoError(t, err)
	assert.False(t, wm.WalletExists(wallet.Name))
}
