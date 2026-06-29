package cli

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xrpl-commons/bedrock/pkg/config"
	"github.com/xrpl-commons/bedrock/pkg/jade"
	"github.com/xrpl-commons/bedrock/pkg/wallet"
)

var jadeAlgorithm string

var jadeCmd = &cobra.Command{
	Use:   "jade",
	Short: "Wallet management and XRPL utilities",
	Long: `Jade - Wallet management and XRPL utilities for smart contracts

Create, import, and manage XRPL wallets securely with encryption.
Query balances, send XRP, and inspect transactions.

Wallet Commands:
  new       Create a new XRPL wallet
  import    Import an existing wallet from seed
  list      List all stored wallets
  export    Export wallet seed and address
  remove    Remove a wallet from storage

Network Commands:
  balance   Get XRP balance for an address
  send      Send XRP to a destination
  tx        Get transaction details by hash
  account   Get detailed account information
  server    Get XRPL server information

Utility Commands:
  encode    Encode text to hexadecimal
  decode    Decode hexadecimal to text`,
}

var jadeNewCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Create a new XRPL wallet",
	Long: `Create a new XRPL wallet with a randomly generated seed.

The wallet will be encrypted and stored securely on disk.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		walletName := args[0]

		// Initialize components
		walletManager, err := wallet.NewWalletManager()
		if err != nil {
			fmt.Printf("Error: Failed to initialize wallet manager: %v\n", err)
			os.Exit(1)
		}

		xrplWallet, err := wallet.NewXRPLWallet()
		if err != nil {
			fmt.Printf("Error: Failed to initialize XRPL wallet: %v\n", err)
			os.Exit(1)
		}
		authProvider := wallet.NewAuthProvider()

		// Generate new wallet
		newWallet, err := xrplWallet.GenerateWalletWithAlgorithm(walletName, jadeAlgorithm)
		if err != nil {
			fmt.Printf("Error: Failed to generate wallet: %v\n", err)
			os.Exit(1)
		}

		// Get password for encryption
		password, err := authProvider.GetPassword("Enter password to encrypt wallet: ")
		if err != nil {
			fmt.Printf("Error: Failed to read password: %v\n", err)
			os.Exit(1)
		}

		if len(password) == 0 {
			fmt.Println("Error: Password cannot be empty")
			os.Exit(1)
		}

		// Save encrypted wallet
		if err := walletManager.SaveWallet(newWallet, password); err != nil {
			fmt.Printf("Error: Failed to save wallet: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Wallet '%s' created successfully\n", walletName)
		fmt.Printf("   Address: %s\n", newWallet.Address)
		fmt.Printf("   Stored at: ~/.config/bedrock/wallets/%s.json\n", walletName)
	},
}

var jadeImportCmd = &cobra.Command{
	Use:   "import <name>",
	Short: "Import an existing XRPL wallet from seed",
	Long: `Import an existing XRPL wallet using its seed.

The seed input will be hidden for security.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		walletName := args[0]

		// Initialize components
		walletManager, err := wallet.NewWalletManager()
		if err != nil {
			fmt.Printf("Error: Failed to initialize wallet manager: %v\n", err)
			os.Exit(1)
		}

		xrplWallet, err := wallet.NewXRPLWallet()
		if err != nil {
			fmt.Printf("Error: Failed to initialize XRPL wallet: %v\n", err)
			os.Exit(1)
		}
		authProvider := wallet.NewAuthProvider()

		// Get seed (hidden input)
		seed, err := authProvider.GetPassword("Enter seed: ")
		if err != nil {
			fmt.Printf("Error: Failed to read seed: %v\n", err)
			os.Exit(1)
		}

		if len(seed) == 0 {
			fmt.Println("Error: Seed cannot be empty")
			os.Exit(1)
		}

		// Import wallet
		importedWallet, err := xrplWallet.ImportWalletWithAlgorithm(walletName, seed, jadeAlgorithm)
		if err != nil {
			fmt.Printf("Error: Failed to import wallet: %v\n", err)
			os.Exit(1)
		}

		// Get password for encryption
		password, err := authProvider.GetPassword("Enter password to encrypt wallet: ")
		if err != nil {
			fmt.Printf("Error: Failed to read password: %v\n", err)
			os.Exit(1)
		}

		if len(password) == 0 {
			fmt.Println("Error: Password cannot be empty")
			os.Exit(1)
		}

		// Save encrypted wallet
		if err := walletManager.SaveWallet(importedWallet, password); err != nil {
			fmt.Printf("Error: Failed to save wallet: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Wallet '%s' imported successfully\n", walletName)
		fmt.Printf("   Address: %s\n", importedWallet.Address)
		fmt.Printf("   Stored at: ~/.config/bedrock/wallets/%s.json\n", walletName)
	},
}

var jadeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all stored wallets",
	Long:  `List all wallets stored in the keystore with their addresses.`,
	Run: func(cmd *cobra.Command, args []string) {
		walletManager, err := wallet.NewWalletManager()
		if err != nil {
			fmt.Printf("Error: Failed to initialize wallet manager: %v\n", err)
			os.Exit(1)
		}

		wallets, err := walletManager.ListWallets()
		if err != nil {
			fmt.Printf("Error: Failed to list wallets: %v\n", err)
			os.Exit(1)
		}

		if len(wallets) == 0 {
			fmt.Println("No wallets found")
			return
		}

		fmt.Printf("Found %d wallet(s):\n\n", len(wallets))
		for _, w := range wallets {
			fmt.Printf("Name:    %s\n", w.Name)
			fmt.Printf("Address: %s\n", w.Address)
			fmt.Printf("Created: %s\n", w.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Println()
		}
	},
}

var jadeExportCmd = &cobra.Command{
	Use:   "export <name>",
	Short: "Export wallet seed and address",
	Long: `Export the seed and address of a stored wallet.

Requires password authentication.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		walletName := args[0]

		// Initialize components
		walletManager, err := wallet.NewWalletManager()
		if err != nil {
			fmt.Printf("Error: Failed to initialize wallet manager: %v\n", err)
			os.Exit(1)
		}

		authProvider := wallet.NewAuthProvider()

		// Check if wallet exists
		if !walletManager.WalletExists(walletName) {
			fmt.Printf("Error: Wallet '%s' not found\n", walletName)
			os.Exit(1)
		}

		// Get password for wallet decryption
		password, err := authProvider.GetPassword("Enter password: ")
		if err != nil {
			fmt.Printf("Error: Failed to read password: %v\n", err)
			os.Exit(1)
		}

		// Load wallet
		loadedWallet, err := walletManager.LoadWallet(walletName, password)
		if err != nil {
			fmt.Printf("Error: Failed to load wallet: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Wallet: %s\n", loadedWallet.Name)
		fmt.Printf("Address: %s\n", loadedWallet.Address)
		fmt.Printf("Seed: %s\n", loadedWallet.Seed)
	},
}

var jadeRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a wallet from storage",
	Long: `Permanently remove a wallet from storage.

This action cannot be undone. Make sure you have backed up the seed if needed.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		walletName := args[0]

		walletManager, err := wallet.NewWalletManager()
		if err != nil {
			fmt.Printf("Error: Failed to initialize wallet manager: %v\n", err)
			os.Exit(1)
		}

		// Check if wallet exists
		if !walletManager.WalletExists(walletName) {
			fmt.Printf("Error: Wallet '%s' not found\n", walletName)
			os.Exit(1)
		}

		// Confirm removal
		fmt.Printf("Are you sure you want to remove wallet '%s'? (y/N): ", walletName)
		var response string
		_, _ = fmt.Scanln(&response)

		if response != "y" && response != "Y" {
			fmt.Println("Operation cancelled")
			return
		}

		// Remove wallet
		if err := walletManager.RemoveWallet(walletName); err != nil {
			fmt.Printf("Error: Failed to remove wallet: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Wallet '%s' removed successfully\n", walletName)
	},
}

// Network operation commands

var jadeBalanceCmd = &cobra.Command{
	Use:   "balance <address>",
	Short: "Get XRP balance for an address",
	Long: `Get the XRP balance for an XRPL address.

Examples:
  bedrock jade balance rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D8
  bedrock jade balance rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D8 --network local`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		address := args[0]

		if err := validateXRPLAddress(address); err != nil {
			return err
		}

		network, _ := cmd.Flags().GetString("network")
		networkURL, err := getNetworkConfig(network)
		if err != nil {
			return err
		}

		verbose, _ := cmd.Flags().GetBool("verbose")
		ops, err := jade.NewOperations(verbose)
		if err != nil {
			return err
		}

		result, err := ops.GetBalance(networkURL, address)
		if err != nil {
			return err
		}

		fmt.Printf("Address: %s\n", result.Address)
		fmt.Printf("Balance: %s XRP\n", result.Balance)
		fmt.Printf("Balance: %s drops\n", result.BalanceDrops)
		if result.Balance == "0" {
			fmt.Println("Status:  Not funded")
		}
		return nil
	},
}

