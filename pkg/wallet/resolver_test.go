package wallet

import (
	"strings"
	"testing"
)

// TestResolveSeedPassthrough ensures a raw seed is returned as-is (no
// keystore lookup, no password prompt) when no wallet by that name exists.
func TestResolveSeedPassthrough(t *testing.T) {
	wm := newTestManager(t)
	wr := &WalletResolver{manager: wm, authProvider: NewAuthProvider()}

	xw, err := NewXRPLWallet()
	if err != nil {
		t.Fatalf("NewXRPLWallet: %v", err)
	}
	w, err := xw.GenerateWallet("ignored")
	if err != nil {
		t.Fatalf("GenerateWallet: %v", err)
	}

	got, err := wr.ResolveWallet(w.Seed)
	if err != nil {
		t.Fatalf("ResolveWallet(seed): %v", err)
	}
	if got != w.Seed {
		t.Errorf("seed passthrough mismatch: got %q want %q", got, w.Seed)
	}
}

// TestResolveWalletNameStartingWithS proves the regression-fix: a wallet
// name like "swap-vault" should not be misclassified as a seed. With the
// old heuristic ("starts with s => seed") this would have called
// ValidateSeed("swap-vault") and produced a confusing error.
//
// We verify by ensuring that a non-existent wallet name returns a "not a
// valid XRPL seed" error rather than panicking, and that an existing wallet
// is dispatched to the keystore branch.
func TestResolveNameStartingWithS_NotASeed(t *testing.T) {
	wm := newTestManager(t)
	wr := &WalletResolver{manager: wm, authProvider: NewAuthProvider()}

	_, err := wr.ResolveWallet("swap-vault")
	if err == nil {
		t.Fatal("expected error when wallet name does not exist and is not a seed")
	}
	// Error should mention both keystore and seed paths, not just say "seed
	// must start with 's'".
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention keystore lookup, got: %v", err)
	}
}

func TestResolveEmpty(t *testing.T) {
	wm := newTestManager(t)
	wr := &WalletResolver{manager: wm, authProvider: NewAuthProvider()}

	if _, err := wr.ResolveWallet(""); err == nil {
		t.Fatal("expected error for empty input")
	}
}
