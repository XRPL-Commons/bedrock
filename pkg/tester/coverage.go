package tester

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CoverageOptions configures coverage collection
type CoverageOptions struct {
	LCOV   bool   // Generate LCOV format
	HTML   bool   // Generate HTML report
	Output string // Output path
}

// CoverageResult contains coverage data
type CoverageResult struct {
	Summary  string // Human-readable summary
	LCOVPath string // Path to LCOV file if generated
	HTMLPath string // Path to HTML report if generated
	Duration time.Duration
}

// RunCoverage executes tests with code coverage
func RunCoverage(ctx context.Context, projectRoot string, opts CoverageOptions) (*CoverageResult, error) {
	contractDir := filepath.Join(projectRoot, "contract")

	if _, err := os.Stat(filepath.Join(contractDir, "Cargo.toml")); os.IsNotExist(err) {
		return nil, fmt.Errorf("Cargo.toml not found in %s", contractDir)
	}

	startTime := time.Now()

	// Try cargo-tarpaulin first, fall back to cargo-llvm-cov
	result, err := runTarpaulin(ctx, contractDir, opts)
	if err != nil {
		// Try llvm-cov as fallback
		result, err = runLLVMCov(ctx, contractDir, opts)
		if err != nil {
			return nil, fmt.Errorf("coverage tools not available. Install with:\n  cargo install cargo-tarpaulin\nor:\n  cargo install cargo-llvm-cov")
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func runTarpaulin(ctx context.Context, contractDir string, opts CoverageOptions) (*CoverageResult, error) {
	if _, err := exec.LookPath("cargo-tarpaulin"); err != nil {
		return nil, fmt.Errorf("cargo-tarpaulin not found")
	}

	args := []string{"tarpaulin", "--skip-clean"}

	if opts.LCOV {
		args = append(args, "--out", "lcov")
		if opts.Output != "" {
			args = append(args, "--output-dir", opts.Output)
		}
	}

	cmd := exec.CommandContext(ctx, "cargo", args...)
	cmd.Dir = contractDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("tarpaulin failed: %w\n%s", err, stderr.String())
	}

	result := &CoverageResult{
		Summary: extractCoverageSummary(stdout.String() + stderr.String()),
	}

	if opts.LCOV {
		lcovPath := filepath.Join(contractDir, "lcov.info")
		if opts.Output != "" {
			lcovPath = filepath.Join(opts.Output, "lcov.info")
		}
		if _, err := os.Stat(lcovPath); err == nil {
			result.LCOVPath = lcovPath
		}
	}

	return result, nil
}

func runLLVMCov(ctx context.Context, contractDir string, opts CoverageOptions) (*CoverageResult, error) {
	if _, err := exec.LookPath("cargo-llvm-cov"); err != nil {
		return nil, fmt.Errorf("cargo-llvm-cov not found")
	}

	args := []string{"llvm-cov"}

	if opts.LCOV {
		args = append(args, "--lcov", "--output-path")
		lcovPath := "lcov.info"
		if opts.Output != "" {
			lcovPath = filepath.Join(opts.Output, "lcov.info")
		}
		args = append(args, lcovPath)
	}

	cmd := exec.CommandContext(ctx, "cargo", args...)
	cmd.Dir = contractDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("llvm-cov failed: %w\n%s", err, stderr.String())
	}

	result := &CoverageResult{
		Summary: extractCoverageSummary(stdout.String() + stderr.String()),
	}

	if opts.LCOV {
		lcovPath := "lcov.info"
		if opts.Output != "" {
			lcovPath = filepath.Join(opts.Output, "lcov.info")
		}
		if _, err := os.Stat(lcovPath); err == nil {
			result.LCOVPath = lcovPath
		}
	}

	return result, nil
}

func extractCoverageSummary(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "coverage") && (strings.Contains(lower, "%") || strings.Contains(lower, "total")) {
			return strings.TrimSpace(line)
		}
	}
	return output
}
