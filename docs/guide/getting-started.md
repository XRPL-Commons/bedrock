# Getting Started

This guide walks you through installing Bedrock and creating your first XRPL project.

## Prerequisites

Before installing Bedrock, ensure you have the following tools:

| Tool | Version | Purpose |
|------|---------|---------|
| [Go](https://go.dev/dl/) | 1.21+ | Building Bedrock from source |
| [Node.js](https://nodejs.org/) | 18+ | XRPL transaction handling |
| [Rust](https://rustup.rs/) | 1.70+ | Compiling smart contracts |
| [Docker](https://www.docker.com/) | Latest | Local XRPL node (optional) |

### Verify Prerequisites

```bash
go version      # Should show 1.21+
node --version  # Should show v18+
rustc --version # Should show 1.70+
cargo --version
```

### Installing Rust

If you don't have Rust installed:

```bash
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source $HOME/.cargo/env
```

## Installation

### Install from Source

```bash
# Clone the repository
git clone https://github.com/XRPL-Commons/bedrock.git
cd bedrock

# Build and install
go build -o bedrock cmd/bedrock/main.go
sudo mv bedrock /usr/local/bin/

# Verify installation
bedrock --help
```

### First Run

On first run, Bedrock automatically installs JavaScript dependencies:

```
First run detected - installing JavaScript dependencies...
```

This takes ~10-15 seconds and only happens once. Dependencies are cached in `~/.cache/bedrock/modules/`.

## Choose Your Primitive

Bedrock supports three XRPL primitives. Each has its own project structure, build target, and deployment flow:

| Primitive | Description | WASM Entry Points |
|-----------|-------------|-------------------|
| **Smart Contract** | Custom logic via `ContractCreate`/`ContractCall` | `@xrpl-function` annotated functions |
| **Smart Escrow** | Conditional payments via `EscrowCreate`/`EscrowFinish` | `finish()` |
| **Smart Vault** | Asset custody via `VaultCreate`/`VaultDeposit`/`VaultWithdraw` | `on_deposit()`, `on_withdraw()` |

### Initialize a Project

```bash
# Interactive mode — prompts you to select a primitive
bedrock init my-project

# Or specify directly
bedrock init my-contract --primitives contract
bedrock init my-escrow --primitives escrow
bedrock init my-vault --primitives vault

# With a specific template
bedrock init my-vault --primitives vault --template vault-whitelist

# Multiple primitives in one project
bedrock init my-project --primitives contract,escrow
```

In interactive mode, Bedrock presents a menu:

```
Select a primitive:

  1. Contract  - Custom Rust WASM logic (ContractCreate/ContractCall)
  2. Escrow    - Conditional payments with WASM conditions (EscrowCreate/EscrowFinish)
  3. Vault     - Asset custody with WASM deposit/withdraw logic (VaultCreate/VaultDeposit/VaultWithdraw)
```

### Adding Primitives Later

You can add a new primitive to an existing project at any time:

```bash
bedrock add escrow
bedrock add vault --template vault-whitelist
bedrock add contract --template token
```

## Quick Start by Primitive

### Smart Contract

```bash
bedrock init my-contract --primitives contract && cd my-contract
bedrock node start
bedrock build
bedrock deploy --network local
# Note the contract address and wallet seed
bedrock call <contract> hello --wallet <seed> --network local
```

See the full guide: **[Building Contracts](/guide/building-contracts)**, **[ABI Generation](/guide/abi-generation)**, **[Deploying & Calling](/guide/deployment-and-calling)**

### Smart Escrow

```bash
bedrock init my-escrow --primitives escrow && cd my-escrow
bedrock node start
bedrock build
bedrock escrow deploy --destination <addr> --amount 1000000 --wallet <seed> --network local
bedrock escrow status <owner> <sequence> --network local
bedrock escrow finish <owner> <sequence> --wallet <seed> --network local
```

See the full guide: **[Smart Escrows](/guide/smart-escrows)**

### Smart Vault

```bash
bedrock init my-vault --primitives vault && cd my-vault
bedrock node start
bedrock build
bedrock vault deploy --asset XRP --wallet <seed> --network local
bedrock vault deposit <vault-id> --amount 1000000 --wallet <seed> --network local
bedrock vault status <vault-id> --network local
```

See the full guide: **[Smart Vaults](/guide/smart-vaults)**

## Building

```bash
# Build all primitives in the project
bedrock build

# Build a specific primitive
bedrock build --type contract
bedrock build --type escrow
bedrock build --type vault
```

## Next Steps

- **[Building Contracts](/guide/building-contracts)** - Deep dive into the contract build system
- **[ABI Generation](/guide/abi-generation)** - Learn the annotation syntax (contracts)
- **[Deploying & Calling](/guide/deployment-and-calling)** - Full contract deployment guide
- **[Smart Escrows](/guide/smart-escrows)** - Conditional payments with WASM logic
- **[Smart Vaults](/guide/smart-vaults)** - Asset custody with WASM logic
- **[Local Node](/guide/local-node)** - Configure your local environment
- **[Wallet Management](/guide/wallet)** - Secure wallet handling
