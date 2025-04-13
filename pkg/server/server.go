package server

import (
	"log"
	"os"
	"path/filepath"

	"github.com/novacove/imbued/pkg/keychain"
)

type Server struct {
	mgr *keychain.KeychainManager
}

func NewServer() *Server {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get user home directory: %v", err)
	}

	keychainDir := filepath.Join(homeDir, "Library", "Application Support", "imbued")
	keychainPath := filepath.Join(keychainDir, "imbued.keychain")
	return &Server{
		mgr: keychain.NewKeychainManager(keychainPath),
	}
}

func (s *Server) ensureKeychainExists() {
	// Define paths for our custom keychain

	// Ensure the keychain exists, creating it if necessary
	if err := s.mgr.EnsureKeychainExists(""); err != nil {
		log.Fatalf("Error ensuring keychain exists: %v", err)
	}
	// Ensure it's in the search path
	if err := s.mgr.EnsureInSearchPath(); err != nil {
		log.Fatalf("Error adding keychain to search path: %v", err)
	}
}

func (s *Server) AddItem(service, account, label string, data []byte) error {
	return s.mgr.AddItem(service, account, label, data)
}
func (s *Server) GetItem(service, account string) ([]byte, error) {
	return s.mgr.GetItem(service, account)
}
func (s *Server) ListItems(service string) ([]string, error) {
	return s.mgr.ListItems(service)
}
