# Quick Reference

A quick lookup guide for common Bedrock operations.

## Commands at a Glance

| Command | Purpose |
|---------|---------|
| `bedrock init <name>` | Create new project |
| `bedrock init <name> -p escrow` | Create escrow project |
| `bedrock init <name> -p vault` | Create vault project |
| `bedrock add <primitive>` | Add primitive to existing project |
| `bedrock build` | Compile all primitives to WASM |
| `bedrock build --type vault` | Build specific primitive |
| `bedrock deploy` | Deploy smart contract |
| `bedrock call <addr> <fn>` | Call contract function |
| `bedrock escrow deploy` | Deploy smart escrow |
| `bedrock escrow finish <owner> <seq>` | Release escrow |
| `bedrock escrow cancel <owner> <seq>` | Cancel escrow |
| `bedrock escrow status <owner> <seq>` | Check escrow status |
| `bedrock vault deploy` | Deploy smart vault |
| `bedrock vault deposit <id>` | Deposit into vault |
| `bedrock vault withdraw <id>` | Withdraw from vault |
| `bedrock vault status <id>` | Check vault status |
| `bedrock node start/stop` | Manage local node |
| `bedrock jade new <name>` | Create wallet |
| `bedrock faucet` | Get testnet funds |
| `bedrock clean` | Clean build artifacts |

## Quick Start — Smart Contract

```bash
bedrock init my-app --primitives contract && cd my-app
bedrock node start
bedrock deploy --network local
bedrock call <contract> hello --wallet <seed> --network local
```

## Quick Start — Smart Escrow

```bash
bedrock init my-escrow --primitives escrow && cd my-escrow
bedrock node start
bedrock build
bedrock escrow deploy --destination <addr> --amount 1000000 --wallet <seed> --network local
bedrock escrow finish <owner> <seq> --wallet <seed> --network local
```

## Quick Start — Smart Vault

```bash
bedrock init my-vault --primitives vault && cd my-vault
bedrock node start
bedrock build
bedrock vault deploy --asset XRP --wallet <seed> --network local
bedrock vault deposit <vault-id> --amount 1000000 --wallet <seed> --network local
bedrock vault withdraw <vault-id> --amount 500000 --destination <addr> --wallet <seed> --network local
```

## Networks

| Network | WebSocket | Faucet |
|---------|-----------|--------|
| Local | `ws://localhost:6006` | `http://localhost:8080/faucet` |
| Alphanet | `wss://alphanet.nerdnest.xyz` | `https://alphanet.faucet.nerdnest.xyz/accounts` |

## Deploy Options (Contracts)

```bash
bedrock deploy                      # Default (alphanet, auto-build)
bedrock deploy --network local      # Local node
bedrock deploy --wallet <seed>      # Specific wallet
bedrock deploy --skip-build         # Skip rebuild
```

## Escrow Deploy Options

```bash
bedrock escrow deploy \
  --destination <addr>              # Beneficiary (required)
  --amount <drops>                  # Amount (required)
  --wallet <seed>                   # Wallet seed or jade name
  --network local                   # Target network
  --cancel-after <epoch>            # Cancel after time
  --finish-after <epoch>            # Finish after time
```

## Vault Deploy Options

```bash
bedrock vault deploy \
  --asset XRP                       # Asset (default: XRP)
  --issuer <addr>                   # Issuer (for non-XRP)
  --max-capacity <amount>           # Max capacity
  --wallet <seed>                   # Wallet seed or jade name
  --network local                   # Target network
```

## Call Options (Contracts)

```bash
bedrock call <contract> <function> --wallet <seed>
  --params '{"key":"value"}'        # Inline JSON
  --params-file params.json         # From file
  --gas 1000000                     # Computation limit
  --network alphanet                # Target network
```

## Wallet Commands (Jade)

```bash
bedrock jade new <name>             # Create encrypted wallet
bedrock jade import <name>          # Import from seed
bedrock jade list                   # List all wallets
bedrock jade export <name>          # Show seed
bedrock jade remove <name>          # Delete wallet
```

Wallet names can be used in `--wallet` flags across all commands.

## ABI Annotations (Contracts Only)

```rust
/// @xrpl-function function_name
/// @param name TYPE - description
/// @return TYPE - description
/// @flag 0  // required (default)
/// @flag 1  // optional
```

## XRPL Types (Contracts)

| Type | Use for |
|------|---------|
| `UINT8/16/32/64/128/256` | Integers |
| `VL` | Bytes/strings |
| `ACCOUNT` | Addresses |
| `AMOUNT` | XRP/token amounts |
| `CURRENCY` | Currency codes |
| `ISSUE` | Currency+issuer |

## Build Targets

| Primitive | WASM Target | Rust Edition |
|-----------|-------------|--------------|
| Contract | `wasm32-unknown-unknown` | 2021 |
| Escrow | `wasm32v1-none` | 2024 |
| Vault | `wasm32v1-none` | 2024 |

## WASM Entry Points

| Primitive | Function | Returns |
|-----------|----------|---------|
| Contract | `@xrpl-function` annotated | Status code |
| Escrow | `finish()` | `1` = release, `0` = keep locked |
| Vault | `on_deposit()` | `1` = allow, `0` = deny |
| Vault | `on_withdraw()` | `1` = allow, `0` = deny |

## Templates

| Primitive | Templates |
|-----------|-----------|
| Contract | `basic`, `token`, `nft`, `contract-escrow`, `counter` |
| Escrow | `escrow-hello`, `escrow-oracle` |
| Vault | `vault-hello`, `vault-whitelist` |

## Troubleshooting

| Problem | Solution |
|---------|----------|
| WASM target missing (contract) | `rustup target add wasm32-unknown-unknown` |
| WASM target missing (escrow/vault) | `rustup target add wasm32v1-none` |
| Node won't start | Check Docker: `docker ps` |
| Deployment fails | Ensure 100+ XRP balance |
| Modules not found | Check Node.js 18+: `node --version` |

## Useful Paths

| Path | Contents |
|------|----------|
| `~/.config/bedrock/wallets/` | Encrypted wallets |
| `~/.cache/bedrock/modules/` | JS module cache |
