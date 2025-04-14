package secrets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	// KeychainServiceName is the name of the keychain service for 1Password credentials
	KeychainServiceName = "com.novacove.imbued.onepass"

	// KeychainAccountTokenKey is the key for the account token in the keychain
	KeychainAccountTokenKey = "account_token"

	// KeychainVaultIDKey is the key for the vault ID in the keychain
	KeychainVaultIDKey = "vault_id"
)

// OnePassBackend implements the Backend interface for 1Password
type OnePassBackend struct {
	accountToken string
	vaultID      string
	initialized  bool
}

// Initialize initializes the OnePassBackend with the given configuration
func (b *OnePassBackend) Initialize(config map[string]string) error {
	// Verify that the 1Password CLI is installed and working
	if err := b.verifyCliInstallation(); err != nil {
		return fmt.Errorf("1Password CLI verification failed: %w", err)
	}

	// Get credentials from keychain
	accountToken, err := GetKeychainItem(KeychainServiceName, KeychainAccountTokenKey)
	if err != nil {
		return fmt.Errorf("failed to get 1Password account token from keychain: %w", err)
	}

	vaultID, err := GetKeychainItem(KeychainServiceName, KeychainVaultIDKey)
	if err != nil {
		return fmt.Errorf("failed to get 1Password vault ID from keychain: %w", err)
	}

	b.accountToken = accountToken
	b.vaultID = vaultID
	b.initialized = true

	return nil
}

// GetKeychainItem retrieves an item from the macOS Keychain
func GetKeychainItem(service, account string) (string, error) {
	cmd := exec.Command("security", "find-generic-password", "-s", service, "-a", account, "-w")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), "could not be found") {
			return "", fmt.Errorf("keychain item not found: service=%s, account=%s", service, account)
		}
		return "", fmt.Errorf("failed to get keychain item: %w, stderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// verifyCliInstallation checks if the 1Password CLI is installed and working
func (b *OnePassBackend) verifyCliInstallation() error {
	cmd := exec.Command("op", "--version")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("1Password CLI not found or not working: %w, stderr: %s",
			err, stderr.String())
	}

	return nil
}

// GetSecret retrieves a secret by its key
func (b *OnePassBackend) GetSecret(key string) (string, error) {
	if !b.initialized {
		return "", fmt.Errorf("onepass backend not initialized")
	}

	// Set the 1Password account token as an environment variable
	cmd := exec.Command("op", "item", "get", key, "--vault", b.vaultID, "--format", "json")
	cmd.Env = append(os.Environ(), fmt.Sprintf("OP_SERVICE_ACCOUNT_TOKEN=%s", b.accountToken))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Check if the error is due to item not found
		if strings.Contains(stderr.String(), "not found") {
			return "", fmt.Errorf("secret not found: %s", key)
		}
		return "", fmt.Errorf("failed to get secret from 1Password: %w, stderr: %s",
			err, stderr.String())
	}

	// Parse the JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		return "", fmt.Errorf("failed to parse 1Password response: %w", err)
	}

	// Extract the secret value
	// The structure of the response depends on the item type
	// For a simple password item, we look for the password field
	fields, ok := response["fields"].([]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response format from 1Password")
	}

	// Look for the password field
	for _, field := range fields {
		fieldMap, ok := field.(map[string]interface{})
		if !ok {
			continue
		}

		// Check if this is the password field
		if label, ok := fieldMap["label"].(string); ok && label == "password" {
			if value, ok := fieldMap["value"].(string); ok {
				return value, nil
			}
		}
	}

	return "", fmt.Errorf("password field not found in 1Password item: %s", key)
}

func (b *OnePassBackend) StoreSecrets(secrets map[string]string) error {
	if !b.initialized {
		return fmt.Errorf("onepass backend not initialized")
	}

	for key, value := range secrets {
		// Check if the secret already exists
		_, err := b.GetSecret(key)
		if err == nil {
			return fmt.Errorf("secret with key %s already exists", key)
		}

		// Store the secret in 1Password
		cmd := exec.Command("op", "item", "create", "--vault", b.vaultID, "--category", "password", "--title", key, fmt.Sprintf("password=%s", value))
		cmd.Env = append(os.Environ(), fmt.Sprintf("OP_SERVICE_ACCOUNT_TOKEN=%s", b.accountToken))

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to store secret in 1Password: %w, stderr: %s", err, stderr.String())
		}
	}

	return fmt.Errorf("storing secrets in 1Password is not implemented")
}

// Close cleans up any resources used by the backend
func (b *OnePassBackend) Close() error {
	b.initialized = false
	return nil
}

// StoreOnePassCredentials stores 1Password credentials in the macOS Keychain
func StoreOnePassCredentials(accountToken, vaultID string) error {
	// Store account token
	if err := storeKeychainItem(KeychainServiceName, KeychainAccountTokenKey, accountToken); err != nil {
		return fmt.Errorf("failed to store account token in keychain: %w", err)
	}

	// Store vault ID
	if err := storeKeychainItem(KeychainServiceName, KeychainVaultIDKey, vaultID); err != nil {
		return fmt.Errorf("failed to store vault ID in keychain: %w", err)
	}

	return nil
}

// storeKeychainItem stores an item in the macOS Keychain
func storeKeychainItem(service, account, password string) error {
	// First, try to delete any existing item
	deleteCmd := exec.Command("security", "delete-generic-password", "-s", service, "-a", account)
	// Ignore errors from delete command, as the item might not exist
	_ = deleteCmd.Run()

	// Create the new item
	createCmd := exec.Command("security", "add-generic-password", "-s", service, "-a", account, "-w", password)
	var stderr bytes.Buffer
	createCmd.Stderr = &stderr

	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to add keychain item: %w, stderr: %s", err, stderr.String())
	}

	return nil
}
