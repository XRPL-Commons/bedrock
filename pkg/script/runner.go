package script

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/xrpl-commons/bedrock/pkg/caller"
	"github.com/xrpl-commons/bedrock/pkg/config"
	"github.com/xrpl-commons/bedrock/pkg/deployer"
	"github.com/xrpl-commons/bedrock/pkg/faucet"
)

// Runner executes scripts step-by-step
type Runner struct {
	cfg       *config.Config
	variables map[string]string
	verbose   bool
}

// StepResult contains the outcome of a single step
type StepResult struct {
	Name     string
	Action   string
	Success  bool
	Duration time.Duration
	Output   map[string]string
	Error    string
}

// RunResult contains the outcome of a full script run
type RunResult struct {
	ScriptName string
	Steps      []StepResult
	Duration   time.Duration
	Success    bool
}

// NewRunner creates a new script runner
func NewRunner(cfg *config.Config, verbose bool) *Runner {
	return &Runner{
		cfg:       cfg,
		variables: make(map[string]string),
		verbose:   verbose,
	}
}

// Run executes all steps in a script
func (r *Runner) Run(ctx context.Context, script *Script) (*RunResult, error) {
	startTime := time.Now()
	result := &RunResult{
		ScriptName: script.Name,
		Success:    true,
	}

	// Initialize variables
	for k, v := range script.Variables {
		r.variables[k] = v
	}

	// Resolve network
	networkName := script.Network
	if networkName == "" {
		networkName = "local"
	}

	for _, step := range script.Steps {
		stepResult := r.executeStep(ctx, step, networkName)
		result.Steps = append(result.Steps, stepResult)

		if !stepResult.Success {
			result.Success = false
			break
		}

		// Store variables from step output
		if step.Store != "" && stepResult.Output != nil {
			for k, v := range stepResult.Output {
				r.variables[step.Store+"."+k] = v
			}
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

func (r *Runner) executeStep(ctx context.Context, step Step, networkName string) StepResult {
	startTime := time.Now()
	result := StepResult{
		Name:   step.Name,
		Action: step.Action,
	}

	switch step.Action {
	case "deploy":
		r.executeDeploy(ctx, step, networkName, &result)
	case "call":
		r.executeCall(ctx, step, networkName, &result)
	case "fund":
		r.executeFund(ctx, step, networkName, &result)
	case "wait":
		r.executeWait(step, &result)
	case "assert":
		r.executeAssert(step, &result)
	}

	result.Duration = time.Since(startTime)
	return result
}

func (r *Runner) executeDeploy(ctx context.Context, step Step, networkName string, result *StepResult) {
	networkCfg := r.resolveNetwork(networkName)

	d, err := deployer.NewDeployer(r.verbose)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create deployer: %v", err)
		return
	}

	wasmPath := r.resolveVar(getStringConfig(step.Config, "wasm_path", ""))
	// Resolve glob patterns in wasm_path (e.g., "contract/target/.../release/*.wasm")
	if strings.ContainsAny(wasmPath, "*?[") {
		matches, err := filepath.Glob(wasmPath)
		if err == nil && len(matches) > 0 {
			wasmPath = matches[0]
		}
	}
	abiPath := r.resolveVar(getStringConfig(step.Config, "abi_path", "abi.json"))
	walletSeed := r.resolveVar(getStringConfig(step.Config, "wallet_seed", ""))

	deployResult, err := d.Deploy(ctx, deployer.DeploymentConfig{
		WasmPath:   wasmPath,
		ABIPath:    abiPath,
		NetworkURL: networkCfg.URL,
		NetworkID:  networkCfg.NetworkID,
		WalletSeed: walletSeed,
		Algorithm:  getStringConfig(step.Config, "algorithm", "secp256k1"),
		FaucetURL:  networkCfg.FaucetURL,
	})

	if err != nil {
		result.Error = fmt.Sprintf("deployment failed: %v", err)
		return
	}

	result.Success = true
	result.Output = map[string]string{
		"contract_account": deployResult.ContractAccount,
		"wallet_address":   deployResult.WalletAddress,
		"wallet_seed":      deployResult.WalletSeed,
		"tx_hash":          deployResult.TxHash,
	}
}

func (r *Runner) executeCall(ctx context.Context, step Step, networkName string, result *StepResult) {
	networkCfg := r.resolveNetwork(networkName)

	c, err := caller.NewCaller(r.verbose)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create caller: %v", err)
		return
	}

	contractAccount := r.resolveVar(getStringConfig(step.Config, "contract_account", ""))
	functionName := r.resolveVar(getStringConfig(step.Config, "function", ""))
	walletSeed := r.resolveVar(getStringConfig(step.Config, "wallet_seed", ""))

	params, _ := step.Config["parameters"].(map[string]interface{})

	callResult, err := c.Call(ctx, caller.CallConfig{
		ContractAccount:      contractAccount,
		FunctionName:         functionName,
		NetworkURL:           networkCfg.URL,
		NetworkID:            networkCfg.NetworkID,
		WalletSeed:           walletSeed,
		Algorithm:            getStringConfig(step.Config, "algorithm", "secp256k1"),
		ABIPath:              getStringConfig(step.Config, "abi_path", "abi.json"),
		Parameters:           params,
		ComputationAllowance: getStringConfig(step.Config, "gas", "1000000"),
		Fee:                  getStringConfig(step.Config, "fee", "1000000"),
	})

	if err != nil {
		result.Error = fmt.Sprintf("call failed: %v", err)
		return
	}

	result.Success = true
	result.Output = map[string]string{
		"tx_hash":      callResult.TxHash,
		"return_code":  fmt.Sprintf("%d", callResult.ReturnCode),
		"return_value": callResult.ReturnValue,
		"gas_used":     fmt.Sprintf("%d", callResult.GasUsed),
	}
}

func (r *Runner) executeFund(ctx context.Context, step Step, networkName string, result *StepResult) {
	networkCfg := r.resolveNetwork(networkName)

	f, err := faucet.NewFaucet(r.verbose)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create faucet: %v", err)
		return
	}

	isLocal := networkName == "local"
	walletSeed := r.resolveVar(getStringConfig(step.Config, "wallet_seed", ""))

	faucetResult, err := f.Request(ctx, faucet.FaucetConfig{
		FaucetURL:  networkCfg.FaucetURL,
		WalletSeed: walletSeed,
		Algorithm:  getStringConfig(step.Config, "algorithm", "secp256k1"),
		NetworkURL: networkCfg.URL,
		IsLocal:    isLocal,
	})

	if err != nil {
		result.Error = fmt.Sprintf("faucet request failed: %v", err)
		return
	}

	result.Success = true
	result.Output = map[string]string{
		"wallet_address": faucetResult.WalletAddress,
		"wallet_seed":    faucetResult.WalletSeed,
		"balance":        faucetResult.Balance,
	}
}

