# Bedrock - XRPL Smart Contract CLI

**The foundation for XRPL smart contract development**

Bedrock is a developer tool for building, deploying, and interacting with XRPL smart contracts written in Rust. Think Foundry, but for XRPL.

## Features

- 🏗️ **Build** - Compile Rust smart contracts to WASM
- 🚀 **Deploy** - Smart deployment with auto-build and ABI generation
- 📞 **Call** - Interact with deployed contracts
- 🔧 **Local Node** - Manage local XRPL test network
- 📝 **ABI Generation** - Automatic ABI extraction from Rust code
- ⚡ **Fast** - Embedded JS modules with intelligent caching

## Requirements

Before installing Bedrock, ensure you have:

- **[Go](https://go.dev/dl/)** (1.24 or later) - For building/installing Bedrock
- **[Node.js](https://nodejs.org/)** (18 or later) - For XRPL transaction handling
- **[Rust](https://rustup.rs/)** - For compiling smart contracts
  ```bash
  curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
  ```
- **[Docker](https://www.docker.com/)** (optional) - For running local XRPL node

### Verify Installation

```bash
go version      # Should show 1.24+
node --version  # Should show v18+
rustc --version # Should show 1.70+
cargo --version
```

## Installation

### Download Binary (Recommended)

```bash
curl -sSfL https://raw.githubusercontent.com/XRPL-Commons/Bedrock/main/install.sh | sh
```

This auto-detects your OS and architecture, downloads the latest release binary, and installs it to `/usr/local/bin/`.

### Install from Source

```bash
# Clone the repository
git clone https://github.com/XRPL-Commons/Bedrock.git
cd Bedrock

# Build and install
go build -o bedrock cmd/bedrock/main.go
sudo mv bedrock /usr/local/bin/

# Verify installation
bedrock --help
```

### First Run Setup

On first run, Bedrock will automatically:
1. Install JavaScript dependencies (XRPL libraries)
2. Cache them in `~/.cache/bedrock/modules` (Linux/macOS)
3. You'll see: `⚡ First run detected - installing JavaScript dependencies...`

This only happens once. Subsequent runs are instant!

## Quick Start

### 1. Initialize a New Project

```bash
bedrock init my-contract
cd my-contract
```

This creates:
```
my-contract/
├── bedrock.toml          # Project configuration
├── contract/
│   └── src/
│       └── lib.rs        # Your smart contract
└── abi.json              # Generated ABI (after build)
```

### 2. Build Your Contract

```bash
bedrock build
```

This compiles your Rust contract to WASM in release mode.

### 3. Start Local Node (Optional)

```bash
bedrock node start
```

Starts a local XRPL test network in Docker at:
- WebSocket: `ws://localhost:6006`
- Faucet: `http://localhost:8080/faucet`

### 4. Deploy Your Contract

```bash
# Deploy to local node
bedrock deploy --network local

# Deploy to alphanet (testnet)
bedrock deploy --network alphanet

# Deploy with specific wallet
bedrock deploy --wallet sXXX...
```

**Smart Deployment** automatically:
1. ✅ Builds contract in release mode (if needed)
2. ✅ Generates ABI (if needed)
3. ✅ Deploys to network

### 5. Call Contract Functions

```bash
# Call without parameters
bedrock call rContract123... hello --wallet sXXX...

# Call with parameters
bedrock call rContract123... transfer \
  --wallet sXXX... \
  --params '{"to":"rRecipient...","amount":1000}'

# Use parameters from file
bedrock call rContract123... register \
  --wallet sXXX... \
  --params-file params.json
```

## Commands

### Core Workflow

| Command | Description |
|---------|-------------|
| `bedrock init [name]` | Initialize new project |
| `bedrock build` | Build contract (release mode) |
| `bedrock deploy` | Deploy with auto-build & ABI |
| `bedrock call <contract> <fn>` | Call contract function |
| `bedrock node <start\|stop\|status>` | Manage local node |

### Build Options

```bash
bedrock build              # Build in release mode (default)
bedrock build --debug      # Build in debug mode (faster, larger)
```

### Deploy Options

```bash
bedrock deploy                      # Smart deploy (auto-build + ABI)
bedrock deploy --skip-build         # Skip building
bedrock deploy --skip-abi           # Skip ABI generation
bedrock deploy --network alphanet   # Deploy to alphanet
bedrock deploy --wallet sXXX...     # Use specific wallet
```

### Call Options

```bash
bedrock call <contract> <function> \
  --wallet sXXX...                  # Wallet seed or jade keystore name (required)
  --network local                   # Network (default: local; use alphanet for testnet)
  --params '{"key":"value"}'        # JSON parameters
  --params-file params.json         # Parameters from file
  --gas 1000000                     # Computation allowance
  --fee 1000000                     # Transaction fee (drops)
```

### Node Management

```bash
bedrock node start             # Start local XRPL node
bedrock node stop              # Stop local node
bedrock node status            # Check if running
bedrock node logs              # Print recent node logs
bedrock node logs --follow     # Stream logs (like `docker logs -f`)
bedrock node logs --tail 100   # Show only the last N lines
```

## Project Configuration

Edit `bedrock.toml` to configure your project:

```toml
[project]
name = "my-contract"
version = "0.1.0"

[build]
source = "contract/src/lib.rs"
target = "wasm32-unknown-unknown"

[networks.local]
url = "ws://localhost:6006"
faucet_url = "http://localhost:8080/faucet"

[networks.alphanet]
url = "wss://alphanet.nerdnest.xyz"
faucet_url = "https://alphanet.faucet.nerdnest.xyz/accounts"
```

## Writing Smart Contracts

### Basic Contract Structure

```rust
use xrpl_wasm::*;

/// @xrpl-function hello
/// @xrpl-param name: VL - User name
/// @xrpl-return UINT64 - Success code
#[wasm_export]
fn hello(name: Vec<u8>) -> u64 {
    let _ = trace("Hello from XRPL!");
    0 // Success
}
```

### ABI Annotations

Bedrock automatically generates ABIs from JSDoc-style comments:

```rust
/// @xrpl-function transfer
/// @xrpl-param to: ACCOUNT - Recipient address
/// @xrpl-param amount: UINT64 - Amount to transfer
/// @xrpl-return UINT64 - Success code
#[wasm_export]
fn transfer(to: Vec<u8>, amount: u64) -> u64 {
    // Implementation
    0
}
```

Supported parameter types:
- `UINT8`, `UINT16`, `UINT32`, `UINT64`, `UINT128`, `UINT256`
- `VL` (variable length bytes/string)
- `ACCOUNT` (XRPL address)
- `AMOUNT` (XRP drops or IOU)
- `CURRENCY`, `ISSUE`

## Development Workflow

### Local Development Loop

```bash
# Terminal 1: Start local node
bedrock node start

# Terminal 2: Develop and test
bedrock build              # Build contract
bedrock deploy --local     # Deploy to local node
bedrock call rXXX... hello --wallet sXXX...  # Test function

# Make changes to contract...
bedrock deploy --local     # Redeploy (auto-rebuilds)
```

### Deploying to Testnet

```bash
# 1. Build and deploy to alphanet
bedrock deploy --network alphanet

# Save the output:
#   Wallet Seed: sXXX...
#   Contract Account: rContract123...

# 2. Call your contract
bedrock call rContract123... myFunction \
  --wallet sXXX... \
  --network alphanet
```

## Architecture

### Module System

Bedrock uses embedded JavaScript modules for XRPL transaction handling:

```
~/.cache/bedrock/modules/
├── deploy.js           # Deployment module
├── call.js            # Contract calling module
├── package.json       # Dependencies (xrpl)
└── node_modules/      # Installed on first run
```

### Why JavaScript Modules?

XRPL smart contracts are in alpha, and the only stable tooling is in JavaScript. Bedrock embeds these modules and will migrate to pure Go as XRPL tooling matures.

### Cache Management

- **First run**: Installs npm dependencies (~10-15 seconds)
- **Subsequent runs**: Uses cached modules (instant)
- **Updates**: Automatic reinstall when CLI is updated (version detection via SHA256)
- **Manual cleanup**: `rm -rf ~/.cache/bedrock`

## Troubleshooting

### Dependencies Not Installing

```bash
# Check Node.js version
node --version  # Should be 18+

# Manually install dependencies
cd ~/.cache/bedrock/modules
npm install
```

### WASM Build Fails

```bash
# Add wasm32 target
rustup target add wasm32-unknown-unknown

# Update Rust
rustup update
```

### Local Node Won't Start

```bash
# Check Docker is running
docker ps

# View Docker logs
docker logs bedrock-xrpl-node

# Restart Docker daemon
```

### Config Not Found

```bash
# Make sure you're in a Bedrock project directory
ls bedrock.toml

# Or initialize a new project
bedrock init my-project
```

## Contributing

Contributions are welcome! Please check out:
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines
- [BRANDING.md](BRANDING.md) - Design philosophy
- [Issues](https://github.com/XRPL-Commons/Bedrock/issues) - Bug reports & features

## Roadmap

- [x] Smart contract compilation (Rust → WASM)
- [x] Automatic ABI generation
- [x] Contract deployment
- [x] Contract function calling
- [x] Local node management
- [ ] Contract testing framework
- [ ] Wallet management (`bedrock wallet`)
- [ ] Contract verification
- [ ] TypeScript SDK generation
- [ ] Watch mode for development
- [ ] Migration to pure Go (when XRPL tooling matures)

## Documentation

Bedrock includes comprehensive documentation for developers and AI assistants:

### Core Documentation
| Document | Description |
|----------|-------------|
| [Quick Reference](docs/QUICK_REFERENCE.md) | Cheat sheet for common operations |
| [Commands Reference](docs/COMMANDS_REFERENCE.md) | Complete CLI command documentation |
| [Building Contracts](docs/building-contracts.md) | Contract compilation guide |
| [Deployment & Calling](docs/deployment-and-calling.md) | Deploy and interact with contracts |
| [ABI Generation](docs/abi-generation.md) | ABI annotations and types |
| [Local Node](docs/local-node.md) | Local development environment |
| [Wallet Management](docs/wallet.md) | Secure wallet storage |

### For AI Assistants & MCPs
| Document | Description |
|----------|-------------|
| [llms.txt](llms.txt) | Structured overview for LLMs |
| [CLAUDE.md](CLAUDE.md) | Detailed instructions for AI agents |

## License

MIT License - See [LICENSE](LICENSE) for details

## Resources

- **XRPL Docs**: https://xrpl.org/
- **Smart Contracts**: https://xrpl.org/smart-contracts.html
- **Alphanet**: https://alphanet.nerdnest.xyz
- **Community**: [XRPL Discord](https://discord.gg/xrpl)

---

Built with ⚡ by the Bedrock team
