package secrets

import (
	"github.com/keybase/go-keychain"
)

type MacOSKeychainBackend struct{}

func NewMacOSKeychainBackend() *MacOSKeychainBackend {
	return &MacOSKeychainBackend{}
}

// Get retrieves a secret from the macOS keychain.
func (b *MacOSKeychainBackend) Get(service, account string) (string, error) {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(service)
	query.SetAccount(account)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(true)

	results, err := keychain.QueryItem(query)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", nil // No matching item found
	}

	return string(results[0].Data), nil
}

// Set stores a secret in the macOS keychain.
func (b *MacOSKeychainBackend) Set(service, account, secret string) error {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(service)
	item.SetAccount(account)
	item.SetData([]byte(secret))
	item.SetAccessible(keychain.AccessibleWhenUnlocked)

	// Add or update the item in the keychain
	err := keychain.AddItem(item)
	if err == keychain.ErrorDuplicateItem {
		// Update the existing item
		return keychain.UpdateItem(item, item)
	}
	return err
}

// Delete removes a secret from the macOS keychain.
func (b *MacOSKeychainBackend) Delete(service, account string) error {
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(service)
	item.SetAccount(account)

	return keychain.DeleteItem(item)
}

func (b *MacOSKeychainBackend) Initialize(config map[string]string) error {
	// No initialization needed for macOS Keychain
	return nil
}

// GetSecret retrieves a secret by its key
func (b *MacOSKeychainBackend) GetSecret(key string) (string, error) {
	if key == "" {
		return "", nil
	}
	secret, err := b.Get("imbued", key)
	if err != nil {
		return "", err
	}
	return secret, nil
}

func (b *MacOSKeychainBackend) StoreSecrets(secrets map[string]string) error {
	for key, value := range secrets {
		if err := b.Set("imbued", key, value); err != nil {
			return err
		}
	}
	return nil
}

// Close cleans up any resources used by the backend
func (b *MacOSKeychainBackend) Close() error {
	// No cleanup needed for macOS Keychain
	return nil
}