func (r *Runner) executeWait(step Step, result *StepResult) {
	seconds := getIntConfig(step.Config, "seconds", 5)
	time.Sleep(time.Duration(seconds) * time.Second)
	result.Success = true
}

func (r *Runner) executeAssert(step Step, result *StepResult) {
	for _, assertion := range step.Assertions {
		actual := r.resolveVar(assertion.Field)
		expected := r.resolveVar(assertion.Value)

		passed := false
		switch assertion.Operator {
		case "eq", "==", "equals":
			passed = actual == expected
		case "neq", "!=", "not_equals":
			passed = actual != expected
		case "contains":
			passed = strings.Contains(actual, expected)
		default:
			result.Error = fmt.Sprintf("unknown operator: %s", assertion.Operator)
			return
		}

		if !passed {
			result.Error = fmt.Sprintf("assertion failed: %s %s %s (got: %s)", assertion.Field, assertion.Operator, expected, actual)
			return
		}
	}

	result.Success = true
}

func (r *Runner) resolveVar(s string) string {
	// Replace ${var.field} patterns with variable values
	for k, v := range r.variables {
		s = strings.ReplaceAll(s, "${"+k+"}", v)
	}
	return s
}

func (r *Runner) resolveNetwork(name string) config.NetworkConfig {
	if cfg, ok := r.cfg.Networks[name]; ok {
		return cfg
	}
	if name == "local" {
		return config.NetworkConfig{
			URL:       "ws://localhost:6006",
			NetworkID: 100,
		}
	}
	return config.NetworkConfig{}
}

func getStringConfig(cfg map[string]interface{}, key string, def string) string {
	if v, ok := cfg[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return def
}

func getIntConfig(cfg map[string]interface{}, key string, def int) int {
	if v, ok := cfg[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		}
	}
	return def
}