var jadeSendCmd = &cobra.Command{
	Use:   "send <destination> <amount>",
	Short: "Send XRP to a destination address",
	Long: `Send XRP to a destination address.

Amount is specified in XRP (not drops).

Examples:
  bedrock jade send rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D8 10 --wallet myWallet
  bedrock jade send rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D8 100.5 --wallet myWallet --network local`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		destination := args[0]
		amount := args[1]

		if err := validateXRPLAddress(destination); err != nil {
			return fmt.Errorf("invalid destination: %w", err)
		}

		walletFlag, _ := cmd.Flags().GetString("wallet")
		algorithm, _ := cmd.Flags().GetString("algorithm")
		network, _ := cmd.Flags().GetString("network")

		// Resolve wallet using the standard WalletResolver
		resolver, err := wallet.NewWalletResolver()
		if err != nil {
			return fmt.Errorf("failed to initialize wallet resolver: %w", err)
		}

		walletSeed, err := resolver.ResolveWallet(walletFlag)
		if err != nil {
			return fmt.Errorf("failed to resolve wallet: %w", err)
		}

		networkURL, err := getNetworkConfig(network)
		if err != nil {
			return err
		}

		verbose, _ := cmd.Flags().GetBool("verbose")
		ops, err := jade.NewOperations(verbose)
		if err != nil {
			return err
		}

		fmt.Printf("Sending %s XRP to %s...\n", amount, destination)

		result, err := ops.Send(networkURL, walletSeed, destination, amount, algorithm)
		if err != nil {
			return err
		}

		if result.Result == "tesSUCCESS" {
			fmt.Println("Transaction successful!")
		} else {
			fmt.Printf("Transaction result: %s\n", result.Result)
		}
		fmt.Printf("TX Hash:   %s\n", result.TxHash)
		fmt.Printf("From:      %s\n", result.From)
		fmt.Printf("To:        %s\n", result.To)
		fmt.Printf("Amount:    %s XRP\n", result.Amount)
		fmt.Printf("Fee:       %s drops\n", result.Fee)
		fmt.Printf("Validated: %v\n", result.Validated)
		return nil
	},
}

var jadeTxCmd = &cobra.Command{
	Use:   "tx <hash>",
	Short: "Get transaction details by hash",
	Long: `Get detailed information about a transaction by its hash.

Examples:
  bedrock jade tx 5B5C8D3E2A1F4B6C7D8E9F0A1B2C3D4E5F6A7B8C9D0E1F2A3B4C5D6E7F8A9B0C
  bedrock jade tx 5B5C8D... --network alphanet`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hash := args[0]

		if err := validateTxHash(hash); err != nil {
			return err
		}

		network, _ := cmd.Flags().GetString("network")
		networkURL, err := getNetworkConfig(network)
		if err != nil {
			return err
		}

		verbose, _ := cmd.Flags().GetBool("verbose")
		ops, err := jade.NewOperations(verbose)
		if err != nil {
			return err
		}

		result, err := ops.GetTransaction(networkURL, hash)
		if err != nil {
			return err
		}

		fmt.Printf("Hash:        %s\n", result.Hash)
		fmt.Printf("Type:        %s\n", result.Type)
		fmt.Printf("Account:     %s\n", result.Account)
		fmt.Printf("Result:      %s\n", result.Result)
		fmt.Printf("Fee:         %s drops\n", result.Fee)
		fmt.Printf("Sequence:    %d\n", result.Sequence)
		fmt.Printf("Ledger:      %d\n", result.LedgerIndex)
		fmt.Printf("Validated:   %v\n", result.Validated)

		// Type-specific fields
		if result.Destination != "" {
			fmt.Printf("Destination: %s\n", result.Destination)
		}
		if result.Amount != nil {
			fmt.Printf("Amount:      %v\n", result.Amount)
		}
		if result.ContractAccount != "" {
			fmt.Printf("Contract:    %s\n", result.ContractAccount)
		}
		if result.FunctionName != "" {
			fmt.Printf("Function:    %s\n", result.FunctionName)
		}
		return nil
	},
}

