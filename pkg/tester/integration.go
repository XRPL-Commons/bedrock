package tester

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/xrpl-commons/bedrock/pkg/abi"
	"github.com/xrpl-commons/bedrock/pkg/builder"
	"github.com/xrpl-commons/bedrock/pkg/caller"
	"github.com/xrpl-commons/bedrock/pkg/config"
	"github.com/xrpl-commons/bedrock/pkg/deployer"
	"github.com/xrpl-commons/bedrock/pkg/faucet"
	"github.com/xrpl-commons/bedrock/pkg/network"
)

// IntegrationRunner executes integration tests against a live node
type IntegrationRunner struct {
	projectRoot string
	cfg         *config.Config
	verbose     bool
}

// NewIntegrationRunner creates a new integration test runner
func NewIntegrationRunner(projectRoot string, cfg *config.Config, verbose bool) *IntegrationRunner {
	return &IntegrationRunner{
		projectRoot: projectRoot,
		cfg:         cfg,
		verbose:     verbose,
	}
}

// IntegrationResult holds results from an integration test run
type IntegrationResult struct {
	SuiteName       string
	Passed          int
	Failed          int
	Duration        time.Duration
	Tests           []IntegrationTestResult
	ContractAccount string
}

// IntegrationTestResult holds the result of a single integration test
type IntegrationTestResult struct {
	Name       string
	Passed     bool
	Duration   time.Duration
	Assertions []AssertionResult
	Error      string
	GasUsed    int64
}

// Run executes all integration test fixtures
func (r *IntegrationRunner) Run(ctx context.Context, opts TestOptions) ([]IntegrationResult, error) {
	// Determine fixtures directory
	fixturesDir := r.cfg.Test.FixturesDir
	if fixturesDir == "" {
		fixturesDir = config.DefaultTestConfig().FixturesDir
	}

	fixtures, err := LoadFixtures(fixturesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load fixtures: %w", err)
	}

	if len(fixtures) == 0 {
		return nil, fmt.Errorf("no test fixtures found in %s", fixturesDir)
	}

	// Ensure local node is running
	if err := r.ensureNodeRunning(ctx); err != nil {
		return nil, fmt.Errorf("local node not available: %w", err)
	}

	var results []IntegrationResult
	for _, fixture := range fixtures {
		if opts.Match != "" && fixture.Name != opts.Match {
			continue
		}

		suite := ConvertFixtureToSuite(&fixture)
		result, err := r.runSuite(ctx, suite, opts)
		if err != nil {
			return nil, fmt.Errorf("suite '%s' failed: %w", suite.Name, err)
		}
		results = append(results, *result)
	}

	return results, nil
}

