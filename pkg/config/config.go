package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// ImbuedConfig represents the parsed configuration from a .imbued file
type ImbuedConfig struct {
	Secrets       map[string]string // Map of secret name to environment variable name
	ValidDepth    int               // Number of child directories down that secrets are available for
	BackendType   string            // Type of secret backend to use
	BackendConfig map[string]string // Backend-specific configuration
}

// FindConfig looks for a .imbued file in the current directory or parent directories
// up to maxLevels levels up. Returns the path to the file if found, or an empty string if not.
func FindConfig(startDir string, maxLevels int) (string, error) {
	currentDir, err := filepath.Abs(startDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	for i := 0; i <= maxLevels; i++ {
		configPath := filepath.Join(currentDir, ".imbued")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		// Move up one directory
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// We've reached the root directory
			break
		}
		currentDir = parentDir
	}

	return "", fmt.Errorf("no .imbued file found within %d levels up from %s", maxLevels, startDir)
}

// LoadConfig loads and parses the .imbued file at the given path
func LoadConfig(configPath string) (*ImbuedConfig, error) {
	var imbuedConfigFile struct {
		Secrets       map[string]string `toml:"secrets"`
		ValidDepth    int               `toml:"valid_depth"`
		BackendType   string            `toml:"backend_type"`
		BackendConfig map[string]string `toml:"backend_config"`
	}

	_, err := toml.DecodeFile(configPath, &imbuedConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	config := &ImbuedConfig{
		Secrets:       imbuedConfigFile.Secrets,
		ValidDepth:    imbuedConfigFile.ValidDepth,
		BackendType:   imbuedConfigFile.BackendType,
		BackendConfig: imbuedConfigFile.BackendConfig,
	}

	// Set default valid depth if not specified
	if config.ValidDepth <= 0 {
		config.ValidDepth = 1
	}

	return config, nil
}

// IsWithinValidDepth checks if the current directory is within the valid depth
// from the directory containing the .imbued file
func IsWithinValidDepth(configDir, currentDir string, validDepth int) (bool, error) {
	// Get absolute paths
	absConfigDir, err := filepath.Abs(configDir)
	if err != nil {
		return false, fmt.Errorf("failed to get absolute path for config dir: %w", err)
	}

	absCurrentDir, err := filepath.Abs(currentDir)
	if err != nil {
		return false, fmt.Errorf("failed to get absolute path for current dir: %w", err)
	}

	// Check if current directory is the same as or a subdirectory of the config directory
	if !strings.HasPrefix(absCurrentDir, absConfigDir) {
		return false, nil
	}

	// Calculate the depth difference
	relPath, err := filepath.Rel(absConfigDir, absCurrentDir)
	if err != nil {
		return false, fmt.Errorf("failed to get relative path: %w", err)
	}

	if relPath == "." {
		// Current directory is the same as config directory
		return true, nil
	}

	// Count the number of directory separators to determine depth
	depth := strings.Count(relPath, string(os.PathSeparator)) + 1
	return depth <= validDepth, nil
}

// GetConfigDir returns the directory containing the .imbued file
func GetConfigDir(configPath string) string {
	return filepath.Dir(configPath)
}
