package embedded

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

//go:embed modules/deploy/deploy.js modules/call/call.js modules/faucet/faucet.js modules/modify/modify.js modules/delete/delete.js modules/user_delete/user_delete.js modules/clawback/clawback.js modules/package.json modules/escrow/escrow_deploy.js modules/escrow/escrow_finish.js modules/escrow/escrow_cancel.js modules/escrow/escrow_status.js modules/escrow/package.json modules/vault/vault_deploy.js modules/vault/vault_deposit.js modules/vault/vault_withdraw.js modules/vault/vault_status.js modules/vault/package.json
var ModulesFS embed.FS

var (
	setupMu   sync.Mutex
	setupDone map[string]string // group -> cacheDir
)

const (
	versionFile = ".bedrock-version"
)

// moduleFile maps an embedded path to a cache filename
type moduleFile struct {
	EmbedPath string
	CacheName string
}

// contractModules are the core contract modules (backward compat)
var contractModules = []moduleFile{
	{"modules/package.json", "package.json"},
	{"modules/deploy/deploy.js", "deploy.js"},
	{"modules/call/call.js", "call.js"},
	{"modules/faucet/faucet.js", "faucet.js"},
	{"modules/modify/modify.js", "modify.js"},
	{"modules/delete/delete.js", "delete.js"},
	{"modules/user_delete/user_delete.js", "user_delete.js"},
	{"modules/clawback/clawback.js", "clawback.js"},
}

var escrowModules = []moduleFile{
	{"modules/escrow/package.json", "package.json"},
	{"modules/escrow/escrow_deploy.js", "escrow_deploy.js"},
	{"modules/escrow/escrow_finish.js", "escrow_finish.js"},
	{"modules/escrow/escrow_cancel.js", "escrow_cancel.js"},
	{"modules/escrow/escrow_status.js", "escrow_status.js"},
}

var vaultModules = []moduleFile{
	{"modules/vault/package.json", "package.json"},
	{"modules/vault/vault_deploy.js", "vault_deploy.js"},
	{"modules/vault/vault_deposit.js", "vault_deposit.js"},
	{"modules/vault/vault_withdraw.js", "vault_withdraw.js"},
	{"modules/vault/vault_status.js", "vault_status.js"},
}

// moduleGroups maps group name to its module list
var moduleGroups = map[string][]moduleFile{
	"contract": contractModules,
	"escrow":   escrowModules,
	"vault":    vaultModules,
}

func init() {
	setupDone = make(map[string]string)
}

// getCacheDir returns the XDG-compliant cache directory for bedrock
func getCacheDir() (string, error) {
	var baseDir string

	if runtime.GOOS == "windows" {
		baseDir = os.Getenv("LOCALAPPDATA")
		if baseDir == "" {
			baseDir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
	} else {
		baseDir = os.Getenv("XDG_CACHE_HOME")
		if baseDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to get home directory: %w", err)
			}
			baseDir = filepath.Join(home, ".cache")
		}
	}

	return filepath.Join(baseDir, "bedrock", "modules"), nil
}

// getGroupVersion returns a hash of the modules in a group
func getGroupVersion(modules []moduleFile) (string, error) {
	hasher := sha256.New()
	for _, m := range modules {
		data, err := ModulesFS.ReadFile(m.EmbedPath)
		if err != nil {
			return "", fmt.Errorf("failed to read %s: %w", m.EmbedPath, err)
		}
		hasher.Write(data)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// needsReinstall checks if a module group needs to be installed
func needsReinstall(cacheDir string, modules []moduleFile) (bool, error) {
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return true, nil
	}

	nodeModulesPath := filepath.Join(cacheDir, "node_modules")
	if _, err := os.Stat(nodeModulesPath); os.IsNotExist(err) {
		return true, nil
	}

	versionPath := filepath.Join(cacheDir, versionFile)
	cachedVersion, err := os.ReadFile(versionPath)
	if err != nil {
		return true, nil
	}

	currentVersion, err := getGroupVersion(modules)
	if err != nil {
		return false, err
	}

	return string(cachedVersion) != currentVersion, nil
}

// extractAndInstall extracts modules and runs npm install for a group
func extractAndInstall(cacheDir string, modules []moduleFile) error {
	// Clean and recreate
	if err := os.RemoveAll(cacheDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean cache: %w", err)
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache: %w", err)
	}

	// Extract all module files
	for _, m := range modules {
		data, err := ModulesFS.ReadFile(m.EmbedPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", m.EmbedPath, err)
		}
		outPath := filepath.Join(cacheDir, m.CacheName)
		// Modules are invoked via `node <path>`, never executed directly,
		// so they do not need the executable bit.
		if err := os.WriteFile(outPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", m.CacheName, err)
		}
	}

	// Install npm dependencies
	cmd := exec.Command("npm", "install", "--silent", "--no-progress")
	cmd.Dir = cacheDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install npm dependencies: %w", err)
	}

	// Write version file
	currentVersion, err := getGroupVersion(modules)
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}
	versionPath := filepath.Join(cacheDir, versionFile)
	if err := os.WriteFile(versionPath, []byte(currentVersion), 0644); err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	return nil
}

// SetupModuleGroup extracts and installs a specific module group.
// Returns the cache directory for that group.
func SetupModuleGroup(group string) (string, error) {
	setupMu.Lock()
	defer setupMu.Unlock()

	if dir, ok := setupDone[group]; ok {
		return dir, nil
	}

	modules, ok := moduleGroups[group]
	if !ok {
		return "", fmt.Errorf("unknown module group: %s", group)
	}

	baseCache, err := getCacheDir()
	if err != nil {
		return "", err
	}

	groupCache := filepath.Join(baseCache, group)

	reinstall, err := needsReinstall(groupCache, modules)
	if err != nil {
		return "", err
	}

	if reinstall {
		fmt.Printf("⚡ Setting up %s modules...\n", group)
		fmt.Printf("   Cache: %s\n", groupCache)

		if err := extractAndInstall(groupCache, modules); err != nil {
			return "", err
		}

		fmt.Printf("✓ %s modules ready\n", group)
	}

	setupDone[group] = groupCache
	return groupCache, nil
}

// SetupModules extracts and installs the contract modules (backward compat).
func SetupModules() (string, error) {
	return SetupModuleGroup("contract")
}

// GetModulePath returns the path to a module in a group.
func GetModulePath(moduleName string) (string, error) {
	return GetGroupModulePath("contract", moduleName)
}

// GetGroupModulePath returns the path to a module in a specific group.
func GetGroupModulePath(group, moduleName string) (string, error) {
	dir, err := SetupModuleGroup(group)
	if err != nil {
		return "", err
	}

	modulePath := filepath.Join(dir, moduleName)
	if _, err := os.Stat(modulePath); err != nil {
		return "", fmt.Errorf("module %s not found in %s group: %w", moduleName, group, err)
	}

	return modulePath, nil
}

// CleanCache removes the entire bedrock cache directory
func CleanCache() error {
	cache, err := getCacheDir()
	if err != nil {
		return fmt.Errorf("failed to get cache directory: %w", err)
	}

	if err := os.RemoveAll(cache); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache directory: %w", err)
	}

	return nil
}

// GetCacheDir returns the cache directory path (for display purposes)
func GetCacheDir() (string, error) {
	return getCacheDir()
}
