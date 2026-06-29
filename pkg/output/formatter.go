package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// Format determines the output format
type Format int

const (
	FormatHuman Format = iota
	FormatJSON
)

// Formatter handles output formatting
type Formatter struct {
	format Format
}

// NewFormatter creates a new output formatter
func NewFormatter(jsonMode bool) *Formatter {
	f := FormatHuman
	if jsonMode {
		f = FormatJSON
	}
	return &Formatter{format: f}
}

// IsJSON returns true if JSON output mode is active
func (f *Formatter) IsJSON() bool {
	return f.format == FormatJSON
}

// Print outputs data in the configured format
func (f *Formatter) Print(data interface{}) {
	if f.format == FormatJSON {
		f.printJSON(data)
	} else {
		fmt.Printf("%v\n", data)
	}
}

// PrintResult outputs a result with a key-value structure
func (f *Formatter) PrintResult(result map[string]interface{}) {
	if f.format == FormatJSON {
		f.printJSON(result)
	} else {
		for k, v := range result {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}
}

// PrintError outputs an error in the configured format
func (f *Formatter) PrintError(err error) {
	if f.format == FormatJSON {
		f.printJSON(map[string]interface{}{
			"error":   true,
			"message": err.Error(),
		})
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

func (f *Formatter) printJSON(data interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(data)
}
