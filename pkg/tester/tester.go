package tester

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Tester handles running contract tests
type Tester struct {
	projectRoot string
}

// New creates a new Tester
func New(projectRoot string) *Tester {
	return &Tester{projectRoot: projectRoot}
}

// RunUnit executes cargo test on the contract project
func (t *Tester) RunUnit(ctx context.Context, opts TestOptions) (*TestResult, error) {
	if err := t.verifyToolchain(); err != nil {
		return nil, err
	}

	contractDir := filepath.Join(t.projectRoot, "contract")

	if _, err := os.Stat(filepath.Join(contractDir, "Cargo.toml")); os.IsNotExist(err) {
		return nil, fmt.Errorf("Cargo.toml not found in %s", contractDir)
	}

	args := []string{"test"}

	if opts.Match != "" {
		args = append(args, opts.Match)
	}

	if opts.Verbose {
		args = append(args, "--", "--nocapture")
	}

	cmd := exec.CommandContext(ctx, "cargo", args...)
	cmd.Dir = contractDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	// Combine stdout and stderr for parsing (cargo test outputs to both)
	combined := stderr.String() + stdout.String()

	result := t.parseTestOutput(combined)
	result.Duration = duration
	result.Output = combined

	if err != nil {
		// cargo test returns non-zero when tests fail -- that is expected
		if result.Failed > 0 {
			return result, nil
		}
		// Actual execution error (compilation failure, etc.)
		return nil, fmt.Errorf("cargo test failed: %w\n%s", err, combined)
	}

	return result, nil
}

// parseTestOutput parses the output of cargo test into structured results
func (t *Tester) parseTestOutput(output string) *TestResult {
	result := &TestResult{}

	// Match individual test results: "test tests::test_name ... ok" or "... FAILED"
	testPattern := regexp.MustCompile(`test\s+([\w:]+)\s+\.\.\.\s+(\w+)`)
	// Match summary line: "test result: ok. 3 passed; 0 failed; 0 ignored"
	summaryPattern := regexp.MustCompile(`test result:\s+(\w+)\.\s+(\d+)\s+passed;\s+(\d+)\s+failed;\s+(\d+)\s+ignored`)

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		// Parse individual test results
		if matches := testPattern.FindStringSubmatch(line); matches != nil {
			tc := TestCase{
				Name: matches[1],
			}
			switch matches[2] {
			case "ok":
				tc.Status = StatusPass
			case "FAILED":
				tc.Status = StatusFail
			case "ignored":
				tc.Status = StatusIgnored
			}
			result.Tests = append(result.Tests, tc)
		}

		// Parse summary line
		if matches := summaryPattern.FindStringSubmatch(line); matches != nil {
			_, _ = fmt.Sscanf(matches[2], "%d", &result.Passed)
			_, _ = fmt.Sscanf(matches[3], "%d", &result.Failed)
			_, _ = fmt.Sscanf(matches[4], "%d", &result.Ignored)
		}
	}

	return result
}

// verifyToolchain checks if cargo is installed
func (t *Tester) verifyToolchain() error {
	if _, err := exec.LookPath("cargo"); err != nil {
		return fmt.Errorf("cargo not found: please install Rust from https://rustup.rs")
	}
	return nil
}
