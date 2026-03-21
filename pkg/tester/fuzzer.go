package tester

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/xrpl-commons/bedrock/pkg/abi"
	"github.com/xrpl-commons/bedrock/pkg/caller"
	"github.com/xrpl-commons/bedrock/pkg/config"
)

// Fuzzer performs fuzz testing on contract functions
type Fuzzer struct {
	cfg     *config.Config
	verbose bool
}

// FuzzResult contains the outcome of a fuzz test run
type FuzzResult struct {
	Function   string
	Runs       int
	Failures   int
	Duration   time.Duration
	FailInputs []map[string]interface{} // Inputs that caused failures
}

// NewFuzzer creates a new fuzz tester
func NewFuzzer(cfg *config.Config, verbose bool) *Fuzzer {
	return &Fuzzer{cfg: cfg, verbose: verbose}
}

// Run executes fuzz testing for all contract functions
func (f *Fuzzer) Run(ctx context.Context, contractAccount string, walletSeed string, opts TestOptions) ([]FuzzResult, error) {
	// Load ABI
	abiPath := "abi.json"
	if contracts := f.cfg.Contracts; contracts != nil {
		if main, ok := contracts["main"]; ok {
			abiPath = main.ABI
		}
	}

	abiData, err := loadABI(abiPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load ABI: %w", err)
	}

	networkName := f.cfg.Test.IntegrationNetwork
	if networkName == "" {
		networkName = "local"
	}

	networkCfg, ok := f.cfg.Networks[networkName]
	if !ok {
		if networkName == "local" {
			networkCfg = config.NetworkConfig{
				URL:       "ws://localhost:6006",
				RPCURL:    "http://localhost:5005",
				NetworkID: 63456,
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
	var results []FuzzResult

	for _, fn := range abiData.Functions {
		if opts.Match != "" && fn.Name != opts.Match {
			continue
		}

		result := f.fuzzFunction(ctx, fn, contractAccount, walletSeed, networkCfg, gen, opts, abiPath)
		results = append(results, result)
	}

	return results, nil
}

func (f *Fuzzer) fuzzFunction(ctx context.Context, fn abi.Function, contractAccount string, walletSeed string, networkCfg config.NetworkConfig, gen *ValueGenerator, opts TestOptions, abiPath string) FuzzResult {
	startTime := time.Now()
	result := FuzzResult{
		Function: fn.Name,
	}

	runs := opts.FuzzRuns
	if runs <= 0 {
		runs = 256
	}

	c, err := caller.NewCaller(f.verbose)
	if err != nil {
		result.Failures = runs
		return result
	}

	for i := 0; i < runs; i++ {
		select {
		case <-ctx.Done():
			result.Duration = time.Since(startTime)
			result.Runs = i
			return result
		default:
		}

		// Generate random parameters
		params := make(map[string]interface{})
		for _, p := range fn.Parameters {
			params[p.Name] = gen.Generate(p.Type)
		}

		result.Runs++

		// Call the function with generated params
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
			result.Failures++
			result.FailInputs = append(result.FailInputs, params)
			continue
		}

		// Check for unexpected failures (non-success transaction result)
		if callResult.TransactionResult != "tesSUCCESS" && callResult.TransactionResult != "" {
			result.Failures++
			result.FailInputs = append(result.FailInputs, params)
		}
	}

	result.Duration = time.Since(startTime)
	return result
}

func loadABI(path string) (*abi.ABI, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read ABI: %w", err)
	}

	var a abi.ABI
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	return &a, nil
}
