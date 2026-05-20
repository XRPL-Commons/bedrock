package wallet

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// newTestManager returns a WalletManager rooted in a temp directory, so tests
// never touch the user's real ~/.config/bedrock/wallets.
func newTestManager(t *testing.T) *WalletManager {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatalf("mkdir tempdir: %v", err)
	}
	return &WalletManager{walletsDir: dir}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	wm := newTestManager(t)
	w := &Wallet{
		Name:    "alice",
		Address: "rExampleAddress",
		Seed:    "sEdSeedExampleValue",
	}
	if err := wm.SaveWallet(w, "correct horse battery staple"); err != nil {
		t.Fatalf("SaveWallet: %v", err)
	}

	got, err := wm.LoadWallet("alice", "correct horse battery staple")
	if err != nil {
		t.Fatalf("LoadWallet: %v", err)
	}
	if got.Seed != w.Seed {
		t.Errorf("seed round-trip mismatch: got %q want %q", got.Seed, w.Seed)
	}
	if got.Address != w.Address {
		t.Errorf("address mismatch: got %q want %q", got.Address, w.Address)
	}
}

func TestLoadWrongPasswordFails(t *testing.T) {
	wm := newTestManager(t)
	w := &Wallet{Name: "bob", Address: "rBob", Seed: "sBobSeed"}
	if err := wm.SaveWallet(w, "right-password"); err != nil {
		t.Fatalf("SaveWallet: %v", err)
	}
	if _, err := wm.LoadWallet("bob", "wrong-password"); err == nil {
		t.Fatal("expected error decrypting with wrong password, got nil")
	}
}

func TestSaveRefusesDuplicate(t *testing.T) {
	wm := newTestManager(t)
	w := &Wallet{Name: "carol", Address: "rCarol", Seed: "sCarolSeed"}
	if err := wm.SaveWallet(w, "pw"); err != nil {
		t.Fatalf("first save: %v", err)
	}
	if err := wm.SaveWallet(w, "pw"); err == nil {
		t.Fatal("expected duplicate save to error")
	}
}

// TestLegacyKeystoreUpgradePath verifies that a keystore written with the v1
// schema (no Iterations field, fixed 100k iterations) still decrypts after the
// PBKDF2 iteration count was bumped. Without the legacy fallback this would
// silently corrupt every wallet created before the upgrade.
func TestLegacyKeystoreUpgradePath(t *testing.T) {
	wm := newTestManager(t)

	const (
		name     = "legacy"
		address  = "rLegacy"
		seed     = "sLegacySeed"
		password = "hunter2"
	)

	// Hand-build a v1 keystore (no Iterations field) using the legacy
	// iteration count.
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		t.Fatal(err)
	}
	key := pbkdf2.Key([]byte(password), salt, LegacyPBKDF2Iterations, 32, sha256.New)
	encryptedSeed, nonce, err := wm.encrypt([]byte(seed), key)
	if err != nil {
		t.Fatal(err)
	}

	legacy := &Keystore{
		Version:       1,
		Name:          name,
		Address:       address,
		CreatedAt:     time.Now(),
		EncryptedSeed: hex.EncodeToString(encryptedSeed),
		Salt:          hex.EncodeToString(salt),
		Nonce:         hex.EncodeToString(nonce),
		// Iterations intentionally omitted.
	}
	data, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(wm.walletsDir, name+".json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	got, err := wm.LoadWallet(name, password)
	if err != nil {
		t.Fatalf("loading v1 keystore should still work, got %v", err)
	}
	if got.Seed != seed {
		t.Errorf("seed mismatch: got %q want %q", got.Seed, seed)
	}
}

func TestValidateWalletName(t *testing.T) {
	wm := newTestManager(t)
	cases := []struct {
		name    string
		wantErr bool
	}{
		{"alice", false},
		{"alice_42", false},
		{"a-b-c", false},
		{"", true},
		{"way-too-long-of-a-wallet-name", true},
		{"has spaces", true},
		{"has/slash", true},
		{"weird!chars", true},
	}
	for _, tc := range cases {
		err := wm.ValidateWalletName(tc.name)
		if (err != nil) != tc.wantErr {
			t.Errorf("ValidateWalletName(%q): err=%v wantErr=%v", tc.name, err, tc.wantErr)
		}
	}
}
