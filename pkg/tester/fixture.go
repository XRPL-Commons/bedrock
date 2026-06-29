package tester

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// FixtureFile represents a test fixture file (TOML or JSON)
type FixtureFile struct {
	Name    string        `toml:"name" json:"name"`
	Network string        `toml:"network" json:"network"`
	Setup   FixtureSetup  `toml:"setup" json:"setup"`
	Tests   []FixtureTest `toml:"tests" json:"tests"`
}

// FixtureSetup defines setup steps before running tests
type FixtureSetup struct {
	Deploy     bool                   `toml:"deploy" json:"deploy"`
	Fund       bool                   `toml:"fund" json:"fund"`
	WalletSeed string                 `toml:"wallet_seed" json:"wallet_seed"`
	Params     map[string]interface{} `toml:"params" json:"params"`
}

// FixtureTest defines a single test case in a fixture
type FixtureTest struct {
	Name       string                 `toml:"name" json:"name"`
	Function   string                 `toml:"function" json:"function"`
	Parameters map[string]interface{} `toml:"parameters" json:"parameters"`
	Expect     FixtureExpect          `toml:"expect" json:"expect"`
}

// FixtureExpect defines expected outcomes for a test
type FixtureExpect struct {
	ReturnCode  *int                     `toml:"return_code" json:"return_code,omitempty"`
	ReturnValue *string                  `toml:"return_value" json:"return_value,omitempty"`
	TxResult    *string                  `toml:"tx_result" json:"tx_result,omitempty"`
	GasBelow    *int64                   `toml:"gas_below" json:"gas_below,omitempty"`
	Events      []map[string]interface{} `toml:"events" json:"events,omitempty"`
}

// LoadFixtures loads all test fixtures from a directory
func LoadFixtures(dir string) ([]FixtureFile, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("fixtures directory not found: %s", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read fixtures directory: %w", err)
	}

	var fixtures []FixtureFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".toml" && ext != ".json" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		fixture, err := LoadFixture(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load fixture %s: %w", entry.Name(), err)
		}

		if fixture.Name == "" {
			fixture.Name = strings.TrimSuffix(entry.Name(), ext)
		}

		fixtures = append(fixtures, *fixture)
	}

	return fixtures, nil
}

// LoadFixture loads a single test fixture file
func LoadFixture(path string) (*FixtureFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read fixture file: %w", err)
	}

	var fixture FixtureFile
	ext := filepath.Ext(path)

	switch ext {
	case ".toml":
		if err := toml.Unmarshal(data, &fixture); err != nil {
			return nil, fmt.Errorf("failed to parse TOML fixture: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &fixture); err != nil {
			return nil, fmt.Errorf("failed to parse JSON fixture: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported fixture format: %s (use .toml or .json)", ext)
	}

	if err := validateFixture(&fixture); err != nil {
		return nil, fmt.Errorf("invalid fixture: %w", err)
	}

	return &fixture, nil
}

// ConvertFixtureToSuite converts a FixtureFile to an IntegrationTestSuite
func ConvertFixtureToSuite(f *FixtureFile) *IntegrationTestSuite {
	suite := &IntegrationTestSuite{
		Name:    f.Name,
		Network: f.Network,
		Setup: &IntegrationSetup{
			Deploy:     f.Setup.Deploy,
			Fund:       f.Setup.Fund,
			WalletSeed: f.Setup.WalletSeed,
			Params:     f.Setup.Params,
		},
	}

	for _, ft := range f.Tests {
		test := IntegrationTest{
			Name:       ft.Name,
			Function:   ft.Function,
			Parameters: ft.Parameters,
			Assertions: convertExpectToAssertions(ft.Expect),
		}
		suite.Tests = append(suite.Tests, test)
	}

	return suite
}

func convertExpectToAssertions(expect FixtureExpect) []Assertion {
	var assertions []Assertion

	if expect.ReturnCode != nil {
		assertions = append(assertions, Assertion{
			Type:     AssertReturnCode,
			Expected: *expect.ReturnCode,
		})
	}

	if expect.ReturnValue != nil {
		assertions = append(assertions, Assertion{
			Type:     AssertReturnValue,
			Expected: *expect.ReturnValue,
		})
	}

	if expect.TxResult != nil {
		if *expect.TxResult == "tesSUCCESS" {
			assertions = append(assertions, Assertion{
				Type: AssertTxSuccess,
			})
		} else {
			assertions = append(assertions, Assertion{
				Type:     AssertTxFailure,
				Expected: *expect.TxResult,
			})
		}
	}

	if expect.GasBelow != nil {
		assertions = append(assertions, Assertion{
			Type:     AssertGasBelow,
			Expected: *expect.GasBelow,
		})
	}

	for _, event := range expect.Events {
		assertions = append(assertions, Assertion{
			Type:     AssertEvent,
			Expected: event,
		})
	}

	return assertions
}

func validateFixture(f *FixtureFile) error {
	if len(f.Tests) == 0 {
		return fmt.Errorf("fixture must contain at least one test")
	}

	for i, t := range f.Tests {
		if t.Function == "" {
			return fmt.Errorf("test %d (%s): function is required", i, t.Name)
		}
	}

	return nil
}
