# Commands Reference

Complete reference for all Bedrock CLI commands with detailed options and examples.

## init

Create a new Bedrock project.

```bash
bedrock init [project-name] [flags]
```

| Argument | Description |
|----------|-------------|
| `project-name` | Name of the project directory to create |

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--primitives` | `-p` | Comma-separated primitives (contract, escrow, vault) | Interactive |
| `--template` | `-t` | Project template | Primitive default |

In interactive mode (no `--primitives` flag), Bedrock presents a menu to select a primitive.

```bash
bedrock init my-project                               # Interactive
bedrock init my-contract --primitives contract         # Smart contract
bedrock init my-escrow --primitives escrow             # Smart escrow
bedrock init my-vault --primitives vault               # Smart vault
bedrock init my-project --primitives contract,escrow   # Multiple primitives
bedrock init my-vault -p vault -t vault-whitelist      # With template
```

**Available templates:**

| Primitive | Templates |
|-----------|-----------|
| Contract | `basic` (default), `token`, `nft`, `contract-escrow`, `counter` |
| Escrow | `escrow-hello` (default), `escrow-oracle` |
| Vault | `vault-hello` (default), `vault-whitelist` |

## add

Add a primitive to an existing project.

```bash
bedrock add <primitive> [flags]
```

| Argument | Description |
|----------|-------------|
| `primitive` | Primitive to add: `contract`, `escrow`, or `vault` |

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--template` | `-t` | Template for the new primitive | Primitive default |

```bash
bedrock add escrow
bedrock add vault --template vault-whitelist
bedrock add contract --template token
```

## build

Compile Rust source to WebAssembly.

```bash
bedrock build [flags]
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--release` | `-r` | Build in release mode (optimized) | `true` |
| `--type` | | Build specific primitive (contract, escrow, vault) | All |

By default, builds all primitives configured in the project.

```bash
bedrock build                    # Build all primitives
bedrock build --type contract    # Build only contract
bedrock build --type escrow      # Build only escrow
bedrock build --type vault       # Build only vault
bedrock build --release=false    # Debug build (faster)
```

**Build targets:**

| Primitive | WASM Target | Output |
|-----------|-------------|--------|
| Contract | `wasm32-unknown-unknown` | `contract/target/wasm32-unknown-unknown/release/` |
| Escrow | `wasm32v1-none` | `escrow/target/wasm32v1-none/release/` |
| Vault | `wasm32v1-none` | `vault/target/wasm32v1-none/release/` |

## deploy

Deploy a smart contract to an XRPL network.

```bash
bedrock deploy [flags]
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--network` | `-n` | Target network (local, alphanet) | `alphanet` |
| `--wallet` | `-w` | Wallet seed or jade name | Auto-generated |
| `--skip-build` | | Skip automatic contract rebuild | `false` |
| `--skip-abi` | | Skip ABI generation | `false` |
| `--abi` | `-a` | Path to ABI file | `abi.json` |
| `--algorithm` | | Cryptographic algorithm (secp256k1, ed25519) | `secp256k1` |

**Transaction fee:** 100 XRP (100,000,000 drops)

```bash
bedrock deploy                          # Deploy to alphanet
bedrock deploy --network local          # Deploy to local node
bedrock deploy --wallet sEd7...         # Use specific wallet
bedrock deploy --skip-build             # Skip rebuild
```

## call

Call a function on a deployed smart contract.

```bash
bedrock call <contract> <function> [flags]
```

| Argument | Description |
|----------|-------------|
| `contract` | The contract's XRPL account address (rXXX...) |
| `function` | Name of the function to call |

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--wallet` | `-w` | Wallet seed or jade name (required) | - |
| `--network` | `-n` | Target network | `alphanet` |
| `--params` | `-p` | JSON string of function parameters | - |
| `--params-file` | `-f` | Path to JSON file with parameters | - |
| `--gas` | `-g` | Computation allowance | `1000000` |
| `--fee` | | Transaction fee in drops | `1000000` |
| `--abi` | `-a` | Path to ABI file | `abi.json` |
| `--algorithm` | | Cryptographic algorithm | `secp256k1` |

**Transaction fee:** 1 XRP (1,000,000 drops) by default

```bash
bedrock call rContract... hello --wallet sEd7...
bedrock call rContract... transfer --wallet sEd7... --params '{"to":"rRecipient...","amount":1000}'
bedrock call rContract... register --wallet sEd7... --params-file params.json
bedrock call rContract... expensive_op --wallet sEd7... --gas 5000000 --fee 2000000
bedrock call rContract... test --wallet sEd7... --network local
```

## escrow

Manage XRPL smart escrows. See the full guide: [Smart Escrows](/guide/smart-escrows).

### escrow deploy

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
bedrock escrow deploy --destination rXXX... --amount 1000000 --wallet sXXX... --network local
```

