package secrets

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// EnvFileBackend implements the Backend interface for .env files
type EnvFileBackend struct {
	filePath string
	secrets  map[string]string
}

// Initialize initializes the EnvFileBackend with the given configuration
func (b *EnvFileBackend) Initialize(config map[string]string) error {
	filePath, ok := config["file_path"]
	if !ok {
		return fmt.Errorf("file_path is required for env_file backend")
	}

	b.filePath = filePath
	b.secrets = make(map[string]string)

	// Load secrets from the file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) > 1 && (value[0] == '"' || value[0] == '\'') && value[0] == value[len(value)-1] {
			value = value[1 : len(value)-1]
		}

		b.secrets[key] = value
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read env file: %w", err)
	}

	return nil
}

// GetSecret retrieves a secret by its key
func (b *EnvFileBackend) GetSecret(key string) (string, error) {
	value, ok := b.secrets[key]
	if !ok {
		return "", fmt.Errorf("secret not found: %s", key)
	}
	return value, nil
}

// Close cleans up any resources used by the backend
func (b *EnvFileBackend) Close() error {
	// Nothing to clean up for EnvFileBackend
	return nil
}
