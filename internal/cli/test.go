package cli

import (
	"fmt"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/xrpl-commons/bedrock/pkg/config"
	"github.com/xrpl-commons/bedrock/pkg/tester"
	"github.com/xrpl-commons/bedrock/pkg/watcher"
)

var (
	testMatch       string
	testVerbose     bool
	testIntegration bool
	testGasReport   bool
	testWatch       bool
	testFuzz        bool
	testFuzzRuns    int
	testFuzzSeed    int64
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run contract tests",
	Long: `Run tests for the smart contract.

By default, runs Rust unit tests via cargo test.
Use --integration to run integration tests against a local node.

Examples:
  bedrock test
  bedrock test --match test_hello
  bedrock test --verbose
  bedrock test --integration
  bedrock test --gas-report
  bedrock test --watch
  bedrock test --fuzz --fuzz-runs 100`,
	RunE: runTest,
}

func init() {
	rootCmd.AddCommand(testCmd)

	testCmd.Flags().StringVarP(&testMatch, "match", "m", "", "Filter tests by name pattern")
	testCmd.Flags().BoolVarP(&testVerbose, "verbose", "V", false, "Show detailed test output")
	testCmd.Flags().BoolVar(&testIntegration, "integration", false, "Run integration tests against local node")
	testCmd.Flags().BoolVar(&testGasReport, "gas-report", false, "Show computation cost report")
	testCmd.Flags().BoolVarP(&testWatch, "watch", "w", false, "Watch for changes and re-run tests")
	testCmd.Flags().BoolVar(&testFuzz, "fuzz", false, "Enable fuzz testing")
	testCmd.Flags().IntVar(&testFuzzRuns, "fuzz-runs", 256, "Number of fuzz iterations")
	testCmd.Flags().Int64Var(&testFuzzSeed, "fuzz-seed", 0, "Seed for reproducible fuzzing")
}

func runTest(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w (run 'bedrock init' first)", err)
	}
	_ = cfg

	opts := tester.TestOptions{
		Match:       testMatch,
		Verbose:     testVerbose,
		Integration: testIntegration,
		GasReport:   testGasReport,
		Watch:       testWatch,
		Fuzz:        testFuzz,
		FuzzRuns:    testFuzzRuns,
		FuzzSeed:    testFuzzSeed,
	}

	if testWatch {
		return runTestWatch(cmd, opts)
	}

	if testIntegration {
		return runIntegrationTests(cmd, opts)
	}

	if testFuzz {
		return runFuzzTests(cmd, opts)
	}

	return runUnitTests(cmd, opts)
}

func runUnitTests(cmd *cobra.Command, opts tester.TestOptions) error {
	color.Cyan("Running unit tests\n")
	if opts.Match != "" {
		fmt.Printf("   Filter: %s\n", opts.Match)
	}
	fmt.Println()

	t := tester.New(".")
	ctx := cmd.Context()

	result, err := t.RunUnit(ctx, opts)
	if err != nil {
		color.Red("\n  Build or execution error: %v\n", err)
		return err
	}

	// Print raw output if verbose
	if opts.Verbose && result.Output != "" {
		fmt.Println(result.Output)
		fmt.Println()
	}

	// Display results
	displayTestResults(result)

	if opts.GasReport {
		color.Yellow("\nGas reporting requires --integration flag for on-chain execution\n")
	}

	if result.Failed > 0 {
		return fmt.Errorf("%d test(s) failed", result.Failed)
	}

	return nil
}

func displayTestResults(result *tester.TestResult) {
	if len(result.Tests) > 0 {
		for _, tc := range result.Tests {
			switch tc.Status {
			case tester.StatusPass:
				color.Green("  PASS  %s\n", tc.Name)
			case tester.StatusFail:
				color.Red("  FAIL  %s\n", tc.Name)
				if tc.Error != "" {
					color.Red("        %s\n", tc.Error)
				}
			case tester.StatusIgnored:
				color.Yellow("  SKIP  %s\n", tc.Name)
			}
		}
		fmt.Println()
	}

	// Summary line
	total := result.Passed + result.Failed + result.Ignored
	if result.Failed > 0 {
		color.Red("Test result: FAILED. %d passed; %d failed; %d ignored (%v)\n",
			result.Passed, result.Failed, result.Ignored, result.Duration)
	} else {
		color.Green("Test result: ok. %d/%d passed", result.Passed, total)
		if result.Ignored > 0 {
			color.Yellow(" (%d ignored)", result.Ignored)
		}
		fmt.Printf(" (%v)\n", result.Duration)
	}
}

func runIntegrationTests(cmd *cobra.Command, opts tester.TestOptions) error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	color.Cyan("Running integration tests\n")
	fmt.Println()

	runner := tester.NewIntegrationRunner(".", cfg, opts.Verbose)
	ctx := cmd.Context()

	results, err := runner.Run(ctx, opts)
	if err != nil {
		color.Red("\n  Integration test error: %v\n", err)
		return err
	}

	totalPassed := 0
	totalFailed := 0

	for _, suite := range results {
		color.Cyan("\nSuite: %s\n", suite.SuiteName)
		if suite.ContractAccount != "" {
			fmt.Printf("  Contract: %s\n", suite.ContractAccount)
		}
		fmt.Println()

		for _, test := range suite.Tests {
			if test.Passed {
				color.Green("  PASS  %s (%v)\n", test.Name, test.Duration)
			} else {
				color.Red("  FAIL  %s (%v)\n", test.Name, test.Duration)
				if test.Error != "" {
					color.Red("        Error: %s\n", test.Error)
				}
				for _, ar := range test.Assertions {
					if !ar.Passed {
						color.Red("        Assert: %s (expected: %s, got: %s)\n",
							ar.Message, ar.Expected, ar.Actual)
					}
				}
			}

			if opts.GasReport && test.GasUsed > 0 {
				fmt.Printf("        Gas: %d\n", test.GasUsed)
			}
		}

		totalPassed += suite.Passed
		totalFailed += suite.Failed
	}

	fmt.Println()
	total := totalPassed + totalFailed
	if totalFailed > 0 {
		color.Red("Integration result: FAILED. %d/%d passed\n", totalPassed, total)
		return fmt.Errorf("%d integration test(s) failed", totalFailed)
	}

	color.Green("Integration result: ok. %d/%d passed\n", totalPassed, total)
	return nil
}

func runFuzzTests(cmd *cobra.Command, opts tester.TestOptions) error {
	color.Cyan("Running fuzz tests\n")
	color.Yellow("Fuzz testing not yet implemented\n")
	return fmt.Errorf("fuzz testing coming soon (M5.1)")
}

func runTestWatch(cmd *cobra.Command, opts tester.TestOptions) error {
	cfg, err := config.LoadFromWorkingDir()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	sourceDir := filepath.Dir(cfg.Build.Source)
	color.Cyan("Watching %s for changes...\n", sourceDir)
	fmt.Println()

	// Run tests once before watching
	_ = runUnitTests(cmd, opts)

	w := watcher.New([]string{sourceDir}, []string{".rs"})
	ctx := cmd.Context()

	return w.Watch(ctx, func() {
		fmt.Println()
		color.Cyan("File changed, re-running tests...\n")
		fmt.Println()
		_ = runUnitTests(cmd, opts)
	})
}
