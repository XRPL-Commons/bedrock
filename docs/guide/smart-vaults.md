# Smart Vaults

Smart vaults use WASM code to define custom deposit and withdraw logic on XRPL. The WASM `on_deposit()` and `on_withdraw()` functions return `1` to allow or `0` to deny.

## Overview

A smart vault is an on-chain asset container with programmable access control. You write Rust code that decides who can deposit and withdraw, under what conditions. Use cases include whitelisted custody, time-locked savings, and rule-based treasury management.

**XRPL transactions involved:**
- `VaultCreate` — Create a vault with WASM logic
- `VaultDeposit` — Deposit assets (executes `on_deposit()`)
- `VaultWithdraw` — Withdraw assets (executes `on_withdraw()`)

## Getting Started

### 1. Create a Vault Project

```bash
bedrock init my-vault --primitives vault
cd my-vault
```

This creates:

```
my-vault/
├── bedrock.toml
├── .wallets/
├── .bedrock/
│   └── node-config/
│       ├── xrpld.cfg
│       └── validators.txt
├── vault/
│   ├── Cargo.toml
│   └── src/
│       └── lib.rs
└── README.md
```

### 2. Explore the Default Template

The default `vault-hello` template creates a vault that allows all deposits and withdrawals:

```rust
#![cfg_attr(target_arch = "wasm32", no_std)]

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
```

- `on_deposit()` — Called when someone deposits into the vault. Return `1` to allow, `0` to deny.
- `on_withdraw()` — Called when someone withdraws from the vault. Return `1` to allow, `0` to deny.

### 3. Build

```bash
bedrock build
```

Vaults compile to the `wasm32v1-none` target using Rust edition 2024.

### 4. Start a Local Node

```bash
bedrock node start
```

Bedrock auto-configures the node with SmartVault features enabled.

### 5. Deploy

```bash
bedrock vault deploy \
  --asset XRP \
  --wallet <seed> \
  --network local
```

Save the **vault ID** from the output.

### 6. Deposit

```bash
bedrock vault deposit <vault-id> \
  --amount 1000000 \
  --wallet <seed> \
  --network local
```

### 7. Check Status

```bash
bedrock vault status <vault-id> --network local
```

### 8. Withdraw

```bash
bedrock vault withdraw <vault-id> \
  --amount 500000 \
  --destination <address> \
  --wallet <seed> \
  --network local
```

## Commands Reference

### `bedrock vault deploy`

Create a smart vault with WASM logic.

```bash
bedrock vault deploy [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--asset` | Asset currency code | `XRP` |
| `--issuer` | Asset issuer (required for non-XRP) | - |
| `--max-capacity` | Maximum vault capacity | - |
| `--wallet` | Wallet seed or jade name | - |
| `--network` | Network (local, alphanet) | `local` |
| `--fee` | Transaction fee in drops | - |
| `--skip-build` | Skip building WASM | `false` |

```bash
# XRP vault
bedrock vault deploy --asset XRP --wallet sXXX... --network local

# IOU vault
bedrock vault deploy --asset USD --issuer rXXX... --wallet sXXX... --network alphanet

# With capacity limit
bedrock vault deploy --asset XRP --max-capacity 1000000000 --wallet sXXX...
```

### `bedrock vault deposit`

Deposit assets into a vault.

```bash
bedrock vault deposit <vault-id> [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--amount` | Amount in drops (required) | - |
| `--wallet` | Wallet seed or jade name (required) | - |
| `--network` | Network | `local` |
| `--fee` | Transaction fee in drops | - |

### `bedrock vault withdraw`

Withdraw assets from a vault.

```bash
bedrock vault withdraw <vault-id> [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--amount` | Amount in drops (required) | - |
| `--destination` | Withdrawal destination (required) | - |
| `--wallet` | Wallet seed or jade name (required) | - |
| `--network` | Network | `local` |
| `--fee` | Transaction fee in drops | - |

### `bedrock vault status`

Query the current status of a vault.

```bash
bedrock vault status <vault-id> [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--network` | Network | `local` |

## Templates

### vault-hello (default)

Minimal vault that allows all deposits and withdrawals. Good starting point.

### vault-whitelist

Allows all deposits but restricts withdrawals to a list of whitelisted addresses. Demonstrates reading the current transaction's account field from WASM:

```rust
#![cfg_attr(target_arch = "wasm32", no_std)]

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
```

Use the whitelist template with:

```bash
bedrock init my-vault --primitives vault --template vault-whitelist
```

## Configuration

Vault projects use this `bedrock.toml` structure:

```toml
[project]
name = "my-vault"
version = "0.1.0"
primitives = ["vault"]

[vaults.main]
source = "vault/src/lib.rs"
output = "vault/target/wasm32v1-none/release"

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
name = "my-vault"
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
```

## Key Differences from Contracts

| | Smart Contract | Smart Vault |
|---|---|---|
| WASM target | `wasm32-unknown-unknown` | `wasm32v1-none` |
| Rust edition | 2021 | 2024 |
| Crate | `xrpl_wasm_std` + `xrpl_wasm_macros` | `xrpl_wasm_stdlib` |
| ABI | Required | Not needed |
| Entry points | `@xrpl-function` annotated functions | `on_deposit()`, `on_withdraw()` |
| Docker image | `transia/cluster` | `willemolding/rippled:smart-vaults.0` |
| Node config | genesis.json + xrpld.cfg | xrpld.cfg (standalone) |

## Development Workflow

```bash
# 1. Create project
bedrock init my-vault --primitives vault && cd my-vault

# 2. Start local node
bedrock node start

# 3. Edit vault logic
#    vault/src/lib.rs

# 4. Build
bedrock build

# 5. Fund a wallet
bedrock faucet --network local

# 6. Deploy
bedrock vault deploy --asset XRP --wallet <seed> --network local

# 7. Deposit
bedrock vault deposit <vault-id> --amount 1000000 --wallet <seed> --network local

# 8. Check status
bedrock vault status <vault-id> --network local

# 9. Withdraw
bedrock vault withdraw <vault-id> \
  --amount 500000 \
  --destination <address> \
  --wallet <seed> \
  --network local
```

## Troubleshooting

### Build fails with wasm32v1-none target error

```bash
rustup target add wasm32v1-none
```

### Deposit or withdrawal denied

Your `on_deposit()` or `on_withdraw()` function returned `0`. Check your WASM logic — the vault is working as designed.

### Node won't start

Ensure Docker is running: `docker ps`
