# Smart Escrows

Smart escrows use WASM code to define custom release conditions on XRPL. The WASM `finish()` function returns `1` to release the escrow or `0` to keep it locked.

## Overview

A smart escrow locks funds on-chain and releases them only when a WASM condition is met. This enables programmable conditional payments — time-locks, oracle-based triggers, multi-party approval, and more.

**XRPL transactions involved:**
- `EscrowCreate` — Lock funds with a WASM condition
- `EscrowFinish` — Attempt to release (executes the WASM `finish()` function)
- `EscrowCancel` — Cancel the escrow (if cancel-after time has passed)

## Getting Started

### 1. Create an Escrow Project

```bash
bedrock init my-escrow --primitives escrow
cd my-escrow
```

This creates:

```
my-escrow/
├── bedrock.toml
├── .wallets/
├── .bedrock/
│   └── node-config/
│       ├── xrpld.cfg
│       └── validators.txt
├── escrow/
│   ├── Cargo.toml
│   └── src/
│       └── lib.rs
└── README.md
```

### 2. Explore the Default Template

The default `escrow-hello` template creates a minimal escrow that always releases:

```rust
#![cfg_attr(target_arch = "wasm32", no_std)]

#[cfg(not(target_arch = "wasm32"))]
extern crate std;

use xrpl_wasm_stdlib::host::trace::trace;

#[unsafe(no_mangle)]
pub extern "C" fn finish() -> i32 {
    let _ = trace("Hello World!");
    1 // Release the escrow
}
```

The `finish()` function is the entry point called by the XRPL ledger when someone submits an `EscrowFinish` transaction. Return `1` to release funds to the destination, or `0` to keep them locked.

### 3. Build

```bash
bedrock build
```

Escrows compile to the `wasm32v1-none` target using Rust edition 2024.

### 4. Start a Local Node

```bash
bedrock node start
```

Bedrock auto-configures the node with SmartEscrow features enabled.

### 5. Deploy

```bash
bedrock escrow deploy \
  --destination <beneficiary-address> \
  --amount 1000000 \
  --wallet <seed> \
  --network local
```

Save the **owner address** and **escrow sequence** from the output — you'll need them to finish or cancel the escrow.

### 6. Check Status

```bash
bedrock escrow status <owner> <sequence> --network local
```

### 7. Finish (Release)

```bash
bedrock escrow finish <owner> <sequence> \
  --wallet <seed> \
  --network local
```

The on-chain WASM `finish()` function executes and determines whether the escrow is released.

## Commands Reference

### `bedrock escrow deploy`

Create a smart escrow with a WASM condition.

```bash
bedrock escrow deploy [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--destination` | Escrow beneficiary address (required) | - |
| `--amount` | Amount in drops (required) | - |
| `--cancel-after` | Cancel after time (ripple epoch) | - |
| `--finish-after` | Finish after time (ripple epoch) | - |
| `--wallet` | Wallet seed or jade name | - |
| `--network` | Network (local, alphanet) | `local` |
| `--fee` | Transaction fee in drops | - |
| `--skip-build` | Skip building WASM | `false` |

```bash
# Basic deployment
bedrock escrow deploy --destination rXXX... --amount 1000000 --wallet sXXX... --network local

# With time constraints
bedrock escrow deploy --destination rXXX... --amount 1000000 \
  --cancel-after 1234567 --finish-after 1234000 \
  --wallet sXXX... --network alphanet
```

### `bedrock escrow finish`

Submit an EscrowFinish transaction to release the escrow.

```bash
bedrock escrow finish <owner> <sequence> [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--wallet` | Wallet seed or jade name (required) | - |
| `--network` | Network | `local` |
| `--fee` | Transaction fee in drops | - |

### `bedrock escrow cancel`

Cancel an escrow (only possible after `cancel-after` time).

```bash
bedrock escrow cancel <owner> <sequence> [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--wallet` | Wallet seed or jade name (required) | - |
| `--network` | Network | `local` |
| `--fee` | Transaction fee in drops | - |

### `bedrock escrow status`

Query the current status of an escrow.

```bash
bedrock escrow status <owner> <sequence> [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--network` | Network | `local` |

## Templates

### escrow-hello (default)

Minimal escrow that always releases. Good starting point.

### escrow-oracle

Queries an on-chain oracle and releases the escrow only if the price exceeds a threshold. Demonstrates how to read ledger objects from WASM:

```rust
#![cfg_attr(target_arch = "wasm32", no_std)]

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
```

Use the oracle template with:

```bash
bedrock init my-escrow --primitives escrow --template escrow-oracle
```

## Configuration

Escrow projects use this `bedrock.toml` structure:

```toml
[project]
name = "my-escrow"
version = "0.1.0"
primitives = ["escrow"]

[escrows.main]
source = "escrow/src/lib.rs"
output = "escrow/target/wasm32v1-none/release"

[local_node]
config_dir = ".bedrock/node-config"
docker_image = "willemolding/rippled:smart-vaults.0"
ledger_interval = 1000

[networks.local]
url = "ws://localhost:6006"
network_id = 100
faucet_url = "http://localhost:8080/faucet"

[networks.alphanet]
url = "wss://alphanet.nerdnest.xyz"
network_id = 21465
faucet_url = "https://alphanet.faucet.nerdnest.xyz/accounts"

[wallets]
keystore = ".wallets/keystore.json"
```

### Cargo.toml

```toml
[package]
name = "my-escrow"
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
```

## Key Differences from Contracts

| | Smart Contract | Smart Escrow |
|---|---|---|
| WASM target | `wasm32-unknown-unknown` | `wasm32v1-none` |
| Rust edition | 2021 | 2024 |
| Crate | `xrpl_wasm_std` + `xrpl_wasm_macros` | `xrpl_wasm_stdlib` |
| ABI | Required | Not needed |
| Entry point | `@xrpl-function` annotated functions | `finish()` |
| Docker image | `transia/cluster` | `willemolding/rippled:smart-vaults.0` |
| Node config | genesis.json + xrpld.cfg | xrpld.cfg (standalone) |

## Development Workflow

```bash
# 1. Create project
bedrock init my-escrow --primitives escrow && cd my-escrow

# 2. Start local node
bedrock node start

# 3. Edit escrow logic
#    escrow/src/lib.rs

# 4. Build
bedrock build

# 5. Fund a wallet
bedrock faucet --network local

# 6. Deploy
bedrock escrow deploy \
  --destination <beneficiary> \
  --amount 1000000 \
  --wallet <seed> \
  --network local

# 7. Check status
bedrock escrow status <owner> <sequence> --network local

# 8. Finish (release)
bedrock escrow finish <owner> <sequence> --wallet <seed> --network local

# 9. Or cancel
bedrock escrow cancel <owner> <sequence> --wallet <seed> --network local
```

## Troubleshooting

### Build fails with wasm32v1-none target error

```bash
rustup target add wasm32v1-none
```

### Escrow finish returns 0

Your `finish()` function returned `0`, meaning the condition was not met. Check your WASM logic.

### Node won't start

Ensure Docker is running: `docker ps`
