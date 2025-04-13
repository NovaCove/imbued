package secrets

import (
	"fmt"
)

// Backend defines the interface for secret backends
type Backend interface {
	// Initialize initializes the backend with the given configuration
	Initialize(config map[string]string) error

	// GetSecret retrieves a secret by its key
	GetSecret(key string) (string, error)

	// Close cleans up any resources used by the backend
	Close() error
}

// BackendType represents the type of secret backend
type BackendType string

const (
	// EnvFile represents a simple .env file backend
	EnvFile BackendType = "env_file"

	// Vault represents HashiCorp Vault backend
	Vault BackendType = "vault"

	// OnePass represents 1Password backend
	OnePass BackendType = "onepass"

	// AWSSecretManager represents AWS Secret Manager backend
	AWSSecretManager BackendType = "aws_secret_manager"

	// GCPSecretManager represents GCP Secret Manager backend
	GCPSecretManager BackendType = "gcp_secret_manager"
)

// NewBackend creates a new secret backend based on the given type
func NewBackend(backendType string) (Backend, error) {
	switch BackendType(backendType) {
	case EnvFile:
		return &EnvFileBackend{}, nil
	case Vault:
		return &VaultBackend{}, nil
	case OnePass:
		return &OnePassBackend{}, nil
	case AWSSecretManager:
		return &AWSSecretManagerBackend{}, nil
	case GCPSecretManager:
		return &GCPSecretManagerBackend{}, nil
	default:
		return nil, fmt.Errorf("unsupported backend type: %s", backendType)
	}
}
