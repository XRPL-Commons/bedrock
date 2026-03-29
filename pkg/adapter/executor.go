package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/xrpl-commons/bedrock/embedded"
)

// Executor handles execution of embedded JavaScript modules
type Executor struct {
	modulesDir string
	group      string
	verbose    bool
}

// NewExecutor creates a new executor for the contract module group (backward compat)
func NewExecutor(verbose bool) (*Executor, error) {
	return NewGroupExecutor("contract", verbose)
}

// NewGroupExecutor creates a new executor for a specific module group
func NewGroupExecutor(group string, verbose bool) (*Executor, error) {
	dir, err := embedded.SetupModuleGroup(group)
	if err != nil {
		return nil, fmt.Errorf("failed to setup %s modules: %w", group, err)
	}

	return &Executor{
		modulesDir: dir,
		group:      group,
		verbose:    verbose,
	}, nil
}

// ExecuteModule runs a JavaScript module with JSON config input/output
func (e *Executor) ExecuteModule(ctx context.Context, moduleName string, config interface{}) (*Result, error) {
	// Get module path
	modulePath := filepath.Join(e.modulesDir, moduleName)
	if _, err := os.Stat(modulePath); err != nil {
		return nil, fmt.Errorf("module %s not found: %w", moduleName, err)
	}

	// Create temp config file
	configFile, err := e.writeConfigFile(config)
	if err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}
	defer os.Remove(configFile)

	// Verify config file exists before running
	if _, err := os.Stat(configFile); err != nil {
		return nil, fmt.Errorf("config file disappeared: %w", err)
	}

	// Execute Node.js module
	cmd := exec.CommandContext(ctx, "node", modulePath, configFile)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout

	// In verbose mode, show stderr in real-time. Otherwise capture it.
	if e.verbose {
		cmd.Stderr = os.Stderr
		fmt.Printf("[executor] Running: node %s %s\n", modulePath, configFile)
		fmt.Printf("[executor] Config file exists: %v\n", configFile)
	} else {
		cmd.Stderr = &stderr
	}

	start := time.Now()
	err = cmd.Run()
	duration := time.Since(start)

	if e.verbose {
		fmt.Printf("[executor] Execution took %v\n", duration)
	}

	// Check for execution errors
	if err != nil {
		stderrMsg := stderr.String()
		stdoutMsg := stdout.String()
		if stderrMsg == "" {
			stderrMsg = "(stderr empty - verbose mode was on, check output above)"
		}
		return nil, fmt.Errorf("module execution failed: %w\nStdout: %s\nStderr: %s", err, stdoutMsg, stderrMsg)
	}

	// Parse JSON result from stdout
	result, err := e.parseResult(stdout.Bytes())
	if err != nil {
		// Show both stdout and stderr on parse errors
		errMsg := fmt.Sprintf("failed to parse result: %v\nStdout: %s", err, stdout.String())
		if stderr.Len() > 0 {
			errMsg += fmt.Sprintf("\nStderr: %s", stderr.String())
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	return result, nil
}

// writeConfigFile writes the config object as JSON to a temp file
func (e *Executor) writeConfigFile(config interface{}) (string, error) {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "bedrock-config-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write config: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	if e.verbose {
		fmt.Printf("[executor] Config file: %s\n", tmpFile.Name())
		fmt.Printf("[executor] Config data:\n%s\n", string(data))
	}

	return tmpFile.Name(), nil
}

// parseResult parses the JSON result from module output
func (e *Executor) parseResult(data []byte) (*Result, error) {
	var result Result
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if !result.Success {
		return &result, fmt.Errorf("module returned error: %s - %s", result.Error, result.Details)
	}

	return &result, nil
}

// Result represents the standardized result from a JS module
type Result struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
	Details string          `json:"details,omitempty"`
}
