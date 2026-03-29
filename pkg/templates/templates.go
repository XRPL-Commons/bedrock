package templates

import "github.com/xrpl-commons/bedrock/pkg/primitives"

// Template represents a project template
type Template struct {
	Name        string
	Primitive   string
	Description string
	LibRS       string
}

// Available returns all available project templates
func Available() map[string]Template {
	return map[string]Template{
		// Contract templates
		"basic": {
			Name:        "basic",
			Primitive:   primitives.Contract,
			Description: "Basic contract with hello function",
			LibRS:       basicLib,
		},
		"token": {
			Name:        "token",
			Primitive:   primitives.Contract,
			Description: "Fungible token contract with mint/transfer/balance",
			LibRS:       tokenLib,
		},
		"nft": {
			Name:        "nft",
			Primitive:   primitives.Contract,
			Description: "NFT contract with mint/transfer/owner tracking",
			LibRS:       nftLib,
		},
		"contract-escrow": {
			Name:        "contract-escrow",
			Primitive:   primitives.Contract,
			Description: "Escrow contract with create/release/cancel",
			LibRS:       escrowLib,
		},
		"counter": {
			Name:        "counter",
			Primitive:   primitives.Contract,
			Description: "Simple counter contract for learning",
			LibRS:       counterLib,
		},
		// Escrow templates
		"escrow-hello": {
			Name:        "escrow-hello",
			Primitive:   primitives.Escrow,
			Description: "Minimal escrow that releases immediately",
			LibRS:       escrowHelloLib,
		},
		"escrow-oracle": {
			Name:        "escrow-oracle",
			Primitive:   primitives.Escrow,
			Description: "Escrow that checks oracle price before releasing",
			LibRS:       escrowOracleLib,
		},
		// Vault templates
		"vault-hello": {
			Name:        "vault-hello",
			Primitive:   primitives.Vault,
			Description: "Minimal vault that allows all deposits/withdrawals",
			LibRS:       vaultHelloLib,
		},
		"vault-whitelist": {
			Name:        "vault-whitelist",
			Primitive:   primitives.Vault,
			Description: "Vault that restricts withdrawals to whitelisted addresses",
			LibRS:       vaultWhitelistLib,
		},
	}
}

// AvailableForPrimitive returns templates filtered by primitive type
func AvailableForPrimitive(p string) map[string]Template {
	all := Available()
	filtered := make(map[string]Template)
	for k, t := range all {
		if t.Primitive == p {
			filtered[k] = t
		}
	}
	return filtered
}

// DefaultTemplate returns the default template name for a given primitive
func DefaultTemplate(p string) string {
	switch p {
	case primitives.Escrow:
		return "escrow-hello"
	case primitives.Vault:
		return "vault-hello"
	default:
		return "basic"
	}
}

// CargoToml returns the Cargo.toml content for a given primitive and project name
func CargoToml(p string, projectName string) string {
	switch p {
	case primitives.Escrow:
		return escrowCargoToml(projectName)
	case primitives.Vault:
		return vaultCargoToml(projectName)
	default:
		return contractCargoToml(projectName)
	}
}

func contractCargoToml(name string) string {
	return `[package]
name = "` + name + `"
version = "0.1.0"
edition = "2021"

[lib]
crate-type = ["cdylib"]

[dependencies]
xrpl-wasm-std = { git = "https://github.com/Transia-RnD/craft.git", branch = "dangell/smart-contracts", package = "xrpl-wasm-std" }
xrpl-wasm-macros= { git = "https://github.com/Transia-RnD/craft.git", branch = "dangell/smart-contracts", package = "xrpl-wasm-macros" }

[profile.release]
opt-level = "z"
lto = true
strip = true
panic = "abort"
`
}

func escrowCargoToml(name string) string {
	return `[package]
name = "` + name + `"
version = "0.1.0"
edition = "2024"

[lib]
crate-type = ["cdylib"]

[dependencies]
xrpl-wasm-stdlib = { git = "https://github.com/ripple/xrpl-wasm-stdlib.git", package = "xrpl-wasm-stdlib" }

[profile.release]
opt-level = "s"
lto = true
codegen-units = 1
panic = "abort"
`
}

func vaultCargoToml(name string) string {
	return `[package]
name = "` + name + `"
version = "0.1.0"
edition = "2024"

[lib]
crate-type = ["cdylib"]

[dependencies]
xrpl-wasm-stdlib = { git = "https://github.com/willemolding/xrpl-wasm-stdlib.git", branch = "willem/smart-vaults", package = "xrpl-wasm-stdlib" }

[profile.release]
opt-level = "s"
lto = true
codegen-units = 1
panic = "abort"
`
}

const basicLib = `#![cfg_attr(target_arch = "wasm32", no_std)]

#[cfg(not(target_arch = "wasm32"))]
extern crate std;

use xrpl_wasm_macros::wasm_export;
use xrpl_wasm_std::host::trace::trace;

/// @xrpl-function hello
#[wasm_export]
fn hello() -> i32 {
    let _ = trace("Hello from XRPL Smart Contract!");
    0
}
`

