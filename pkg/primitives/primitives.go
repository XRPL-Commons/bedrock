package primitives

const (
	Contract = "contract"
	Escrow   = "escrow"
	Vault    = "vault"
)

// PrimitiveDef describes the capabilities and configuration of an XRPL primitive type.
type PrimitiveDef struct {
	Kind        string
	DisplayName string
	Description string
	NeedsBuild  bool
	NeedsABI    bool
	WasmTarget  string
	RustEdition string
	SourceDir   string
	DockerImage string
}

// Registry maps primitive kind to its definition.
var Registry = map[string]PrimitiveDef{
	Contract: {
		Kind:        Contract,
		DisplayName: "Smart Contract",
		Description: "Custom Rust WASM logic (ContractCreate/ContractCall)",
		NeedsBuild:  true,
		NeedsABI:    true,
		WasmTarget:  "wasm32-unknown-unknown",
		RustEdition: "2021",
		SourceDir:   "contract",
		DockerImage: "transia/cluster:f5d78179c9d1fbaf8bff8b77a052e263df90faa1",
	},
	Escrow: {
		Kind:        Escrow,
		DisplayName: "Smart Escrow",
		Description: "Conditional payments with WASM conditions (EscrowCreate/EscrowFinish)",
		NeedsBuild:  true,
		NeedsABI:    false,
		WasmTarget:  "wasm32v1-none",
		RustEdition: "2024",
		SourceDir:   "escrow",
		DockerImage: "willemolding/rippled:smart-vaults.0",
	},
	Vault: {
		Kind:        Vault,
		DisplayName: "Smart Vault",
		Description: "Asset custody with WASM deposit/withdraw logic (VaultCreate/VaultDeposit/VaultWithdraw)",
		NeedsBuild:  true,
		NeedsABI:    false,
		WasmTarget:  "wasm32v1-none",
		RustEdition: "2024",
		SourceDir:   "vault",
		DockerImage: "willemolding/rippled:smart-vaults.0",
	},
}

// All returns all primitive kinds in display order.
func All() []string {
	return []string{Contract, Escrow, Vault}
}

// IsValid returns true if the given kind is a known primitive.
func IsValid(kind string) bool {
	_, ok := Registry[kind]
	return ok
}
