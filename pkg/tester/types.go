package tester

import "time"

// TestOptions configures the test execution
type TestOptions struct {
	Match       string // Filter test names by pattern
	Verbose     bool   // Show detailed output
	Integration bool   // Run integration tests
	GasReport   bool   // Show gas/computation report
	Watch       bool   // Watch mode (re-run on changes)
	Fuzz        bool   // Enable fuzz testing
	FuzzRuns    int    // Number of fuzz iterations
	FuzzSeed    int64  // Seed for reproducible fuzzing
}

// TestResult contains the outcome of a test run
type TestResult struct {
	Passed   int           // Number of passed tests
	Failed   int           // Number of failed tests
	Ignored  int           // Number of ignored tests
	Duration time.Duration // Total test duration
	Tests    []TestCase    // Individual test results
	Output   string        // Raw output from cargo test
}

// TestCase represents a single test result
type TestCase struct {
	Name     string        // Test name (e.g., "tests::test_hello")
	Status   TestStatus    // Pass, Fail, or Ignored
	Duration time.Duration // Test execution time
	Output   string        // Test-specific output (stdout capture)
	Error    string        // Error message if failed
}

// TestStatus represents the outcome of a single test
type TestStatus int

const (
	StatusPass TestStatus = iota
	StatusFail
	StatusIgnored
)

// String returns a human-readable status string
func (s TestStatus) String() string {
	switch s {
	case StatusPass:
		return "PASS"
	case StatusFail:
		return "FAIL"
	case StatusIgnored:
		return "IGNORED"
	default:
		return "UNKNOWN"
	}
}

// GasReport contains computation cost data for functions
type GasReport struct {
	Entries  []GasEntry
	Duration time.Duration
}

// GasEntry represents computation cost for a single function call
type GasEntry struct {
	Function string // Function name
	GasUsed  int64  // Computation units consumed
	TxHash   string // Transaction hash
}

// Snapshot represents a saved gas snapshot
type Snapshot struct {
	Entries   map[string]int64 // Function name -> gas used
	CreatedAt time.Time
}

// SnapshotDiff represents differences between two snapshots
type SnapshotDiff struct {
	Added   map[string]int64    // New entries
	Removed map[string]int64    // Removed entries
	Changed map[string][2]int64 // Changed entries [old, new]
}

// IntegrationTestSuite represents a collection of integration test fixtures
type IntegrationTestSuite struct {
	Name    string            // Suite name
	Network string            // Target network (default: local)
	Setup   *IntegrationSetup // Optional setup steps
	Tests   []IntegrationTest // Test cases
}

// IntegrationSetup defines pre-test setup steps
type IntegrationSetup struct {
	Deploy     bool                   // Deploy contract before tests
	Fund       bool                   // Fund wallet from faucet
	WalletSeed string                 // Wallet seed to use
	Params     map[string]interface{} // Deploy parameters
}

// IntegrationTest defines a single integration test case
type IntegrationTest struct {
	Name       string                 // Test name
	Function   string                 // Contract function to call
	Parameters map[string]interface{} // Call parameters
	Assertions []Assertion            // Expected outcomes
}

// Assertion defines an expected condition to verify
type Assertion struct {
	Type     AssertionType // Type of assertion
	Expected interface{}   // Expected value
	Field    string        // Field to check (for nested assertions)
}

// AssertionType defines what kind of assertion to perform
type AssertionType int

const (
	AssertReturnValue AssertionType = iota
	AssertReturnCode
	AssertTxSuccess
	AssertTxFailure
	AssertEvent
	AssertState
	AssertGasBelow
)

// String returns a human-readable assertion type
func (a AssertionType) String() string {
	switch a {
	case AssertReturnValue:
		return "return_value"
	case AssertReturnCode:
		return "return_code"
	case AssertTxSuccess:
		return "tx_success"
	case AssertTxFailure:
		return "tx_failure"
	case AssertEvent:
		return "event"
	case AssertState:
		return "state"
	case AssertGasBelow:
		return "gas_below"
	default:
		return "unknown"
	}
}