const tokenLib = `#![cfg_attr(target_arch = "wasm32", no_std)]

#[cfg(not(target_arch = "wasm32"))]
extern crate std;

use xrpl_wasm_macros::wasm_export;
use xrpl_wasm_std::host::trace::trace;

/// @xrpl-function mint
/// @param amount UINT64 - Amount to mint
/// @return UINT64 - New total supply
#[wasm_export]
fn mint(amount: u64) -> u64 {
    let _ = trace("Minting tokens");
    amount
}

/// @xrpl-function transfer
/// @param to ACCOUNT - Recipient address
/// @param amount UINT64 - Amount to transfer
/// @return UINT64 - Success code (0 = success)
#[wasm_export]
fn transfer(to: &[u8], amount: u64) -> u64 {
    let _ = trace("Transferring tokens");
    let _ = (to, amount);
    0
}

/// @xrpl-function balance
/// @param account ACCOUNT - Account to check
/// @return UINT64 - Token balance
#[wasm_export]
fn balance(account: &[u8]) -> u64 {
    let _ = trace("Checking balance");
    let _ = account;
    0
}
`

const nftLib = `#![cfg_attr(target_arch = "wasm32", no_std)]

#[cfg(not(target_arch = "wasm32"))]
extern crate std;

use xrpl_wasm_macros::wasm_export;
use xrpl_wasm_std::host::trace::trace;

/// @xrpl-function mint_nft
/// @param to ACCOUNT - Owner address
/// @return UINT64 - Token ID
#[wasm_export]
fn mint_nft(to: &[u8]) -> u64 {
    let _ = trace("Minting NFT");
    let _ = to;
    1
}

/// @xrpl-function transfer_nft
/// @param token_id UINT64 - Token ID to transfer
/// @param to ACCOUNT - New owner address
/// @return UINT64 - Success code
#[wasm_export]
fn transfer_nft(token_id: u64, to: &[u8]) -> u64 {
    let _ = trace("Transferring NFT");
    let _ = (token_id, to);
    0
}

/// @xrpl-function owner_of
/// @param token_id UINT64 - Token ID to check
/// @return UINT64 - Owner info
#[wasm_export]
fn owner_of(token_id: u64) -> u64 {
    let _ = trace("Checking NFT owner");
    let _ = token_id;
    0
}
`

const escrowLib = `#![cfg_attr(target_arch = "wasm32", no_std)]

#[cfg(not(target_arch = "wasm32"))]
extern crate std;

use xrpl_wasm_macros::wasm_export;
use xrpl_wasm_std::host::trace::trace;

/// @xrpl-function create_escrow
/// @param beneficiary ACCOUNT - Escrow beneficiary
/// @param amount UINT64 - Escrow amount
/// @return UINT64 - Escrow ID
#[wasm_export]
fn create_escrow(beneficiary: &[u8], amount: u64) -> u64 {
    let _ = trace("Creating escrow");
    let _ = (beneficiary, amount);
    1
}

/// @xrpl-function release_escrow
/// @param escrow_id UINT64 - Escrow to release
/// @return UINT64 - Success code
#[wasm_export]
fn release_escrow(escrow_id: u64) -> u64 {
    let _ = trace("Releasing escrow");
    let _ = escrow_id;
    0
}

/// @xrpl-function cancel_escrow
/// @param escrow_id UINT64 - Escrow to cancel
/// @return UINT64 - Success code
#[wasm_export]
fn cancel_escrow(escrow_id: u64) -> u64 {
    let _ = trace("Cancelling escrow");
    let _ = escrow_id;
    0
}
`

// Escrow templates

const escrowHelloLib = `#![cfg_attr(target_arch = "wasm32", no_std)]

#[cfg(not(target_arch = "wasm32"))]
extern crate std;

use xrpl_wasm_stdlib::host::trace::trace;

#[unsafe(no_mangle)]
pub extern "C" fn finish() -> i32 {
    let _ = trace("Hello World!");
    1 // Release the escrow
}
`

