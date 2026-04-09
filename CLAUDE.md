# Bedrock CLI Instructions

This file provides instructions for AI assistants working with the Bedrock XRPL smart contract development tool.

## What is Bedrock?

Bedrock is a CLI tool for developing, deploying, and interacting with XRPL smart contracts, smart escrows, and smart vaults written in Rust. It compiles code to WebAssembly and handles deployment to XRPL networks. Bedrock supports three primitives:

- **Smart Contract** — Custom Rust WASM logic deployed via `ContractCreate`/`ContractCall`
- **Smart Escrow** — Conditional payments with WASM conditions via `EscrowCreate`/`EscrowFinish`
- **Smart Vault** — Asset custody with WASM deposit/withdraw logic via `VaultCreate`/`VaultDeposit`/`VaultWithdraw`

## Command Reference

### Project Initialization

```bash
# Create a new project (interactive primitive selection)
bedrock init <project-name>

# Create a project with specific primitives
bedrock init <project-name> --primitives contract
bedrock init <project-name> --primitives escrow
bedrock init <project-name> --primitives vault

# Multiple primitives in one project
bedrock init <project-name> --primitives contract,escrow,vault

# With a specific template
bedrock init <project-name> --primitives vault --template vault-whitelist
```

In interactive mode, `bedrock init` presents a menu to select a primitive. In non-interactive mode, it defaults to `contract`.

### Adding Primitives to an Existing Project

```bash
# Add a primitive to an existing project
bedrock add contract
bedrock add escrow
bedrock add vault

# With a specific template
bedrock add vault --template vault-whitelist
```

### Building

```bash
# Build all primitives in the project
bedrock build

# Build a specific primitive
bedrock build --type contract
bedrock build --type escrow
bedrock build --type vault

# Build in debug mode (faster compilation)
bedrock build --release=false
```

**Build targets per primitive:**

| Primitive | WASM Target | Rust Edition |
|-----------|-------------|--------------|
| Contract  | `wasm32-unknown-unknown` | 2021 |
| Escrow    | `wasm32v1-none` | 2024 |
| Vault     | `wasm32v1-none` | 2024 |

### Deploying Smart Contracts

```bash
# Deploy to alphanet (default)
bedrock deploy

# Deploy to local node
bedrock deploy --network local

# Deploy with specific wallet
bedrock deploy --wallet <seed>

# Skip auto-build or ABI generation
bedrock deploy --skip-build
bedrock deploy --skip-abi
```

**Important deployment details:**
- Deployment fee: 100 XRP (100000000 drops)
- Auto-generates ABI from Rust annotations
- Creates new wallet if none specified

### Calling Contract Functions

```bash
# Basic call
bedrock call <contract-address> <function-name> --wallet <seed>

# With JSON parameters
bedrock call <contract-address> <function-name> \
  --wallet <seed> \
  --params '{"param1": "value1", "param2": 123}'

# Parameters from file
bedrock call <contract-address> <function-name> \
  --wallet <seed> \
  --params-file params.json

# With custom gas and network
bedrock call <contract-address> <function-name> \
  --wallet <seed> \
  --gas 1000000 \
  --network alphanet
```

**Important call details:**
- Call fee: 1 XRP (1000000 drops)
- Function names are hex-encoded automatically
- `--wallet` is required

### Smart Vault Commands

```bash
# Deploy a smart vault
bedrock vault deploy --asset XRP --wallet <seed> --network local
bedrock vault deploy --asset USD --issuer <address> --wallet <seed> --network alphanet
bedrock vault deploy --asset XRP --max-capacity 1000000 --wallet <seed>
bedrock vault deploy --skip-build --wallet <seed>

# Deposit into a vault
bedrock vault deposit <vault-id> --amount <drops> --wallet <seed> --network local

# Withdraw from a vault
bedrock vault withdraw <vault-id> --amount <drops> --destination <address> --wallet <seed> --network local

# Check vault status
bedrock vault status <vault-id> --network local
```

**Vault deploy flags:**
- `--asset` — Asset currency code (default: `XRP`)
- `--issuer` — Asset issuer (required for non-XRP assets)
- `--max-capacity` — Maximum vault capacity
- `--wallet` — Wallet seed or jade keystore name (required)
- `--network` — Network: `local` (default) or `alphanet`
- `--fee` — Transaction fee in drops
- `--skip-build` — Skip building WASM

**Smart vault WASM functions:**
- `on_deposit()` — Returns 1 to allow deposit, 0 to deny
- `on_withdraw()` — Returns 1 to allow withdrawal, 0 to deny

