package auth

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Authenticator defines the interface for authentication providers
type Authenticator interface {
	// Authenticate checks if the current process is authorized to access the given secrets
	Authenticate(processID string, secretNames []string) (bool, error)

	// IsAuthenticated checks if the current process is already authenticated
	IsAuthenticated(processID string) bool

	// RecordAccess records a successful access to secrets
	RecordAccess(processID string, secretNames []string) error
}

// SimpleAuthenticator is a basic implementation of the Authenticator interface
type SimpleAuthenticator struct {
	// Map of process ID to authentication expiry time
	authenticatedProcesses map[string]time.Time
	// Authentication duration
	authDuration time.Duration
}

// NewSimpleAuthenticator creates a new SimpleAuthenticator
func NewSimpleAuthenticator(authDuration time.Duration) *SimpleAuthenticator {
	return &SimpleAuthenticator{
		authenticatedProcesses: make(map[string]time.Time),
		authDuration:           authDuration,
	}
}

func promptTouchID() bool {
	// Using the security command-line tool to prompt for TouchID
	cmd := exec.Command("security", "authorize", "-u", "-p", "imbued would like to use TouchID authentication to hydrate your environment")
	err := cmd.Run()
	return err == nil
}

// Authenticate checks if the current process is authorized to access the given secrets
func (a *SimpleAuthenticator) Authenticate(processID string, secretNames []string) (bool, error) {
	// Check if already authenticated
	if a.IsAuthenticated(processID) {
		return true, nil
	}

	if !promptTouchID() {
		// If TouchID authentication is successful, record the authentication
		fmt.Println("TouchID authentication failed")
		return false, nil
	}

	// In a real implementation, this would prompt the user for authentication
	// For now, we'll just print a message and assume authentication is successful
	fmt.Printf("Process %s is requesting access to secrets: %v\n", processID, secretNames)
	fmt.Println("Authentication successful")

	// Record authentication
	a.authenticatedProcesses[processID] = time.Now().Add(a.authDuration)

	return true, nil
}

// IsAuthenticated checks if the current process is already authenticated
func (a *SimpleAuthenticator) IsAuthenticated(processID string) bool {
	expiryTime, ok := a.authenticatedProcesses[processID]
	if !ok {
		return false
	}

	// Check if authentication has expired
	if time.Now().After(expiryTime) {
		delete(a.authenticatedProcesses, processID)
		return false
	}

	return true
}

// RecordAccess records a successful access to secrets
func (a *SimpleAuthenticator) RecordAccess(processID string, secretNames []string) error {
	// In a real implementation, this would record the access in a log file or database
	// For now, we'll just print a message
	fmt.Printf("Process %s accessed secrets: %v\n", processID, secretNames)
	return nil
}

// GetCurrentProcessID returns the current process ID
func GetCurrentProcessID() string {
	return fmt.Sprintf("%d", os.Getpid())
}

func GetParentProcessID() string {
	parentProc, err := os.FindProcess(os.Getppid())
	if err != nil {
		fmt.Println("Error finding parent process:", err)
		return ""
	}

	grandparentPID := parentProc.Pid
	return fmt.Sprintf("%d", grandparentPID)
}

func GetGroupProcessID() string {
	return fmt.Sprintf("%d", os.Getgid())
}
