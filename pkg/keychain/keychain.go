package keychain

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/keybase/go-keychain"
)

// KeychainManager handles operations for a custom macOS keychain
type KeychainManager struct {
	Path     string
	Password string // Optional: password for unlocking
}

// NewKeychainManager creates a new manager for the specified keychain path
func NewKeychainManager(keychainPath string) *KeychainManager {
	return &KeychainManager{
		Path: keychainPath,
	}
}

// EnsureKeychainExists ensures the custom keychain exists, creating it if necessary
func (km *KeychainManager) EnsureKeychainExists(password string) error {
	// Check if keychain already exists
	if _, err := os.Stat(km.Path); err == nil {
		// Keychain exists
		return nil
	} else if !os.IsNotExist(err) {
		// Some other error occurred
		return fmt.Errorf("error checking keychain existence: %w", err)
	}

	// Create the directory if it doesn't exist
	dir := filepath.Dir(km.Path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory for keychain: %w", err)
	}

	// Create the keychain using security command
	// cmd := exec.Command("security", "create-keychain", "-p", password, km.Path)
	cmd := exec.Command("security", "create-keychain", km.Path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create keychain: %v, output: %s", err, output)
	}

	// Set keychain settings to prevent it from auto-locking
	cmd = exec.Command("security", "set-keychain-settings", "-u", km.Path)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set keychain settings: %v, output: %s", err, output)
	}

	// Store the password for future use
	km.Password = password
	return nil
}

// EnsureInSearchPath makes sure the keychain is in the search path
func (km *KeychainManager) EnsureInSearchPath() error {
	// Get current search list
	cmd := exec.Command("security", "list-keychains", "-d", "user")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get keychain search list: %v, output: %s", err, output)
	}

	// Check if our keychain is already in the list
	if strings.Contains(string(output), km.Path) {
		return nil
	}

	// Parse the current keychains - they're returned as quoted strings
	keychains := string(output)
	keychains = strings.TrimSpace(keychains)

	// Prepare command to update the search list
	// We need to keep existing keychains and add ours
	cmd = exec.Command("security", "list-keychains", "-d", "user", "-s", km.Path, keychains)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update keychain search list: %v, output: %s", err, output)
	}

	return nil
}

// UnlockKeychain unlocks the keychain using the stored password
func (km *KeychainManager) UnlockKeychain() error {
	if km.Password == "" {
		return fmt.Errorf("no password provided to unlock keychain")
	}

	// cmd := exec.Command("security", "unlock-keychain", "-p", km.Password, km.Path)
	cmd := exec.Command("security", "unlock-keychain", km.Path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to unlock keychain: %v, output: %s", err, output)
	}

	return nil
}

// CheckIfUnlocked checks if the keychain is currently unlocked
func (km *KeychainManager) CheckIfUnlocked() (bool, error) {
	// This is a bit tricky - we'll try to use the show-keychain-info command
	// and see if it mentions "The specified keychain is locked."
	cmd := exec.Command("security", "show-keychain-info", km.Path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check if the error indicates the keychain is locked
		if strings.Contains(stderr.String(), "The specified keychain is locked.") {
			return false, nil
		}
		return false, fmt.Errorf("error checking keychain status: %v, stderr: %s", err, stderr.String())
	}

	return true, nil
}