### Smart Escrow Commands

```bash
# Deploy a smart escrow
bedrock escrow deploy --destination <address> --amount <drops> --wallet <seed> --network local
bedrock escrow deploy --destination <address> --amount <drops> \
  --cancel-after <ripple-epoch> --finish-after <ripple-epoch> \
  --wallet <seed> --network alphanet
bedrock escrow deploy --skip-build --destination <address> --amount <drops> --wallet <seed>

# Finish (release) an escrow
bedrock escrow finish <owner> <sequence> --wallet <seed> --network local

# Cancel an escrow
bedrock escrow cancel <owner> <sequence> --wallet <seed> --network local

# Check escrow status
bedrock escrow status <owner> <sequence> --network local
```

**Escrow deploy flags:**
- `--destination` — Escrow beneficiary address (required)
- `--amount` — Amount in drops (required)
- `--cancel-after` — Cancel after time (ripple epoch timestamp)
- `--finish-after` — Finish after time (ripple epoch timestamp)
- `--wallet` — Wallet seed or jade keystore name
- `--network` — Network: `local` (default) or `alphanet`
- `--fee` — Transaction fee in drops
- `--skip-build` — Skip building WASM

**Smart escrow WASM function:**
- `finish()` — Returns 1 to release the escrow, 0 to keep locked

### Local Node Management

```bash
# Start local XRPL node (Docker required)
bedrock node start

# Stop the node
bedrock node stop

# Check status
bedrock node status

# View logs
bedrock node logs
```

The node uses the `lejamon/rippled-smart-contracts-vault:arm64` Docker image for all project types.

A ledger advancement daemon runs in the background (PID in `.bedrock/ledger-daemon.pid`).

**Local node endpoints:**
- WebSocket: `ws://localhost:6006`
- Faucet: `http://localhost:8080/faucet`

### Wallet Management (Jade)

```bash
# Create new wallet
bedrock jade new <name>
bedrock jade new <name> --algorithm ed25519

# Import existing wallet
bedrock jade import <name>

# List wallets
bedrock jade list

# Export wallet (shows seed)
bedrock jade export <name>

# Remove wallet
bedrock jade remove <name>
```

Wallets are encrypted and stored in `~/.config/bedrock/wallets/`. Wallet names can be used in place of seeds in `--wallet` flags across all commands.

### Other Commands

```bash
# Request testnet funds (generates new wallet)
bedrock faucet

# Request funds for a specific address
bedrock faucet --address <address>

# Request funds using a wallet seed
bedrock faucet --wallet <seed>

# Clean cached JS modules and dependencies
bedrock clean
```

## Network Information

### Alphanet (Testnet)
- WebSocket: `wss://alphanet.nerdnest.xyz`
- Faucet: `https://alphanet.faucet.nerdnest.xyz/accounts`
- Network ID: 21465

### Local Node
- WebSocket: `ws://localhost:6006`
- Faucet: `http://localhost:8080/faucet`

## Smart Contract Development

### Contract Structure

```rust
#![cfg_attr(target_arch = "wasm32", no_std)]

#[cfg(not(target_arch = "wasm32"))]
extern crate std;

use xrpl_wasm_macros::wasm_export;
use xrpl_wasm_std::host::trace::trace;

/// @xrpl-function my_function
#[wasm_export]
fn my_function() -> i32 {
    let _ = trace("Hello from XRPL Smart Contract!");
    0
}
```

### Smart Escrow Structure

```rust
#![no_std]

use xrpl_wasm_stdlib::finish;

#[no_mangle]
pub extern "C" fn finish() -> i32 {
    // Return 1 to release the escrow, 0 to keep locked
    1
}
```

### Smart Vault Structure

```rust
#![no_std]

use xrpl_wasm_stdlib::{on_deposit, on_withdraw};

#[no_mangle]
pub extern "C" fn on_deposit() -> i32 {
    // Return 1 to allow deposit, 0 to deny
    1
}

#[no_mangle]
pub extern "C" fn on_withdraw() -> i32 {
    // Return 1 to allow withdrawal, 0 to deny
    1
}
```

### ABI Annotation Syntax (Contracts only)

```rust
/// @xrpl-function <function_name>
/// @param <name> <TYPE> - <description>
/// @return <TYPE> - <description>
/// @flag 0  (required) or @flag 1 (optional)
```

### Supported Types (Contracts)