const escrowOracleLib = `#![cfg_attr(target_arch = "wasm32", no_std)]

#[cfg(not(target_arch = "wasm32"))]
extern crate std;

use xrpl_wasm_stdlib::core::keylets::oracle_keylet;
use xrpl_wasm_stdlib::core::locator::Locator;
use xrpl_wasm_stdlib::core::types::account_id::AccountID;
use xrpl_wasm_stdlib::host::error_codes::match_result_code;
use xrpl_wasm_stdlib::host::trace::{trace_num, trace_data, DataRepr};
use xrpl_wasm_stdlib::host::{self, Result, Result::Ok, Result::Err};
use xrpl_wasm_stdlib::{r_address, sfield};

// TODO: Replace with your oracle owner address
const ORACLE_OWNER: AccountID = AccountID(r_address!("rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh"));
const ORACLE_DOCUMENT_ID: u32 = 1;

pub fn get_price_from_oracle(slot: i32) -> Result<u64> {
    let mut locator = Locator::new();
    locator.pack(sfield::PriceDataSeries);
    locator.pack(0);
    locator.pack(sfield::AssetPrice);

    let mut data: [u8; 8] = [0; 8];
    let result_code = unsafe {
        host::get_ledger_obj_nested_field(
            slot,
            locator.as_ptr(),
            locator.num_packed_bytes(),
            data.as_mut_ptr(),
            data.len(),
        )
    };
    let _ = trace_data("get_price_from_oracle: data=", &data, DataRepr::AsHex);

    match match_result_code(result_code, || data) {
        Ok(asset_bytes) => {
            let price = u64::from_le_bytes(asset_bytes);
            let _ = trace_num("get_price_from_oracle: asset_price=", price as i64);
            Ok(price)
        }
        Err(error) => {
            let _ = trace_num("Error getting asset_price", error.code() as i64);
            Err(error)
        }
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn finish() -> i32 {
    let oracle_keylet = match oracle_keylet(&ORACLE_OWNER, ORACLE_DOCUMENT_ID) {
        Ok(keylet) => keylet,
        Err(error) => {
            let _ = trace_num("finish: oracle_keylet error_code=", error.code() as i64);
            return error.code();
        }
    };

    let slot: i32;
    unsafe {
        slot = host::cache_ledger_obj(oracle_keylet.as_ptr(), oracle_keylet.len(), 0);
        let _ = trace_num("finish: cache_ledger_obj slot=", slot as i64);

        if slot < 0 {
            let _ = trace_num("finish: cache_ledger_obj failed, returning 0", 0);
            return 0;
        };
    }

    let price = match get_price_from_oracle(slot) {
        Ok(v) => v,
        Err(e) => return e.code(),
    };

    (price > 1) as i32 // Release escrow if price > 1
}
`

// Vault templates

const vaultHelloLib = `#![cfg_attr(target_arch = "wasm32", no_std)]

#[cfg(not(target_arch = "wasm32"))]
extern crate std;

use xrpl_wasm_stdlib::host::trace::trace;

#[unsafe(no_mangle)]
pub extern "C" fn on_deposit() -> i32 {
    let _ = trace("Deposit received!");
    1 // Allow deposit
}

#[unsafe(no_mangle)]
pub extern "C" fn on_withdraw() -> i32 {
    let _ = trace("Withdrawal requested!");
    1 // Allow withdrawal
}
`

const vaultWhitelistLib = `#![cfg_attr(target_arch = "wasm32", no_std)]

#[cfg(not(target_arch = "wasm32"))]
extern crate std;

use xrpl_wasm_stdlib::core::current_tx::get_field;
use xrpl_wasm_stdlib::core::types::account_id::AccountID;
use xrpl_wasm_stdlib::host::trace::{trace, trace_num};
use xrpl_wasm_stdlib::{r_address, sfield};

// TODO: Replace with your whitelisted addresses
const ALLOWED: [AccountID; 2] = [
    AccountID(r_address!("rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh")),
    AccountID(r_address!("rPT1Sjq2YGrBMTttX4GZHjKu9dyfzbpAYe")),
];

#[unsafe(no_mangle)]
pub extern "C" fn on_deposit() -> i32 {
    let _ = trace("Deposit received - allowed");
    1 // Allow all deposits
}

#[unsafe(no_mangle)]
pub extern "C" fn on_withdraw() -> i32 {
    let account: AccountID = match get_field(sfield::Account.into()) {
        xrpl_wasm_stdlib::host::Result::Ok(a) => a,
        xrpl_wasm_stdlib::host::Result::Err(e) => {
            let _ = trace_num("Failed to get account", e.code() as i64);
            return 0;
        }
    };

    for allowed in &ALLOWED {
        if account.0 == allowed.0 {
            let _ = trace("Withdrawal allowed - whitelisted address");
            return 1;
        }
    }

    let _ = trace("Withdrawal denied - address not whitelisted");
    0
}
`

const counterLib = `#![cfg_attr(target_arch = "wasm32", no_std)]

#[cfg(not(target_arch = "wasm32"))]
extern crate std;

use xrpl_wasm_macros::wasm_export;
use xrpl_wasm_std::host::trace::trace;

/// @xrpl-function increment
/// @return UINT64 - New counter value
#[wasm_export]
fn increment() -> u64 {
    let _ = trace("Incrementing counter");
    1
}

/// @xrpl-function decrement
/// @return UINT64 - New counter value
#[wasm_export]
fn decrement() -> u64 {
    let _ = trace("Decrementing counter");
    0
}

/// @xrpl-function get_count
/// @return UINT64 - Current counter value
#[wasm_export]
fn get_count() -> u64 {
    let _ = trace("Getting counter value");
    0
}
`
