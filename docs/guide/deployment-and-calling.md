# Deploying & Calling Contracts

This guide covers the `bedrock deploy` and `bedrock call` commands for deploying and interacting with your smart contracts.

## Deploying Contracts

### Smart Deployment

`bedrock deploy` automatically performs the following steps:

1. **Builds the contract** - Compiles your Rust contract to WASM in release mode
2. **Generates the ABI** - Extracts the ABI from your Rust source code annotations
3. **Deploys to the network** - Deploys the compiled WASM to the specified network

### Usage

```bash
bedrock deploy [flags]
```

### Options

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--network` | `-n` | Target network (local, alphanet) | `local` |
| `--wallet` | `-w` | Wallet seed for signing transactions | Auto-generated |
| `--skip-build` | | Skip automatic contract rebuild | `false` |
| `--skip-abi` | | Skip ABI generation | `false` |
| `--abi` | `-a` | Path to ABI file | `abi.json` |
| `--algorithm` | | Cryptographic algorithm (secp256k1, ed25519) | `secp256k1` |

### Examples

```bash
# Deploy to local node (default)
bedrock deploy

# Deploy to alphanet (testnet)
bedrock deploy --network alphanet

# Deploy with existing wallet
bedrock deploy --wallet sEd7...

# Deploy without rebuilding
bedrock deploy --skip-build

# Full example with all options
bedrock deploy \
  --network alphanet \
  --wallet sEd7... \
  --algorithm secp256k1
```

### Output

On success, the deploy command displays:
- **Wallet address and seed** - Save these for future interactions
- **Contract account address** - The deployed contract's address
- **Transaction hash** - For verification on the ledger

### Transaction Fees

| Operation | Fee |
|-----------|-----|
| ContractCreate | 100 XRP (100,000,000 drops) |

Ensure your wallet has sufficient balance before deploying.

## Calling Contract Functions

### Usage

```bash
bedrock call <contract> <function> [flags]
```

### Arguments

| Argument | Description |
|----------|-------------|
| `contract` | The contract's XRPL account address (rXXX...) |
| `function` | Name of the function to call |

### Options

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--wallet` | `-w` | Wallet seed for signing (required) | - |
| `--network` | `-n` | Target network | `local` |
| `--params` | `-p` | JSON string of function parameters | - |
| `--params-file` | `-f` | Path to JSON file with parameters | - |
| `--gas` | `-g` | Computation allowance | `1000000` |
| `--fee` | | Transaction fee in drops | `1000000` |
| `--abi` | `-a` | Path to ABI file | `abi.json` |
| `--algorithm` | | Cryptographic algorithm | `secp256k1` |

### Parameter Passing

Parameters can be passed in two ways:

**Inline JSON:**
```bash
bedrock call rContract123... transfer \
  --wallet sEd7... \
  --params '{"to":"rRecipient...","amount":1000}'
```

**From a JSON file:**
```bash
bedrock call rContract123... register \
  --wallet sEd7... \
  --params-file params.json
```

Parameter names must match the ABI definition.

### Examples

```bash
# Simple call without parameters
bedrock call rContract123... hello --wallet sEd7...

# Call with inline parameters
bedrock call rContract123... transfer \
  --wallet sEd7... \
  --params '{"to":"rRecipient...","amount":1000}'

# Call with parameters from file
bedrock call rContract123... register \
  --wallet sEd7... \
  --params-file ./params.json

# Call with custom gas and fee
bedrock call rContract123... expensive_operation \
  --wallet sEd7... \
  --gas 5000000 \
  --fee 2000000

# Call on local network
bedrock call rContract123... test_function \
  --wallet sEd7... \
  --network local
```

### Transaction Fees

| Operation | Fee |
|-----------|-----|
| ContractCall | 1 XRP (1,000,000 drops) by default |

Increase the fee for complex operations using `--fee`.

## Common Workflows

### Local Development

```bash
# Start local node
bedrock node start

# Deploy to local
bedrock deploy --network local
# Note: wallet seed = sXXX..., contract = rXXX...

# Call contract
bedrock call rXXX... hello --wallet sXXX... --network local

# Make changes, redeploy
bedrock deploy --network local
```

### Deploying to Testnet

```bash
# Deploy to alphanet
bedrock deploy --network alphanet
# Save the wallet seed and contract address from output

# Interact with the contract
bedrock call rContract... myFunction \
  --wallet sXXX... \
  --network alphanet
```

### Using a Saved Wallet

```bash
# Create and fund a wallet first
bedrock jade new my-wallet
bedrock faucet --wallet <seed>

# Deploy with that wallet
bedrock deploy --wallet <seed> --network alphanet

# Call with the same wallet
bedrock call rContract... hello --wallet <seed>
```