func (r *IntegrationRunner) runSuite(ctx context.Context, suite *IntegrationTestSuite, opts TestOptions) (*IntegrationResult, error) {
	startTime := time.Now()
	result := &IntegrationResult{
		SuiteName: suite.Name,
	}

	// Determine network config
	networkName := suite.Network
	if networkName == "" {
		networkName = r.cfg.Test.IntegrationNetwork
		if networkName == "" {
			networkName = config.DefaultTestConfig().IntegrationNetwork
		}
	}

	networkCfg, ok := r.cfg.Networks[networkName]
	if !ok {
		if networkName == "local" {
			networkCfg = config.NetworkConfig{
				URL:       "ws://localhost:6006",
				RPCURL:    "http://localhost:5005",
				NetworkID: 63456,
			}
		} else {
			return nil, fmt.Errorf("network '%s' not found in config", networkName)
		}
	}

	var contractAccount string
	var walletSeed string

	// Setup: build, deploy, and fund if needed
	if suite.Setup != nil {
		if suite.Setup.Fund {
			seed, err := r.fundWallet(ctx, networkCfg, suite.Setup.WalletSeed)
			if err != nil {
				return nil, fmt.Errorf("failed to fund wallet: %w", err)
			}
			walletSeed = seed
		} else if suite.Setup.WalletSeed != "" {
			walletSeed = suite.Setup.WalletSeed
		}

		if suite.Setup.Deploy {
			account, seed, err := r.buildAndDeploy(ctx, networkCfg, walletSeed)
			if err != nil {
				return nil, fmt.Errorf("failed to deploy contract: %w", err)
			}
			contractAccount = account
			if walletSeed == "" {
				walletSeed = seed
			}
		}
	}

	result.ContractAccount = contractAccount

	// Run each test
	for _, test := range suite.Tests {
		testResult := r.runTest(ctx, test, contractAccount, walletSeed, networkCfg, opts)
		if testResult.Passed {
			result.Passed++
		} else {
			result.Failed++
		}
		result.Tests = append(result.Tests, testResult)
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (r *IntegrationRunner) runTest(ctx context.Context, test IntegrationTest, contractAccount string, walletSeed string, networkCfg config.NetworkConfig, opts TestOptions) IntegrationTestResult {
	startTime := time.Now()
	result := IntegrationTestResult{
		Name: test.Name,
	}

	// Build call config
	abiPath := "abi.json"
	if contracts := r.cfg.Contracts; contracts != nil {
		if main, ok := contracts["main"]; ok {
			abiPath = main.ABI
		}
	}

	c, err := caller.NewCaller(r.verbose)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create caller: %v", err)
		result.Duration = time.Since(startTime)
		return result
	}

	callResult, err := c.Call(ctx, caller.CallConfig{
		ContractAccount:      contractAccount,
		FunctionName:         test.Function,
		NetworkURL:           networkCfg.URL,
		NetworkID:            networkCfg.NetworkID,
		WalletSeed:           walletSeed,
		Algorithm:            "secp256k1",
		ABIPath:              abiPath,
		Parameters:           test.Parameters,
		ComputationAllowance: "1000000",
		Fee:                  "1000000",
	})

	if err != nil {
		result.Error = fmt.Sprintf("contract call failed: %v", err)
		result.Duration = time.Since(startTime)
		return result
	}

	result.GasUsed = callResult.GasUsed

	// Run assertions
	assertionResults := RunAssertions(callResult, test.Assertions)
	result.Assertions = assertionResults

	allPassed := true
	for _, ar := range assertionResults {
		if !ar.Passed {
			allPassed = false
			break
		}
	}

	result.Passed = allPassed
	result.Duration = time.Since(startTime)
	return result
}

func (r *IntegrationRunner) ensureNodeRunning(ctx context.Context) error {
	manager, err := network.NewManager()
	if err != nil {
		return fmt.Errorf("failed to create network manager: %w", err)
	}

	status, err := manager.Status(ctx)
	if err != nil {
		return fmt.Errorf("failed to check node status: %w", err)
	}

	if !status.Running {
		return fmt.Errorf("local node is not running (start with: bedrock node start)")
	}

	return nil
}

func (r *IntegrationRunner) fundWallet(ctx context.Context, networkCfg config.NetworkConfig, walletSeed string) (string, error) {
	f, err := faucet.NewFaucet(r.verbose)
	if err != nil {
		return "", fmt.Errorf("failed to create faucet: %w", err)
	}

	isLocal := networkCfg.URL == "ws://localhost:6006"
	result, err := f.Request(ctx, faucet.FaucetConfig{
		FaucetURL:  networkCfg.FaucetURL,
		WalletSeed: walletSeed,
		Algorithm:  "secp256k1",
		NetworkURL: networkCfg.URL,
		IsLocal:    isLocal,
	})
	if err != nil {
		return "", err
	}

	return result.WalletSeed, nil
}

func (r *IntegrationRunner) buildAndDeploy(ctx context.Context, networkCfg config.NetworkConfig, walletSeed string) (string, string, error) {
	// Build
	b := builder.New(r.projectRoot)
	buildResult, err := b.Build(ctx, builder.BuildOptions{Release: true})
	if err != nil {
		return "", "", fmt.Errorf("build failed: %w", err)
	}

	// Generate ABI
	sourceDir := filepath.Dir(r.cfg.Build.Source)
	parser := abi.NewParser(sourceDir)
	abiData, err := parser.ParseContract(r.cfg.Project.Name)
	if err != nil {
		return "", "", fmt.Errorf("ABI generation failed: %w", err)
	}

	generator := abi.NewGenerator(".")
	abiPath, err := generator.Generate(abiData, "abi.json")
	if err != nil {
		return "", "", fmt.Errorf("failed to write ABI: %w", err)
	}

	// Deploy
	d, err := deployer.NewDeployer(r.verbose)
	if err != nil {
		return "", "", fmt.Errorf("failed to create deployer: %w", err)
	}

	deployResult, err := d.Deploy(ctx, deployer.DeploymentConfig{
		WasmPath:   buildResult.WasmPath,
		ABIPath:    abiPath,
		NetworkURL: networkCfg.URL,
		WalletSeed: walletSeed,
		Algorithm:  "secp256k1",
		FaucetURL:  networkCfg.FaucetURL,
	})
	if err != nil {
		return "", "", fmt.Errorf("deployment failed: %w", err)
	}

	return deployResult.ContractAccount, deployResult.WalletSeed, nil
}