// AddItem adds an item to the custom keychain
func (km *KeychainManager) AddItem(service, account, label string, data []byte) error {
	// Ensure keychain is in search path
	if err := km.EnsureInSearchPath(); err != nil {
		return err
	}

	// Check if keychain is unlocked
	unlocked, err := km.CheckIfUnlocked()
	if err != nil {
		return err
	}

	// Try to unlock if necessary
	if !unlocked && km.Password != "" {
		if err := km.UnlockKeychain(); err != nil {
			return err
		}
	}

	// Create a new keychain item
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(service)
	item.SetAccount(account)
	item.SetLabel(label)
	item.SetData(data)
	item.SetAccessible(keychain.AccessibleAfterFirstUnlock)

	// We don't set UseKeychain because the Go keychain library doesn't
	// actually allow specifying which keychain to use directly.
	// Instead, we rely on having added our keychain to the search list.

	// Add the item
	err = keychain.AddItem(item)
	if err == keychain.ErrorDuplicateItem {
		// Item already exists, try to update it
		query := keychain.NewItem()
		query.SetSecClass(keychain.SecClassGenericPassword)
		query.SetService(service)
		query.SetAccount(account)

		update := keychain.NewItem()
		update.SetData(data)

		err = keychain.UpdateItem(query, update)
	}

	return err
}

// GetItem retrieves an item from the custom keychain
func (km *KeychainManager) GetItem(service, account string) ([]byte, error) {
	// Ensure keychain is in search path
	if err := km.EnsureInSearchPath(); err != nil {
		return nil, err
	}

	// Check if keychain is unlocked
	unlocked, err := km.CheckIfUnlocked()
	if err != nil {
		return nil, err
	}

	// Try to unlock if necessary
	if !unlocked && km.Password != "" {
		if err := km.UnlockKeychain(); err != nil {
			return nil, err
		}
	}

	// Create a query for the item
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(service)
	query.SetAccount(account)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(true)

	// We don't set UseKeychain as explained in AddItem

	// Query for the item
	results, err := keychain.QueryItem(query)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no item found for service=%s account=%s", service, account)
	}

	return results[0].Data, nil
}

// ListItems lists all items in the keychain for a specific service
func (km *KeychainManager) ListItems(service string) ([]string, error) {
	// Ensure keychain is in search path
	if err := km.EnsureInSearchPath(); err != nil {
		return nil, err
	}

	// Create a query for all items with the given service
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	if service != "" {
		query.SetService(service)
	}
	query.SetMatchLimit(keychain.MatchLimitAll)
	query.SetReturnAttributes(true)

	// We don't set UseKeychain as explained in AddItem

	// Query for items
	results, err := keychain.QueryItem(query)
	if err != nil {
		return nil, err
	}

	// Collect account names
	accounts := make([]string, len(results))
	for i, item := range results {
		accounts[i] = item.Account
	}

	return accounts, nil
}

// // Example usage
// func Example() {
// 	// Create a manager for our custom keychain
// 	homeDir, _ := os.UserHomeDir()
// 	keychainPath := filepath.Join(homeDir, "Library", "Keychains", "MyAppKeychain.keychain-db")
// 	manager := NewKeychainManager(keychainPath)

// 	// Ensure it exists, creating it if necessary with the given password
// 	if err := manager.EnsureKeychainExists("keychainpassword"); err != nil {
// 		fmt.Printf("Error ensuring keychain exists: %v\n", err)
// 		return
// 	}

// 	// Ensure it's in the search path
// 	if err := manager.EnsureInSearchPath(); err != nil {
// 		fmt.Printf("Error adding keychain to search path: %v\n", err)
// 		return
// 	}

// 	// Add an item
// 	if err := manager.AddItem("MyService", "user1", "My Service Credentials", []byte("secretpassword")); err != nil {
// 		fmt.Printf("Error adding item: %v\n", err)
// 		return
// 	}

// 	// Retrieve the item
// 	data, err := manager.GetItem("MyService", "user1")
// 	if err != nil {
// 		fmt.Printf("Error getting item: %v\n", err)
// 		return
// 	}

// 	fmt.Printf("Retrieved password: %s\n", string(data))

// 	// List all items for the service
// 	accounts, err := manager.ListItems("MyService")
// 	if err != nil {
// 		fmt.Printf("Error listing items: %v\n", err)
// 		return
// 	}

// 	fmt.Println("Accounts found:")
// 	for _, account := range accounts {
// 		fmt.Println("- " + account)
// 	}
// }
