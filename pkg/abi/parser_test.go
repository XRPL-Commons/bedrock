package abi

import (
	"os"
	"path/filepath"
	"testing"
)

// writeRust creates a temp dir containing a single lib.rs with the given body
// and returns the dir (suitable for NewParser).
func writeRust(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "lib.rs"), []byte(body), 0644); err != nil {
		t.Fatalf("failed to write lib.rs: %v", err)
	}
	return dir
}

func TestParseContract_FunctionWithParamsFlagReturn(t *testing.T) {
	// @flag is positional: it applies to the params that follow it. Here
	// `amount` precedes the @flag (so flag 0) and `dest` follows it (flag 1).
	dir := writeRust(t, `
/// @xrpl-function transfer
/// @param amount UINT64 - amount in drops
/// @flag 1
/// @param dest ACCOUNT - recipient
/// @return UINT32 - status code
pub fn transfer(amount: u64) -> i32 {
    0
}
`)

	parsed, err := NewParser(dir).ParseContract("demo")
	if err != nil {
		t.Fatalf("ParseContract() error: %v", err)
	}

	if parsed.ContractName != "demo" {
		t.Errorf("ContractName = %q, want demo", parsed.ContractName)
	}
	if len(parsed.Functions) != 1 {
		t.Fatalf("got %d functions, want 1: %+v", len(parsed.Functions), parsed.Functions)
	}

	fn := parsed.Functions[0]
	if fn.Name != "transfer" {
		t.Errorf("function name = %q, want transfer", fn.Name)
	}
	if len(fn.Parameters) != 2 {
		t.Fatalf("got %d params, want 2: %+v", len(fn.Parameters), fn.Parameters)
	}
	if fn.Parameters[0].Name != "amount" || fn.Parameters[0].Type != "UINT64" {
		t.Errorf("param[0] = %+v, want amount/UINT64", fn.Parameters[0])
	}
	if fn.Parameters[0].Description != "amount in drops" {
		t.Errorf("param[0].Description = %q", fn.Parameters[0].Description)
	}
	if fn.Parameters[1].Name != "dest" || fn.Parameters[1].Type != "ACCOUNT" {
		t.Errorf("param[1] = %+v, want dest/ACCOUNT", fn.Parameters[1])
	}
	// Positional @flag: amount (before @flag) = 0, dest (after @flag) = 1.
	if fn.Parameters[0].Flag != 0 {
		t.Errorf("param[0].Flag = %d, want 0", fn.Parameters[0].Flag)
	}
	if fn.Parameters[1].Flag != 1 {
		t.Errorf("param[1].Flag = %d, want 1", fn.Parameters[1].Flag)
	}
	if fn.Returns == nil || fn.Returns.Type != "UINT32" {
		t.Errorf("return = %+v, want UINT32", fn.Returns)
	}
}

func TestParseContract_MultipleFunctions(t *testing.T) {
	dir := writeRust(t, `
/// @xrpl-function hello
pub fn hello() -> i32 { 0 }

/// @xrpl-function ping
pub fn ping() -> i32 { 0 }
`)

	parsed, err := NewParser(dir).ParseContract("demo")
	if err != nil {
		t.Fatalf("ParseContract() error: %v", err)
	}
	if len(parsed.Functions) != 2 {
		t.Fatalf("got %d functions, want 2", len(parsed.Functions))
	}
}

func TestParseContract_InvalidParamTypeErrors(t *testing.T) {
	dir := writeRust(t, `
/// @xrpl-function bad
/// @param x NOTATYPE
pub fn bad() -> i32 { 0 }
`)

	if _, err := NewParser(dir).ParseContract("demo"); err == nil {
		t.Fatal("expected error for invalid parameter type, got nil")
	}
}

func TestParseContract_NameMismatchErrors(t *testing.T) {
	dir := writeRust(t, `
/// @xrpl-function foo
pub fn bar() -> i32 { 0 }
`)

	if _, err := NewParser(dir).ParseContract("demo"); err == nil {
		t.Fatal("expected error when @xrpl-function name != fn name, got nil")
	}
}

func TestIsValidType(t *testing.T) {
	valid := []string{"UINT8", "UINT256", "VL", "ACCOUNT", "AMOUNT", "ISSUE", "CURRENCY", "NUMBER"}
	for _, ty := range valid {
		if !IsValidType(ty) {
			t.Errorf("IsValidType(%q) = false, want true", ty)
		}
	}
	for _, ty := range []string{"", "uint8", "STRING", "u64"} {
		if IsValidType(ty) {
			t.Errorf("IsValidType(%q) = true, want false", ty)
		}
	}
}

func TestGetTypeInfo(t *testing.T) {
	info, ok := GetTypeInfo("UINT64")
	if !ok {
		t.Fatal("GetTypeInfo(UINT64) not found")
	}
	if info.RustType != "u64" {
		t.Errorf("UINT64.RustType = %q, want u64", info.RustType)
	}
	if _, ok := GetTypeInfo("NOPE"); ok {
		t.Error("GetTypeInfo(NOPE) = ok, want not found")
	}
}