| Type | Description |
|------|-------------|
| `UINT8`, `UINT16`, `UINT32`, `UINT64`, `UINT128`, `UINT256` | Unsigned integers |
| `VL` | Variable length bytes/string |
| `ACCOUNT` | XRPL account address |
| `AMOUNT` | XRP or token amount |
| `CURRENCY` | Currency code |
| `ISSUE` | Currency + issuer pair |

### Available Templates

**Contract templates:** `basic` (default), `token`, `nft`, `contract-escrow`, `counter`

**Escrow templates:** `escrow-hello` (default), `escrow-oracle`

**Vault templates:** `vault-hello` (default), `vault-whitelist`

## Typical Development Workflows

### Smart Contract Workflow

1. `bedrock init my-contract --primitives contract && cd my-contract`
2. `bedrock node start`
3. Write contract in `contract/src/lib.rs`
4. `bedrock build`
5. `bedrock deploy --network local` — note the contract address and wallet seed
6. `bedrock call <contract> <function> --wallet <seed> --network local`
7. `bedrock deploy --network alphanet`

### Smart Escrow Workflow

1. `bedrock init my-escrow --primitives escrow && cd my-escrow`
2. `bedrock node start`
3. Write escrow logic in `escrow/src/lib.rs`
4. `bedrock build`
5. `bedrock escrow deploy --destination <addr> --amount 1000000 --wallet <seed> --network local`
6. `bedrock escrow status <owner> <sequence> --network local`
7. `bedrock escrow finish <owner> <sequence> --wallet <seed> --network local`

### Smart Vault Workflow

1. `bedrock init my-vault --primitives vault && cd my-vault`
2. `bedrock node start`
3. Write vault logic in `vault/src/lib.rs`
4. `bedrock build`
5. `bedrock vault deploy --asset XRP --wallet <seed> --network local`
6. `bedrock vault deposit <vault-id> --amount 1000000 --wallet <seed> --network local`
7. `bedrock vault withdraw <vault-id> --amount 500000 --destination <addr> --wallet <seed> --network local`
8. `bedrock vault status <vault-id> --network local`

## Configuration File (bedrock.toml)

### Contract Project

```toml
[project]
name = "my-contract"
version = "0.1.0"
authors = ["Your Name"]
primitives = ["contract"]

[build]
source = "contract/src/lib.rs"
output = "contract/target/wasm32-unknown-unknown/release"
target = "wasm32-unknown-unknown"

[contracts.main]
source = "contract/src/lib.rs"
abi = "contract/build/abi.json"

[local_node]
config_dir = ".bedrock/node-config"
docker_image = "lejamon/rippled_smart_contract_vault_x86"
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

### Escrow Project

```toml
[project]
name = "my-escrow"
version = "0.1.0"
authors = ["Your Name"]
primitives = ["escrow"]

[escrows.main]
source = "escrow/src/lib.rs"
output = "escrow/target/wasm32v1-none/release"

[local_node]
config_dir = ".bedrock/node-config"
docker_image = "lejamon/rippled_smart_contract_vault_x86"
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

### Vault Project

```toml
[project]
name = "my-vault"
version = "0.1.0"
authors = ["Your Name"]
primitives = ["vault"]

[vaults.main]
source = "vault/src/lib.rs"
output = "vault/target/wasm32v1-none/release"

[local_node]
config_dir = ".bedrock/node-config"
docker_image = "lejamon/rippled_smart_contract_vault_x86"
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

## Common Issues and Solutions

### Build fails with wasm32 target error
```bash
# For contracts
rustup target add wasm32-unknown-unknown

# For escrow/vault
rustup target add wasm32v1-none
```

### Node won't start
Ensure Docker is running: `docker ps`

### Module installation fails
Check Node.js version (18+ required): `node --version`

### Contract deployment fails
- Ensure wallet has sufficient XRP (100+ XRP for deployment)
- Check network connectivity to alphanet

## Project File Structures

### Contract Project

```
my-contract/
├── bedrock.toml
├── .wallets/
├── .bedrock/
│   └── node-config/
│       ├── genesis.json
│       ├── xrpld.cfg
│       └── validators.txt
├── contract/
│   ├── Cargo.toml
│   └── src/
│       └── lib.rs
├── abi.json              # Generated ABI (after deploy)
└── README.md
```

### Escrow Project

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

### Vault Project

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

## Best Practices

1. Always use release mode for production deployments
2. Test on local node before deploying to alphanet
3. Keep wallet seeds secure - use `bedrock jade` for encrypted storage
4. Include descriptive ABI annotations for all exported contract functions
5. Optimize with `opt-level = "z"` and `lto = true` in Cargo.toml
6. Use `bedrock add` to combine multiple primitives in a single project
