package tester

import (
	"fmt"
	"sort"
	"strings"
)

// FormatGasReport generates a formatted gas report table from integration test results
func FormatGasReport(results []IntegrationResult) string {
	var entries []GasEntry

	for _, suite := range results {
		for _, test := range suite.Tests {
			if test.GasUsed > 0 {
				entries = append(entries, GasEntry{
					Function: test.Name,
					GasUsed:  test.GasUsed,
				})
			}
		}
	}

	if len(entries) == 0 {
		return "No gas data available"
	}

	return formatGasTable(entries)
}

// FormatGasReportFromEntries formats a gas report from raw entries
func FormatGasReportFromEntries(entries []GasEntry) string {
	if len(entries) == 0 {
		return "No gas data available"
	}
	return formatGasTable(entries)
}

func formatGasTable(entries []GasEntry) string {
	// Sort by gas used (descending)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].GasUsed > entries[j].GasUsed
	})

	// Calculate column widths
	maxNameLen := len("Function")
	maxGasLen := len("Gas Used")
	for _, e := range entries {
		if len(e.Function) > maxNameLen {
			maxNameLen = len(e.Function)
		}
		gasStr := fmt.Sprintf("%d", e.GasUsed)
		if len(gasStr) > maxGasLen {
			maxGasLen = len(gasStr)
		}
	}

	var sb strings.Builder

	// Header
	separator := fmt.Sprintf("+-%s-+-%s-+", strings.Repeat("-", maxNameLen), strings.Repeat("-", maxGasLen))
	sb.WriteString(separator + "\n")
	fmt.Fprintf(&sb, "| %-*s | %-*s |\n", maxNameLen, "Function", maxGasLen, "Gas Used")
	sb.WriteString(separator + "\n")

	// Rows
	for _, e := range entries {
		fmt.Fprintf(&sb, "| %-*s | %*d |\n", maxNameLen, e.Function, maxGasLen, e.GasUsed)
	}

	sb.WriteString(separator + "\n")

	// Summary
	var total int64
	for _, e := range entries {
		total += e.GasUsed
	}
	avg := total / int64(len(entries))
	fmt.Fprintf(&sb, "Total: %d | Average: %d | Functions: %d\n", total, avg, len(entries))

	return sb.String()
}
