package cli

import (
	"github.com/spf13/cobra"
)

// globalVerbose is the resolved value of the persistent --verbose flag.
// Subcommands read it via Verbose() rather than poking at cobra directly.
var globalVerbose bool

var rootCmd = &cobra.Command{
	Use:   "bedrock",
	Short: "The unshakeable foundation for XRPL smart contracts",
	Long: `BEDROCK - XRPL Smart Contract CLI
The foundation for XRPL smart contract development

Build, deploy, and interact with XRPL smart contracts written in Rust.`,
}

func Execute() error {
	return rootCmd.Execute()
}

// Verbose returns whether the user requested verbose output via -v/--verbose.
func Verbose() bool { return globalVerbose }

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&globalVerbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().Bool("json", false, "output in JSON format for scripting")
}