var jadeAccountCmd = &cobra.Command{
	Use:   "account <address>",
	Short: "Get detailed account information",
	Long: `Get detailed information about an XRPL account.

Examples:
  bedrock jade account rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D8
  bedrock jade account rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D8 --network local`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		address := args[0]

		if err := validateXRPLAddress(address); err != nil {
			return err
		}

		network, _ := cmd.Flags().GetString("network")
		networkURL, err := getNetworkConfig(network)
		if err != nil {
			return err
		}

		verbose, _ := cmd.Flags().GetBool("verbose")
		ops, err := jade.NewOperations(verbose)
		if err != nil {
			return err
		}

		result, err := ops.GetAccountInfo(networkURL, address)
		if err != nil {
			return err
		}

		if result.Error != "" {
			fmt.Printf("Address: %s\n", result.Address)
			fmt.Printf("Status:  %s\n", result.Error)
			return nil
		}

		fmt.Printf("Address:       %s\n", result.Address)
		fmt.Printf("Balance:       %s XRP\n", result.GetBalanceString())
		fmt.Printf("Balance:       %s drops\n", result.BalanceDrops)
		fmt.Printf("Sequence:      %d\n", result.Sequence)
		fmt.Printf("Owner Count:   %d\n", result.OwnerCount)
		fmt.Printf("Flags:         %d\n", result.Flags)
		fmt.Printf("Ledger Index:  %d\n", result.LedgerIndex)
		if result.PreviousTxnID != "" {
			fmt.Printf("Previous TX:   %s\n", result.PreviousTxnID)
		}
		return nil
	},
}

var jadeServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Get XRPL server information",
	Long: `Get information about the connected XRPL server.

Examples:
  bedrock jade server
  bedrock jade server --network local`,
	RunE: func(cmd *cobra.Command, args []string) error {
		network, _ := cmd.Flags().GetString("network")
		networkURL, err := getNetworkConfig(network)
		if err != nil {
			return err
		}

		verbose, _ := cmd.Flags().GetBool("verbose")
		ops, err := jade.NewOperations(verbose)
		if err != nil {
			return err
		}

		result, err := ops.GetServerInfo(networkURL)
		if err != nil {
			return err
		}

		fmt.Printf("Build Version:     %s\n", result.BuildVersion)
		fmt.Printf("Server State:      %s\n", result.ServerState)
		fmt.Printf("Network ID:        %d\n", result.NetworkID)
		fmt.Printf("Complete Ledgers:  %s\n", result.CompleteLedgers)
		fmt.Printf("Peers:             %d\n", result.Peers)
		fmt.Printf("Uptime:            %d seconds\n", result.Uptime)
		if result.ValidatedLedger != nil {
			if seq, ok := result.ValidatedLedger["seq"].(float64); ok {
				fmt.Printf("Validated Ledger:  %.0f\n", seq)
			}
		}
		return nil
	},
}

// Encoding/Decoding commands

