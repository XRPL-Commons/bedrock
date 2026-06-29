package primitives

import "testing"

func TestAll(t *testing.T) {
	got := All()
	want := []string{Contract, Escrow, Vault}

	if len(got) != len(want) {
		t.Fatalf("All() returned %d kinds, want %d", len(got), len(want))
	}
	for i, kind := range want {
		if got[i] != kind {
			t.Errorf("All()[%d] = %q, want %q", i, got[i], kind)
		}
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		kind string
		want bool
	}{
		{Contract, true},
		{Escrow, true},
		{Vault, true},
		{"", false},
		{"unknown", false},
		{"Contract", false}, // case-sensitive
	}

	for _, tt := range tests {
		if got := IsValid(tt.kind); got != tt.want {
			t.Errorf("IsValid(%q) = %v, want %v", tt.kind, got, tt.want)
		}
	}
}

// TestRegistryConsistency guards the invariants other code relies on:
// every advertised kind has a matching entry, and the build/ABI/target
// metadata matches each primitive's documented behaviour.
func TestRegistryConsistency(t *testing.T) {
	for _, kind := range All() {
		def, ok := Registry[kind]
		if !ok {
			t.Errorf("All() lists %q but Registry has no entry for it", kind)
			continue
		}
		if def.Kind != kind {
			t.Errorf("Registry[%q].Kind = %q, want %q", kind, def.Kind, kind)
		}
		if !def.NeedsBuild {
			t.Errorf("Registry[%q].NeedsBuild = false, want true (all primitives compile to WASM)", kind)
		}
		if def.WasmTarget == "" || def.RustEdition == "" || def.SourceDir == "" {
			t.Errorf("Registry[%q] has empty target/edition/sourceDir: %+v", kind, def)
		}
	}

	// Contract is the only primitive with an ABI; escrow/vault have none.
	if !Registry[Contract].NeedsABI {
		t.Error("contract should need an ABI")
	}
	if Registry[Escrow].NeedsABI || Registry[Vault].NeedsABI {
		t.Error("escrow and vault should not need an ABI")
	}

	// Targets differ: contract is wasm32-unknown-unknown, the others wasm32v1-none.
	if Registry[Contract].WasmTarget != "wasm32-unknown-unknown" {
		t.Errorf("contract target = %q, want wasm32-unknown-unknown", Registry[Contract].WasmTarget)
	}
	for _, kind := range []string{Escrow, Vault} {
		if Registry[kind].WasmTarget != "wasm32v1-none" {
			t.Errorf("%s target = %q, want wasm32v1-none", kind, Registry[kind].WasmTarget)
		}
	}
}
