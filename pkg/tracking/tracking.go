package tracking

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AccessType represents the type of access being tracked
type AccessType string

const (
	// AuthenticationRequest represents an authentication request
	AuthenticationRequest AccessType = "authentication_request"
	// AuthenticationSuccess represents a successful authentication
	AuthenticationSuccess AccessType = "authentication_success"
	// AuthenticationFailure represents a failed authentication
	AuthenticationFailure AccessType = "authentication_failure"
	// SecretAccess represents a secret access
	SecretAccess AccessType = "secret_access"
	// SecretAccessFailure represents a failed secret access
	SecretAccessFailure AccessType = "secret_access_failure"
)

// AccessRecord represents a record of an access or authentication event
type AccessRecord struct {
	Timestamp   time.Time  `json:"timestamp"`
	Type        AccessType `json:"type"`
	ProcessID   string     `json:"process_id"`
	SecretNames []string   `json:"secret_names,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// Tracker defines the interface for tracking access and authentication events
type Tracker interface {
	// TrackAuthenticationRequest tracks an authentication request
	TrackAuthenticationRequest(processID string, secretNames []string) error
	// TrackAuthenticationSuccess tracks a successful authentication
	TrackAuthenticationSuccess(processID string, secretNames []string) error
	// TrackAuthenticationFailure tracks a failed authentication
	TrackAuthenticationFailure(processID string, secretNames []string, err error) error
	// TrackSecretAccess tracks a secret access
	TrackSecretAccess(processID string, secretNames []string) error
	// TrackSecretAccessFailure tracks a failed secret access
	TrackSecretAccessFailure(processID string, secretNames []string, err error) error
	// Close closes the tracker
	Close() error
}

// FileTracker is a file-based implementation of the Tracker interface
type FileTracker struct {
	logFile *os.File
	mu      sync.Mutex
}

// NewFileTracker creates a new FileTracker
func NewFileTracker(logFilePath string) (*FileTracker, error) {
	// Create the directory if it doesn't exist
	logDir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open the log file for appending
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &FileTracker{
		logFile: logFile,
	}, nil
}

// TrackAuthenticationRequest tracks an authentication request
func (t *FileTracker) TrackAuthenticationRequest(processID string, secretNames []string) error {
	return t.trackEvent(AccessRecord{
		Timestamp:   time.Now(),
		Type:        AuthenticationRequest,
		ProcessID:   processID,
		SecretNames: secretNames,
	})
}

// TrackAuthenticationSuccess tracks a successful authentication
func (t *FileTracker) TrackAuthenticationSuccess(processID string, secretNames []string) error {
	return t.trackEvent(AccessRecord{
		Timestamp:   time.Now(),
		Type:        AuthenticationSuccess,
		ProcessID:   processID,
		SecretNames: secretNames,
	})
}

// TrackAuthenticationFailure tracks a failed authentication
func (t *FileTracker) TrackAuthenticationFailure(processID string, secretNames []string, err error) error {
	return t.trackEvent(AccessRecord{
		Timestamp:   time.Now(),
		Type:        AuthenticationFailure,
		ProcessID:   processID,
		SecretNames: secretNames,
		Error:       err.Error(),
	})
}

// TrackSecretAccess tracks a secret access
func (t *FileTracker) TrackSecretAccess(processID string, secretNames []string) error {
	return t.trackEvent(AccessRecord{
		Timestamp:   time.Now(),
		Type:        SecretAccess,
		ProcessID:   processID,
		SecretNames: secretNames,
	})
}

// TrackSecretAccessFailure tracks a failed secret access
func (t *FileTracker) TrackSecretAccessFailure(processID string, secretNames []string, err error) error {
	return t.trackEvent(AccessRecord{
		Timestamp:   time.Now(),
		Type:        SecretAccessFailure,
		ProcessID:   processID,
		SecretNames: secretNames,
		Error:       err.Error(),
	})
}

// trackEvent writes an event to the log file
func (t *FileTracker) trackEvent(record AccessRecord) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Marshal the record to JSON
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	// Write the record to the log file
	if _, err := t.logFile.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	return nil
}

// Close closes the tracker
func (t *FileTracker) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.logFile.Close()
}
