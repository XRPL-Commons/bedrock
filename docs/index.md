---
description: A CLI tool for developing, deploying, and interacting with XRPL smart contracts, smart escrows, and smart vaults written in Rust. Think Foundry, but for XRPL.
---

# Introduction to Bedrock

<DownloadLLMsFullDoc />

Bedrock is a developer tool for building, deploying, and interacting with XRPL smart contracts, smart escrows, and smart vaults written in Rust. Think **Foundry**, but for XRPL.

## What is Bedrock?

Bedrock provides a complete CLI workflow for XRPL smart features development. It compiles Rust code to WebAssembly and handles deployment to XRPL networks. Bedrock supports three XRPL primitives:

- **Smart Contract** — Custom Rust WASM logic deployed via `ContractCreate`/`ContractCall`
- **Smart Escrow** — Conditional payments with WASM conditions via `EscrowCreate`/`EscrowFinish`
- **Smart Vault** — Asset custody with WASM deposit/withdraw logic via `VaultCreate`/`VaultDeposit`/`VaultWithdraw`

It includes:

- **Build System** - Compile Rust code to optimized WebAssembly for all three primitives
- **Smart Deployment** - Auto-build, ABI generation, and deployment in one command
- **Contract Interaction** - Call deployed contract functions with typed parameters
- **Escrow Management** - Deploy, finish, cancel, and inspect smart escrows
- **Vault Management** - Deploy vaults, deposit, withdraw, and check status
- **Local Node** - Manage a local XRPL test network via Docker
- **ABI Generation** - Automatic ABI extraction from Rust code annotations (contracts)
- **Wallet Management** - Encrypted wallet storage with Jade

## Why Use Bedrock?

Building and deploying XRPL smart primitives involves multiple tools and manual steps. Bedrock abstracts away the complexity of:

- **WASM compilation** - Sensible defaults for Rust-to-WASM compilation across all primitives
- **ABI management** - Auto-generated from code annotations, no manual maintenance
- **Deployment orchestration** - One command to build, generate ABI, and deploy
- **Network configuration** - Pre-configured for local and alphanet environments
- **Wallet security** - Encrypted storage so seeds don't leak into shell history

## Key Features

### Three Primitives

Build smart contracts, smart escrows, and smart vaults — each with its own WASM logic, deployment flow, and interaction commands.

### Build System

Compile Rust code to optimized WebAssembly with a single command. Bedrock auto-detects the WASM target for each primitive (`wasm32-unknown-unknown` for contracts, `wasm32v1-none` for escrows and vaults).

### Smart Deployment

`bedrock deploy` automatically builds your contract, generates the ABI, and deploys. For escrows and vaults, dedicated `bedrock escrow deploy` and `bedrock vault deploy` commands handle their specific deployment flows.

### Local Development

Spin up a local XRPL node in Docker for fast iteration. Bedrock auto-detects your project type and configures the node accordingly.

### Automatic ABI Generation

Annotate your Rust functions with JSDoc-style comments and Bedrock extracts the ABI automatically. No separate ABI files to maintain. (Applies to smart contracts only — escrows and vaults don't require ABI.)

### Wallet Management

Create, import, and manage XRPL wallets with AES-256-GCM encryption. Your seeds are never stored in plaintext. Use wallet names in place of seeds in any `--wallet` flag.

## Architecture Overview

Bedrock uses a hybrid architecture with Go for the CLI and embedded JavaScript modules for XRPL transaction handling:

```
bedrock CLI (Go)
       |
       ├── Build System ──────── cargo (Rust → WASM)
       |
       ├── ABI Generator ─────── Parses Rust annotations (contracts)
       |
       ├── Contract Deployer ─── Embedded JS (deploy.js)
       ├── Escrow Operator ───── Embedded JS (escrow_deploy/finish/cancel.js)
       ├── Vault Operator ────── Embedded JS (vault_deploy/deposit/withdraw.js)
       |
       ├── Caller ────────────── Embedded JS (call.js)
       |
       ├── Local Node ────────── Docker (rippled)
       |
       └── Wallet Manager ────── AES-256-GCM encryption
```

## Supported Networks

| Network  | WebSocket                     | Faucet                                          |
| -------- | ----------------------------- | ----------------------------------------------- |
| Local    | `ws://localhost:6006`         | `http://localhost:8080/faucet`                  |
| Alphanet | `wss://alphanet.nerdnest.xyz` | `https://alphanet.faucet.nerdnest.xyz/accounts` |

## Requirements

Before installing Bedrock, ensure you have:

- **[Go](https://go.dev/dl/)** (1.21 or later) - For building Bedrock from source
- **[Node.js](https://nodejs.org/)** (18 or later) - For XRPL transaction handling
- **[Rust](https://rustup.rs/)** - For compiling smart contracts
- **[Docker](https://www.docker.com/)** (optional) - For running a local XRPL node

## Next Steps

Ready to get started? Here's the recommended path:

1. **[Getting Started](/guide/getting-started)** - Install Bedrock and create your first project
2. **[Building Contracts](/guide/building-contracts)** - Understand the build system
3. **[ABI Generation](/guide/abi-generation)** - Learn the annotation syntax (contracts)
4. **[Deploying & Calling](/guide/deployment-and-calling)** - Deploy and interact with contracts
5. **[Smart Escrows](/guide/smart-escrows)** - Create conditional payments with WASM logic
6. **[Smart Vaults](/guide/smart-vaults)** - Build asset custody with WASM deposit/withdraw logic
7. **[Local Node](/guide/local-node)** - Set up a local development environment
8. **[Wallet Management](/guide/wallet)** - Manage your XRPL wallets securely
9. **[Commands Reference](/guide/commands-reference)** - Complete CLI reference

## Community & Support

- **GitHub** - [XRPL-Commons/Bedrock](https://github.com/XRPL-Commons/bedrock)
- **Issues** - Report bugs or request features on GitHub
- **XRPL Commons** - [xrpl-commons.org](https://www.xrpl-commons.org)
- **XRPL Docs** - [xrpl.org](https://xrpl.org/)

## License

MIT License - See the [LICENSE](https://github.com/XRPL-Commons/bedrock/blob/main/LICENSE) file for details.
