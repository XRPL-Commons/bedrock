# Building Contracts

Bedrock compiles Rust code to WebAssembly (WASM) for deployment on XRPL. It wraps `cargo` with sensible defaults and a great developer experience.

## Overview

Bedrock's build process provides:

- Smart defaults for WASM compilation
- Automatic toolchain validation
- Auto-detection of WASM target per primitive
- Progress indicators and formatted output
- Build optimization options
- Artifact management

## Build Command

```bash
bedrock build [flags]
```

**Flags:**

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--release` | `-r` | Build in release mode (optimized) | `true` |
| `--type` | | Build specific primitive (contract, escrow, vault) | all |

By default, `bedrock build` builds all primitives configured in the project. Use `--type` to build only one:

```bash
bedrock build                    # Build all primitives
bedrock build --type contract    # Build only contract
bedrock build --type escrow      # Build only escrow
bedrock build --type vault       # Build only vault
```

**What it does:**

1. Validates Rust toolchain (cargo, rustc)
2. Ensures the correct WASM target is installed for each primitive
3. Reads build configuration from `bedrock.toml`
4. Compiles Rust to WASM using cargo
5. Reports build results (path, size, duration)

### Build Targets per Primitive

| Primitive | WASM Target | Rust Edition |
|-----------|-------------|--------------|
| Contract | `wasm32-unknown-unknown` | 2021 |
| Escrow | `wasm32v1-none` | 2024 |
| Vault | `wasm32v1-none` | 2024 |

## Build Modes

### Release Build (Default)

```bash
bedrock build
```

- Heavily optimized, small WASM file (50-200 KB)
- Slower compilation (~5-15 seconds)
- Production-ready
- Use for deployment to testnet/mainnet

### Debug Build

```bash
bedrock build --release=false
```

- Fast compilation (~2-5 seconds)
- Includes debug symbols, better error messages
- Large WASM file (1-2 MB)
- Use for rapid development iteration

**Size comparison:**

| Configuration | WASM Size | Build Time |
|---------------|-----------|------------|
| Debug default | 1.2 MB | 2s |
| Release `opt-level = "3"` | 450 KB | 5s |
| Release `opt-level = "z"` | 180 KB | 6s |
| Release `opt-level = "z"` + LTO | 156 KB | 8s |

## Contract Structure

Your contract should be structured as:

```
my-contract/
├── bedrock.toml           # Project config
├── contract/
│   ├── Cargo.toml         # Rust package manifest
│   ├── src/
│   │   └── lib.rs         # Contract code
│   └── target/            # Build output (gitignored)
│       └── wasm32-unknown-unknown/
│           ├── debug/
│           │   └── *.wasm
│           └── release/
│               └── *.wasm
```

### Cargo.toml

Your `contract/Cargo.toml` should specify:

```toml
[package]
name = "my-contract"
version = "0.1.0"
edition = "2021"

[lib]
crate-type = ["cdylib"]    # Required for WASM

[dependencies]
# Your XRPL dependencies

[profile.release]
opt-level = "z"            # Optimize for size
lto = true                 # Link-time optimization
strip = true               # Strip debug symbols
panic = "abort"            # Smaller panic handler
```

**Key settings for WASM:**

- `crate-type = ["cdylib"]` - Creates dynamic library for WASM
- `opt-level = "z"` - Maximize size optimization
- `lto = true` - Enable link-time optimization
- `strip = true` - Remove debug symbols
- `panic = "abort"` - Use simpler panic handler

## Configuration

Build settings are read from `bedrock.toml`:

```toml
[build]
source = "contract/src/lib.rs"
target = "wasm32-unknown-unknown"
```

| Option | Description | Default |
|--------|-------------|---------|
| `source` | Path to contract source file | `contract/src/lib.rs` |
| `target` | Rust compilation target | `wasm32-unknown-unknown` |

## Optimization Tips

### 1. Minimize Dependencies

```toml
# Avoid large dependency trees
# Only include XRPL-specific dependencies
[dependencies]
```

### 2. Use `opt-level = "z"`

```toml
[profile.release]
opt-level = "z"  # Smaller than "s" or "3"
```

### 3. Enable LTO

```toml
[profile.release]
lto = true  # Link-time optimization
```

### 4. Strip Symbols

```toml
[profile.release]
strip = true  # Remove debug info
```

### 5. Avoid Panics

```rust
// Bad: Uses panic machinery
assert!(condition);

// Good: Handle errors explicitly
if !condition {
    return Err(Error::InvalidInput);
}
```

## Toolchain Requirements

### Installing the WASM Targets

Bedrock automatically installs the WASM target if missing:

```bash
# For contracts
rustup target add wasm32-unknown-unknown

# For escrows and vaults
rustup target add wasm32v1-none
```

### Verifying Toolchain

```bash
rustc --version                                    # 1.70.0 or later
cargo --version                                    # 1.70.0 or later
rustup target list | grep wasm32-unknown-unknown   # (installed)
rustup target list | grep wasm32v1-none            # (installed, for escrow/vault)
```

## Under the Hood

`bedrock build` essentially runs:

```bash
# For release (default)
cargo build --target wasm32-unknown-unknown --release

# For debug
cargo build --target wasm32-unknown-unknown
```

With additional steps: toolchain validation, progress formatting, size reporting, and artifact management.

## Troubleshooting

### "cargo not found"

Install the Rust toolchain:

```bash
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source $HOME/.cargo/env
```

### "error: couldn't find library for wasm32-unknown-unknown"

Install the WASM target:

```bash
rustup target add wasm32-unknown-unknown
```

### "Cargo.toml not found"

Ensure you're in a bedrock project directory with `contract/Cargo.toml`.

### Build is Very Slow

- Use debug builds during development: `bedrock build --release=false`
- Reduce dependencies
- Consider using `sccache` for caching
