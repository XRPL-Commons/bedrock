package wallet

import "testing"

// The XRPL "masterpassphrase" root account: a well-known deterministic vector.
// This secp256k1 seed always derives to this classic address (it is also the
// genesis account of a standalone rippled node).
const (
	masterSeed    = "snoPBrXtMeMyMHUVTgbuqAfg1SUTb"
	masterAddress = "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh"
)

func newWallet(t *testing.T) *XRPLWallet {
	t.Helper()
	w, err := NewXRPLWallet()
	if err != nil {
		t.Fatalf("NewXRPLWallet() error: %v", err)
	}
	return w
}

func TestValidateSeed(t *testing.T) {
	w := newWallet(t)
	tests := []struct {
		name    string
		seed    string
		wantErr bool
	}{
		{"valid master seed", masterSeed, false},
		{"empty", "", true},
		{"missing s prefix", "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", true},
		{"garbage", "snotARealSeed000000000000000", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := w.ValidateSeed(tt.seed)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSeed(%q) err = %v, wantErr = %v", tt.seed, err, tt.wantErr)
			}
		})
	}
}

// Known-answer test: the master seed must derive to the master address.
func TestSeedToAddress_KnownVector(t *testing.T) {
	w := newWallet(t)
	addr, err := w.SeedToAddress(masterSeed)
	if err != nil {
		t.Fatalf("SeedToAddress() error: %v", err)
	}
	if addr != masterAddress {
		t.Errorf("SeedToAddress(%q) = %q, want %q", masterSeed, addr, masterAddress)
	}
}

func TestSeedToAddress_Deterministic(t *testing.T) {
	w := newWallet(t)
	a1, err := w.SeedToAddress(masterSeed)
	if err != nil {
		t.Fatalf("SeedToAddress() error: %v", err)
	}
	a2, _ := w.SeedToAddress(masterSeed)
	if a1 != a2 {
		t.Errorf("SeedToAddress not deterministic: %q != %q", a1, a2)
	}
}

func TestValidateAddress(t *testing.T) {
	w := newWallet(t)
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{"valid master address", masterAddress, false},
		{"empty", "", true},
		{"missing r prefix", "sHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", true},
		{"too short", "rShort", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := w.ValidateAddress(tt.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAddress(%q) err = %v, wantErr = %v", tt.address, err, tt.wantErr)
			}
		})
	}
}

// A freshly generated wallet's seed must derive back to its own address.
func TestGenerateWallet_RoundTrip(t *testing.T) {
	w := newWallet(t)
	gen, err := w.GenerateWallet("test")
	if err != nil {
		t.Fatalf("GenerateWallet() error: %v", err)
	}
	if err := w.ValidateSeed(gen.Seed); err != nil {
		t.Errorf("generated seed failed validation: %v", err)
	}
	derived, err := w.SeedToAddress(gen.Seed)
	if err != nil {
		t.Fatalf("SeedToAddress() error: %v", err)
	}
	if derived != gen.Address {
		t.Errorf("round trip mismatch: generated %q but seed derives to %q", gen.Address, derived)
	}
}

func TestGetAlgorithm(t *testing.T) {
	for _, algo := range []string{"secp256k1", "ed25519", "", "SECP256K1"} {
		if _, err := getAlgorithm(algo); err != nil {
			t.Errorf("getAlgorithm(%q) unexpected error: %v", algo, err)
		}
	}
	if _, err := getAlgorithm("rsa"); err == nil {
		t.Error("getAlgorithm(rsa) should error")
	}
}
