package wallet

import (
	"fmt"
)

// WalletResolver helps resolve wallet names to seeds
type WalletResolver struct {
	manager      *WalletManager
	authProvider *AuthProvider
}

// NewWalletResolver creates a new wallet resolver
func NewWalletResolver() (*WalletResolver, error) {
	manager, err := NewWalletManager()
	if err != nil {
		return nil, err
	}

	authProvider := NewAuthProvider()

	return &WalletResolver{
		manager:      manager,
		authProvider: authProvider,
	}, nil
}

// ResolveWallet resolves a wallet input to a seed.
//
// Resolution order:
//  1. If a keystore with that name exists, prompt for a password and decrypt
//     it. This is tried first so wallet names that happen to start with 's'
//     (e.g. "swap-vault") are not misinterpreted as raw seeds.
//  2. Otherwise, validate the input as a raw XRPL seed and return it as-is.
func (wr *WalletResolver) ResolveWallet(walletInput string) (string, error) {
	if walletInput == "" {
		return "", fmt.Errorf("wallet input cannot be empty")
	}

	if wr.manager.WalletExists(walletInput) {
		return wr.resolveWalletName(walletInput)
	}

	xrplWallet, err := NewXRPLWallet()
	if err != nil {
		return "", fmt.Errorf("failed to create XRPL wallet: %w", err)
	}
	if err := xrplWallet.ValidateSeed(walletInput); err != nil {
		return "", fmt.Errorf("wallet %q not found in keystore and is not a valid XRPL seed: %w", walletInput, err)
	}
	return walletInput, nil
}

// resolveWalletName resolves a wallet name to its seed.
// Callers should have already verified the wallet exists.
func (wr *WalletResolver) resolveWalletName(walletName string) (string, error) {
	// Get password and load wallet
	password, err := wr.authProvider.GetPassword("Enter password: ")
	if err != nil {
		return "", fmt.Errorf("failed to get password: %w", err)
	}

	wallet, err := wr.manager.LoadWallet(walletName, password)
	if err != nil {
		return "", fmt.Errorf("failed to load wallet '%s': %w", walletName, err)
	}

	return wallet.Seed, nil
}