var jadeEncodeCmd = &cobra.Command{
	Use:   "encode <text>",
	Short: "Encode text to hexadecimal",
	Long: `Encode a string to its hexadecimal representation.

Useful for encoding function names and parameters for smart contract calls.

Examples:
  bedrock jade encode "hello"
  bedrock jade encode "transfer"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		text := args[0]
		encoded := hex.EncodeToString([]byte(text))
		fmt.Printf("Text:    %s\n", text)
		fmt.Printf("Hex:     %s\n", encoded)
		fmt.Printf("0x Hex:  0x%s\n", encoded)
		return nil
	},
}

var jadeDecodeCmd = &cobra.Command{
	Use:   "decode <hex>",
	Short: "Decode hexadecimal to text",
	Long: `Decode a hexadecimal string to its text representation.

Useful for decoding function names and return values from smart contracts.

Examples:
  bedrock jade decode 68656c6c6f
  bedrock jade decode 0x68656c6c6f`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hexStr := args[0]

		// Remove 0x prefix if present
		hexStr = strings.TrimPrefix(hexStr, "0x")
		hexStr = strings.TrimPrefix(hexStr, "0X")

		decoded, err := hex.DecodeString(hexStr)
		if err != nil {
			return fmt.Errorf("invalid hex string: %w", err)
		}

		// Check if it's printable text
		isPrintable := true
		for _, b := range decoded {
			if b < 32 || b > 126 {
				isPrintable = false
				break
			}
		}

		fmt.Printf("Hex:     %s\n", hexStr)
		if isPrintable {
			fmt.Printf("Text:    %s\n", string(decoded))
		} else {
			fmt.Printf("Bytes:   %v\n", decoded)
			fmt.Printf("Text:    %s (may contain non-printable characters)\n", string(decoded))
		}
		return nil
	},
}

// getNetworkConfig resolves the network name to a URL.
// It first checks bedrock.toml, then falls back to built-in defaults.
func getNetworkConfig(network string) (string, error) {
	networks := map[string]string{
		"local":    "ws://localhost:6006",
		"alphanet": "wss://alphanet.nerdnest.xyz",
		"testnet":  "wss://s.altnet.rippletest.net:51233",
		"mainnet":  "wss://xrplcluster.com",
	}

	// Try to load from bedrock.toml first
	cfg, err := config.LoadFromWorkingDir()
	if err == nil {
		if netCfg, ok := cfg.Networks[network]; ok {
			return netCfg.URL, nil
		}
	}

	// Fall back to built-in defaults
	if url, ok := networks[network]; ok {
		return url, nil
	}

	return "", fmt.Errorf("unknown network %q, available: local, alphanet, testnet, mainnet", network)
}

// validateXRPLAddress performs basic validation on an XRPL address.
func validateXRPLAddress(address string) error {
	if !strings.HasPrefix(address, "r") {
		return fmt.Errorf("invalid XRPL address %q: must start with 'r'", address)
	}
	if len(address) < 25 || len(address) > 35 {
		return fmt.Errorf("invalid XRPL address %q: must be 25-35 characters", address)
	}
	return nil
}

// validateTxHash performs basic validation on a transaction hash.
func validateTxHash(hash string) error {
	if len(hash) != 64 {
		return fmt.Errorf("invalid transaction hash: must be 64 hex characters, got %d", len(hash))
	}
	if _, err := hex.DecodeString(hash); err != nil {
		return fmt.Errorf("invalid transaction hash: must be hexadecimal")
	}
	return nil
}

func init() {
	// Add wallet subcommands to jade
	jadeCmd.AddCommand(jadeNewCmd)
	jadeCmd.AddCommand(jadeImportCmd)
	jadeCmd.AddCommand(jadeListCmd)
	jadeCmd.AddCommand(jadeExportCmd)
	jadeCmd.AddCommand(jadeRemoveCmd)

	// Add network operation subcommands to jade
	jadeCmd.AddCommand(jadeBalanceCmd)
	jadeCmd.AddCommand(jadeSendCmd)
	jadeCmd.AddCommand(jadeTxCmd)
	jadeCmd.AddCommand(jadeAccountCmd)
	jadeCmd.AddCommand(jadeServerCmd)

	// Add encoding/decoding subcommands to jade
	jadeCmd.AddCommand(jadeEncodeCmd)
	jadeCmd.AddCommand(jadeDecodeCmd)

	// Add flags to wallet commands
	jadeNewCmd.Flags().StringVarP(&jadeAlgorithm, "algorithm", "a", "secp256k1", "Cryptographic algorithm (secp256k1, ed25519)")
	jadeImportCmd.Flags().StringVarP(&jadeAlgorithm, "algorithm", "a", "secp256k1", "Cryptographic algorithm (secp256k1, ed25519)")

	// Add common flags to network operation commands
	networkCmds := []*cobra.Command{jadeBalanceCmd, jadeSendCmd, jadeTxCmd, jadeAccountCmd, jadeServerCmd}
	for _, cmd := range networkCmds {
		cmd.Flags().StringP("network", "n", "alphanet", "Network to use (local, alphanet, testnet, mainnet)")
	}

	// Add wallet and algorithm flags to send command
	jadeSendCmd.Flags().StringP("wallet", "w", "", "Wallet name (required)")
	_ = jadeSendCmd.MarkFlagRequired("wallet")
	jadeSendCmd.Flags().StringP("algorithm", "a", "secp256k1", "Cryptographic algorithm (secp256k1, ed25519)")

	// Add jade to root command
	rootCmd.AddCommand(jadeCmd)
}