### escrow finish

Finish (release) a smart escrow. The on-chain WASM `finish()` function executes.

```bash
bedrock escrow finish <owner> <sequence> [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--wallet` | Wallet seed or jade name (required) | - |
| `--network` | Network | `local` |
| `--fee` | Transaction fee in drops | - |

### escrow cancel

Cancel a smart escrow.

```bash
bedrock escrow cancel <owner> <sequence> [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--wallet` | Wallet seed or jade name (required) | - |
| `--network` | Network | `local` |
| `--fee` | Transaction fee in drops | - |

### escrow status

Query escrow status.

```bash
bedrock escrow status <owner> <sequence> [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--network` | Network | `local` |

## vault

Manage XRPL smart vaults. See the full guide: [Smart Vaults](/guide/smart-vaults).

### vault deploy

Create a smart vault with WASM logic.

```bash
bedrock vault deploy [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--asset` | Asset currency code | `XRP` |
| `--issuer` | Asset issuer (for non-XRP assets) | - |
| `--max-capacity` | Maximum vault capacity | - |
| `--wallet` | Wallet seed or jade name | - |
| `--network` | Network (local, alphanet) | `local` |
| `--fee` | Transaction fee in drops | - |
| `--skip-build` | Skip building WASM | `false` |

```bash
bedrock vault deploy --asset XRP --wallet sXXX... --network local
bedrock vault deploy --asset USD --issuer rXXX... --wallet sXXX... --network alphanet
```

### vault deposit

Deposit into a smart vault.

```bash
bedrock vault deposit <vault-id> [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--amount` | Amount in drops (required) | - |
| `--wallet` | Wallet seed or jade name (required) | - |
| `--network` | Network | `local` |
| `--fee` | Transaction fee in drops | - |

### vault withdraw

Withdraw from a smart vault.

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

### vault status

Query vault status.

```bash
bedrock vault status <vault-id> [flags]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--network` | Network | `local` |

## node

Manage a local XRPL development node running in Docker.

```bash
bedrock node <command>
```

| Command | Description |
|---------|-------------|
| `start` | Start the local XRPL node |
| `stop` | Stop the running node |
| `status` | Check if the node is running |
| `logs` | View node container logs |

The node auto-detects the project type and configures accordingly:
- **Contract projects:** Uses `transia/cluster` image with genesis file
- **Escrow/Vault projects:** Uses `willemolding/rippled:smart-vaults.0` in standalone mode

**Requirements:** Docker must be installed and running.

```bash
bedrock node start     # Start local node
bedrock node status    # Check status
bedrock node logs      # View logs
bedrock node stop      # Stop node
```

## jade

Manage XRPL wallets with encrypted local storage. Wallet names can be used in place of seeds in `--wallet` flags across all commands.

### jade new

```bash
bedrock jade new <name> [--algorithm secp256k1|ed25519]
```

Creates a new XRPL wallet, encrypts it, and stores it locally.

### jade import

```bash
bedrock jade import <name> [--algorithm secp256k1|ed25519]
```

Imports an existing wallet from a seed. You'll be prompted to enter the seed securely.

### jade list

```bash
bedrock jade list
```

Lists all stored wallets.

### jade export

```bash
bedrock jade export <name>
```

Exports a wallet's seed and address (password required).

### jade remove

```bash
bedrock jade remove <name>
```

Permanently deletes a stored wallet.

**Storage location:** `~/.config/bedrock/wallets/<name>.json`
**Encryption:** AES-256-GCM with PBKDF2 key derivation

## faucet

Request testnet funds from the XRPL faucet.

```bash
bedrock faucet [flags]
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--network` | `-n` | Target network | `alphanet` |
| `--wallet` | `-w` | Wallet seed | - |
| `--address` | `-a` | Wallet address | - |
| `--algorithm` | | Cryptographic algorithm | `secp256k1` |

If neither `--wallet` nor `--address` is provided, a new wallet is generated automatically.

```bash
bedrock faucet                         # Generate new wallet and fund it
bedrock faucet --address rMyAddr...    # Fund specific address
bedrock faucet --wallet sEd7...        # Fund existing wallet
bedrock faucet --network local         # Fund on local network
```

## clean

Remove build artifacts and cached files.

```bash
bedrock clean
```

**What it removes:**
- Extracted JavaScript modules (deploy.js, call.js, faucet.js)
- Installed npm dependencies (node_modules)
- Version tracking file

After cleaning, the next command that requires JS modules will automatically reinstall dependencies.

## Global Flags

| Flag | Description |
|------|-------------|
| `--help` | Display help for the command |
| `--version` | Display Bedrock version |
