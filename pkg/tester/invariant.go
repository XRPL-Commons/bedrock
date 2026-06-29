package tester

import (
	"context"
	"fmt"
	"time"

	"github.com/xrpl-commons/bedrock/pkg/abi"
	"github.com/xrpl-commons/bedrock/pkg/caller"
	"github.com/xrpl-commons/bedrock/pkg/config"
)

// Invariant defines a property that must hold after every contract call
type Invariant struct {
	Name        string      `toml:"name" json:"name"`
	Function    string      `toml:"function" json:"function"`
	Assertions  []Assertion `toml:"assertions" json:"assertions"`
	Description string      `toml:"description" json:"description"`
}

// InvariantResult contains results from invariant checking
type InvariantResult struct {
	Name       string
	Iterations int
	Violations int
	Duration   time.Duration
	Details    []string
}

// InvariantRunner runs property-based invariant tests
type InvariantRunner struct {
	cfg     *config.Config
	verbose bool
}

// NewInvariantRunner creates a new invariant test runner
func NewInvariantRunner(cfg *config.Config, verbose bool) *InvariantRunner {
	return &InvariantRunner{cfg: cfg, verbose: verbose}
}

// Run executes random sequences of contract calls and checks invariants
func (r *InvariantRunner) Run(ctx context.Context, contractAccount string, walletSeed string, invariants []Invariant, functions []abi.Function, opts TestOptions) ([]InvariantResult, error) {
	abiPath := "abi.json"
	if contracts := r.cfg.Contracts; contracts != nil {
		if main, ok := contracts["main"]; ok {
			abiPath = main.ABI
		}
	}

	networkName := r.cfg.Test.IntegrationNetwork
	if networkName == "" {
		networkName = "local"
	}

	networkCfg, ok := r.cfg.Networks[networkName]
	if !ok {
		if networkName == "local" {
			networkCfg = config.NetworkConfig{
				URL:       "ws://localhost:6006",
				NetworkID: 100,
			}
		} else {
			return nil, fmt.Errorf("network '%s' not found", networkName)
		}
	}

	seed := opts.FuzzSeed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	gen := NewValueGenerator(seed)

	runs := opts.FuzzRuns
	if runs <= 0 {
		runs = 100
	}

	c, err := caller.NewCaller(r.verbose)
	if err != nil {
		return nil, fmt.Errorf("failed to create caller: %w", err)
	}

	var results []InvariantResult

	for _, inv := range invariants {
		result := r.checkInvariant(ctx, inv, contractAccount, walletSeed, networkCfg, functions, gen, c, runs, abiPath)
		results = append(results, result)
	}

	return results, nil
}

func (r *InvariantRunner) checkInvariant(ctx context.Context, inv Invariant, contractAccount string, walletSeed string, networkCfg config.NetworkConfig, functions []abi.Function, gen *ValueGenerator, c *caller.Caller, runs int, abiPath string) InvariantResult {
	startTime := time.Now()
	result := InvariantResult{
		Name: inv.Name,
	}

	for i := 0; i < runs; i++ {
		select {
		case <-ctx.Done():
			result.Duration = time.Since(startTime)
			return result
		default:
		}

		result.Iterations++

		// Pick a random function to call
		fn := functions[gen.rng.Intn(len(functions))]

		// Generate random parameters
		params := make(map[string]interface{})
		for _, p := range fn.Parameters {
			params[p.Name] = gen.Generate(p.Type)
		}

		// Execute the call
		callResult, err := c.Call(ctx, caller.CallConfig{
			ContractAccount:      contractAccount,
			FunctionName:         fn.Name,
			NetworkURL:           networkCfg.URL,
			NetworkID:            networkCfg.NetworkID,
			WalletSeed:           walletSeed,
			Algorithm:            "secp256k1",
			ABIPath:              abiPath,
			Parameters:           params,
			ComputationAllowance: "1000000",
			Fee:                  "1000000",
		})

		if err != nil {
			continue // Skip failed calls for invariant checking
		}

		// Check invariant function if specified
		if inv.Function != "" {
			invResult, err := c.Call(ctx, caller.CallConfig{
				ContractAccount:      contractAccount,
				FunctionName:         inv.Function,
				NetworkURL:           networkCfg.URL,
				NetworkID:            networkCfg.NetworkID,
				WalletSeed:           walletSeed,
				Algorithm:            "secp256k1",
				ABIPath:              abiPath,
				ComputationAllowance: "1000000",
				Fee:                  "1000000",
			})

			if err != nil {
				result.Violations++
				result.Details = append(result.Details, fmt.Sprintf("invariant call failed after %s(%v): %v", fn.Name, params, err))
				continue
			}

			// Check assertions on invariant result
			for _, assertion := range inv.Assertions {
				ar := runAssertion(invResult, assertion)
				if !ar.Passed {
					result.Violations++
					result.Details = append(result.Details, fmt.Sprintf("invariant violated after %s: %s (expected: %s, got: %s)", fn.Name, ar.Message, ar.Expected, ar.Actual))
				}
			}
		} else {
			// Check assertions directly on the call result
			for _, assertion := range inv.Assertions {
				ar := runAssertion(callResult, assertion)
				if !ar.Passed {
					result.Violations++
					result.Details = append(result.Details, fmt.Sprintf("invariant violated after %s: %s", fn.Name, ar.Message))
				}
			}
		}
	}

	result.Duration = time.Since(startTime)
	return result
}
