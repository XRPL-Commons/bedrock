package wallet

import (
	"fmt"
	"syscall"

	"golang.org/x/term"
)

// AuthProvider handles authentication for wallet operations
type AuthProvider struct{}

// NewAuthProvider creates a new authentication provider
func NewAuthProvider() *AuthProvider {
	return &AuthProvider{}
}

// GetPassword prompts for a password with hidden input
func (a *AuthProvider) GetPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println() // Add newline after password input
	return string(bytePassword), nil
}
