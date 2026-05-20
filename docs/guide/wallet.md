# Wallet Management

**Jade** is Bedrock's built-in wallet management tool. It allows you to create, import, and manage XRPL wallets securely with encrypted local storage.

## Overview

Jade provides a secure way to manage your XRPL wallets by encrypting them and storing them on disk. This means you don't have to expose your wallet seeds in your command-line history or scripts.

**Security features:**
- AES-256-GCM encryption with PBKDF2-HMAC-SHA256 key derivation
  (600,000 iterations, aligned with OWASP 2023 baseline)
- Per-keystore iteration count, so future tightening of KDF parameters
  does not break existing wallets
- Password-protected access
- Supports secp256k1 and ed25519 algorithms
- Stored in `~/.config/bedrock/wallets/` (directory mode `0700`,
  keystore files `0600`)
- Wallet names like `swap-vault` (starting with `s`) are resolved as
  keystore lookups, not raw seeds, in every `--wallet` flag

## Commands

### `bedrock jade new <name>`

Creates a new XRPL wallet.

```bash
bedrock jade new <name> [flags]
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--algorithm` | `-a` | Cryptographic algorithm (secp256k1, ed25519) | `secp256k1` |

**What it does:**
1. Generates a new, random XRPL wallet
2. Prompts you to enter a password to encrypt the wallet
3. Saves the encrypted wallet to `~/.config/bedrock/wallets/<name>.json`

```bash
bedrock jade new my-dev-wallet
bedrock jade new my-ed-wallet --algorithm ed25519
```

### `bedrock jade import <name>`

Imports an existing XRPL wallet from a seed.

```bash
bedrock jade import <name> [flags]
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--algorithm` | `-a` | Wallet's cryptographic algorithm | `secp256k1` |

**What it does:**
1. Prompts you to enter the seed (input is hidden)
2. Prompts you to enter a password to encrypt the wallet
3. Saves the encrypted wallet

```bash
bedrock jade import my-existing-wallet
bedrock jade import my-ed-wallet --algorithm ed25519
```

### `bedrock jade list`

Lists all stored wallets.

```bash
bedrock jade list
```

### `bedrock jade export <name>`

Exports a wallet's seed and address.

```bash
bedrock jade export <name>
```

**What it does:**
1. Prompts you to enter the password
2. Displays the wallet's name, address, and seed

### `bedrock jade remove <name>`

Permanently removes a wallet from storage.

```bash
bedrock jade remove <name>
```

**This action cannot be undone.** Make sure you have backed up your wallet's seed before removing it.

## Using Wallets with Other Commands

Once you've exported a wallet's seed, you can use it with other Bedrock commands:

```bash
# Export the seed
bedrock jade export my-wallet
# Output: Seed: sXXX...

# Use the seed for deployment
bedrock deploy --wallet sXXX... --network alphanet

# Use the seed for contract calls
bedrock call rContract... hello --wallet sXXX...
```

## Requesting Testnet Funds

Use the `bedrock faucet` command to fund a wallet:

```bash
# Generate a new wallet and fund it
bedrock faucet

# Fund a specific address
bedrock faucet --address rMyAddress123...

# Fund using an existing wallet seed
bedrock faucet --wallet sEd7...

# Fund on local network
bedrock faucet --network local
```

| Network | Faucet URL |
|---------|------------|
| Local | `http://localhost:8080/faucet` |
| Alphanet | `https://alphanet.faucet.nerdnest.xyz/accounts` |

## Storage Details

Wallets are stored as encrypted JSON files:

```
~/.config/bedrock/wallets/
├── my-dev-wallet.json
├── my-ed-wallet.json
└── my-existing-wallet.json
```

Each file contains:
- Encrypted seed (AES-256-GCM)
- Public address
- Algorithm type
- Key derivation parameters (PBKDF2 salt, iterations)

The seed is **never** stored in plaintext.
