package doc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xrpl-commons/bedrock/pkg/abi"
)

// Generator creates documentation from ABI annotations
type Generator struct {
	outputDir string
}

// NewGenerator creates a new documentation generator
func NewGenerator(outputDir string) *Generator {
	return &Generator{outputDir: outputDir}
}

// Generate creates Markdown documentation from an ABI
func (g *Generator) Generate(abiData *abi.ABI) (string, error) {
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, "# %s\n\n", abiData.ContractName)
	sb.WriteString("## Functions\n\n")

	for _, fn := range abiData.Functions {
		fmt.Fprintf(&sb, "### `%s`\n\n", fn.Name)

		// Signature
		var paramTypes []string
		for _, p := range fn.Parameters {
			paramTypes = append(paramTypes, fmt.Sprintf("%s: %s", p.Name, p.Type))
		}

		retStr := ""
		if fn.Returns != nil {
			retStr = fmt.Sprintf(" -> %s", fn.Returns.Type)
		}

		fmt.Fprintf(&sb, "```\n%s(%s)%s\n```\n\n", fn.Name, strings.Join(paramTypes, ", "), retStr)

		// Parameters
		if len(fn.Parameters) > 0 {
			sb.WriteString("**Parameters:**\n\n")
			sb.WriteString("| Name | Type | Description |\n")
			sb.WriteString("|------|------|-------------|\n")
			for _, p := range fn.Parameters {
				desc := p.Description
				if desc == "" {
					desc = "-"
				}
				fmt.Fprintf(&sb, "| `%s` | `%s` | %s |\n", p.Name, p.Type, desc)
			}
			sb.WriteString("\n")
		}

		// Return type
		if fn.Returns != nil {
			desc := fn.Returns.Description
			if desc == "" {
				desc = fn.Returns.Type
			}
			fmt.Fprintf(&sb, "**Returns:** `%s` - %s\n\n", fn.Returns.Type, desc)
		}

		sb.WriteString("---\n\n")
	}

	// Type reference
	sb.WriteString("## Type Reference\n\n")
	sb.WriteString("| Type | Rust Equivalent | Description |\n")
	sb.WriteString("|------|-----------------|-------------|\n")
	for _, info := range abi.ValidXRPLTypes {
		fmt.Fprintf(&sb, "| `%s` | `%s` | %s |\n", info.Name, info.RustType, info.Description)
	}

	outputPath := filepath.Join(g.outputDir, "README.md")
	if err := os.WriteFile(outputPath, []byte(sb.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write documentation: %w", err)
	}

	return outputPath, nil
}
