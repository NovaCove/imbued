package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/novacove/imbued/pkg/auth"
	"github.com/novacove/imbued/pkg/config"
	"github.com/novacove/imbued/pkg/secrets"
	"github.com/novacove/imbued/pkg/tracking"
	"github.com/spf13/cobra"
)

// Command represents a command sent from client to server
type Command struct {
	Action      string            `json:"action"`
	ConfigPath  string            `json:"config_path,omitempty"`
	SecretName  string            `json:"secret_name,omitempty"`
	ProcessID   string            `json:"process_id,omitempty"`
	MaxLevels   int               `json:"max_levels,omitempty"`
	CurrentDir  string            `json:"current_dir,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	BackendType string            `json:"backend_type,omitempty"`
}

// Response represents a response sent from server to client
type Response struct {
	Success bool              `json:"success"`
	Error   string            `json:"error,omitempty"`
	Data    map[string]string `json:"data,omitempty"`
	Output  string            `json:"output,omitempty"`
}

var (
	// Global flags
	configPath   string
	maxLevels    int
	logFile      string
	authDuration time.Duration
	socketPath   string
)

var (
	lgr *slog.Logger
)

func getLogger() *slog.Logger {
	if lgr == nil {
		logFile, err := os.OpenFile(filepath.Join(
			os.Getenv("HOME"),
			".imbued",
			"logs",
			"info.log",
		), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("failed to open log file: %v", err)
		}
		lgr = slog.New(slog.NewJSONHandler(logFile, nil))
	}
	return lgr
}

// getDefaultSocketPath returns the default path for the Unix socket
func getDefaultSocketPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %v", err)
	}

	socketDir := filepath.Join(homeDir, ".imbued")
	if err := os.MkdirAll(socketDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create socket directory: %v", err)
	}

	return filepath.Join(socketDir, "imbued.sock"), nil
}

// runServer starts the imbued server
func runServer(socketPath string, tracker tracking.Tracker, authenticator auth.Authenticator) error {
	// Remove socket if it already exists
	if err := os.RemoveAll(socketPath); err != nil {
		return fmt.Errorf("failed to remove existing socket: %v", err)
	}

	// Create Unix socket
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on socket: %v", err)
	}
	defer listener.Close()

	log.Printf("Server listening on socket: %s", socketPath)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go handleConnection(conn, tracker, authenticator)
	}
}

// handleConnection handles a client connection
func handleConnection(conn net.Conn, tracker tracking.Tracker, authenticator auth.Authenticator) {
	defer conn.Close()

	// Read command from client
	decoder := json.NewDecoder(conn)
	var cmd Command
	if err := decoder.Decode(&cmd); err != nil {
		log.Printf("Failed to decode command: %v", err)
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Failed to decode command: %v", err)})
		return
	}

	stringifiedCmd, err := json.MarshalIndent(cmd, "", "  ")
	if err != nil {
		log.Printf("Failed to stringify command: %v", err)
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Failed to stringify command: %v", err)})
		return
	}
	log.Default().Printf("Received command: %s\n", stringifiedCmd)

	// Process command
	switch cmd.Action {
	case "check_auth":
		handleCheckAuth(conn, cmd, authenticator)
	case "authenticate":
		handleAuthenticate(conn, cmd, tracker, authenticator)
	case "get_secret":
		handleGetSecret(conn, cmd, tracker, authenticator)
	case "list_secrets":
		handleListSecrets(conn, cmd)
	case "inject_env":
		handleInjectEnv(conn, cmd, tracker, authenticator)
	case "clean_env":
		handleCleanEnv(conn, cmd)
	case "show_config":
		handleShowConfig(conn, cmd)
	case "find_config":
		handleFindConfig(conn, cmd)
	case "store_secrets":
		handleStoreSecrets(conn, cmd, tracker, authenticator)
	default:
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Unknown action: %s", cmd.Action)})
	}
}

// handleCheckAuth handles the check_auth command
func handleCheckAuth(conn net.Conn, cmd Command, authenticator auth.Authenticator) {
	isAuthenticated := authenticator.IsAuthenticated(cmd.ProcessID)
	sendResponse(conn, Response{
		Success: true,
		Data: map[string]string{
			"authenticated": fmt.Sprintf("%t", isAuthenticated),
		},
	})
}

// handleAuthenticate handles the authenticate command
func handleAuthenticate(conn net.Conn, cmd Command, tracker tracking.Tracker, authenticator auth.Authenticator) {
	// Load config
	cfg, err := config.LoadConfig(cmd.ConfigPath)
	if err != nil {
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Failed to load config: %v", err)})
		return
	}

	// Get secret names
	secretNames := make([]string, 0, len(cfg.Secrets))
	for secretName := range cfg.Secrets {
		secretNames = append(secretNames, secretName)
	}

	// Track authentication request
	if err := tracker.TrackAuthenticationRequest(cmd.ProcessID, secretNames); err != nil {
		log.Printf("Failed to track authentication request: %v", err)
	}

	// Authenticate
	authenticated, err := authenticator.Authenticate(cmd.ProcessID, secretNames)
	if err != nil {
		if err := tracker.TrackAuthenticationFailure(cmd.ProcessID, secretNames, err); err != nil {
			log.Printf("Failed to track authentication failure: %v", err)
		}
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Authentication failed: %v", err)})
		return
	}

	if !authenticated {
		if err := tracker.TrackAuthenticationFailure(cmd.ProcessID, secretNames, fmt.Errorf("authentication denied")); err != nil {
			log.Printf("Failed to track authentication failure: %v", err)
		}
		sendResponse(conn, Response{Success: false, Error: "Authentication denied"})
		return
	}

	if err := tracker.TrackAuthenticationSuccess(cmd.ProcessID, secretNames); err != nil {
		log.Printf("Failed to track authentication success: %v", err)
	}

	sendResponse(conn, Response{Success: true, Output: "Authentication successful"})
}

// handleGetSecret handles the get_secret command
func handleGetSecret(conn net.Conn, cmd Command, tracker tracking.Tracker, authenticator auth.Authenticator) {
	// Load config
	cfg, err := config.LoadConfig(cmd.ConfigPath)
	if err != nil {
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Failed to load config: %v", err)})
		return
	}

	// Check if secret exists
	envName, ok := cfg.Secrets[cmd.SecretName]
	if !ok {
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Secret not found: %s", cmd.SecretName)})
		return
	}

	// Check if authenticated
	if !authenticator.IsAuthenticated(cmd.ProcessID) {
		sendResponse(conn, Response{Success: false, Error: "Process is not authenticated"})
		return
	}

	// Track secret access
	if err := tracker.TrackSecretAccess(cmd.ProcessID, []string{cmd.SecretName}); err != nil {
		log.Printf("Failed to track secret access: %v", err)
	}

	// Initialize secret backend
	backend, err := secrets.NewBackend(cfg.BackendType)
	if err != nil {
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Failed to create secret backend: %v", err)})
		return
	}

	if err := backend.Initialize(cfg.BackendConfig); err != nil {
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Failed to initialize secret backend: %v", err)})
		return
	}
	defer backend.Close()

	// Get the secret
	secretValue, err := backend.GetSecret(cmd.SecretName)
	if err != nil {
		if err := tracker.TrackSecretAccessFailure(cmd.ProcessID, []string{cmd.SecretName}, err); err != nil {
			log.Printf("Failed to track secret access failure: %v", err)
		}
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Failed to get secret: %v", err)})
		return
	}

	sendResponse(conn, Response{
		Success: true,
		Data: map[string]string{
			"env_name": envName,
			"value":    secretValue,
		},
	})
}

func handleStoreSecrets(conn net.Conn, cmd Command, tracker tracking.Tracker, authenticator auth.Authenticator) {
	// Load config
	log.Default().Printf("Loading config from %q", cmd.ConfigPath)
	cfg, err := config.LoadConfig(cmd.ConfigPath)
	if err != nil {
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Failed to load config: %v", err)})
		return
	}

	// Check if authenticated
	log.Default().Printf("Checking authentication for process ID %q", cmd.ProcessID)
	if !authenticator.IsAuthenticated(cmd.ProcessID) {
		sendResponse(conn, Response{Success: false, Error: "Process is not authenticated"})
		return
	}

	// Track secret access
	if err := tracker.TrackSecretAccess(cmd.ProcessID, []string{cmd.SecretName}); err != nil {
		log.Printf("Failed to track secret access: %v", err)
	}

	// Initialize secret backend
	backend, err := secrets.NewBackend(cfg.BackendType)
	if err != nil {
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Failed to create secret backend: %v", err)})
		return
	}

	if err := backend.Initialize(cfg.BackendConfig); err != nil {
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Failed to initialize secret backend: %v", err)})
		return
	}
	defer backend.Close()

	// Store the secrets
	err = backend.StoreSecrets(cmd.Environment)
	if err != nil {
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Failed to store secrets: %v", err)})
		return
	}

	sendResponse(conn, Response{Success: true})
}

// handleListSecrets handles the list_secrets command
func handleListSecrets(conn net.Conn, cmd Command) {
	// Load config
	cfg, err := config.LoadConfig(cmd.ConfigPath)
	if err != nil {
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Failed to load config: %v", err)})
		return
	}

	// Build response
	data := make(map[string]string)
	for secretName, envName := range cfg.Secrets {
		data[secretName] = envName
	}

	sendResponse(conn, Response{Success: true, Data: data})
}

// handleInjectEnv handles the inject_env command
func handleInjectEnv(conn net.Conn, cmd Command, tracker tracking.Tracker, authenticator auth.Authenticator) {
	// Load config
	cfg, err := config.LoadConfig(cmd.ConfigPath)
	if err != nil {
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Failed to load config: %v", err)})
		return
	}

	// Check if authenticated
	if !authenticator.IsAuthenticated(cmd.ProcessID) {
		sendResponse(conn, Response{Success: false, Error: "Process is not authenticated"})
		return
	}

	// Get secret names
	secretNames := make([]string, 0, len(cfg.Secrets))
	for secretName := range cfg.Secrets {
		secretNames = append(secretNames, secretName)
	}

	// Track secret access
	if err := tracker.TrackSecretAccess(cmd.ProcessID, secretNames); err != nil {
		log.Printf("Failed to track secret access: %v", err)
	}

	// Initialize secret backend
	backend, err := secrets.NewBackend(cfg.BackendType)
	if err != nil {
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Failed to create secret backend: %v", err)})
		return
	}

	if err := backend.Initialize(cfg.BackendConfig); err != nil {
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Failed to initialize secret backend: %v", err)})
		return
	}
	defer backend.Close()

	// Get and set each secret
	data := make(map[string]string)
	for secretName, envName := range cfg.Secrets {
		secretValue, err := backend.GetSecret(secretName)
		if err != nil {
			if err := tracker.TrackSecretAccessFailure(cmd.ProcessID, []string{secretName}, err); err != nil {
				log.Printf("Failed to track secret access failure: %v", err)
			}
			log.Printf("Failed to get secret %s: %v", secretName, err)
			continue
		}

		data[envName] = secretValue
	}

	sendResponse(conn, Response{Success: true, Data: data})
}

// handleCleanEnv handles the clean_env command
func handleCleanEnv(conn net.Conn, cmd Command) {
	// Load config
	cfg, err := config.LoadConfig(cmd.ConfigPath)
	if err != nil {
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Failed to load config: %v", err)})
		return
	}

	// Build response
	data := make(map[string]string)
	for _, envName := range cfg.Secrets {
		data[envName] = ""
	}

	sendResponse(conn, Response{Success: true, Data: data})
}

// handleShowConfig handles the show_config command
func handleShowConfig(conn net.Conn, cmd Command) {
	// Load config
	cfg, err := config.LoadConfig(cmd.ConfigPath)
	if err != nil {
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Failed to load config: %v", err)})
		return
	}

	// Build response
	data := map[string]string{
		"config_file":  cmd.ConfigPath,
		"valid_depth":  fmt.Sprintf("%d", cfg.ValidDepth),
		"backend_type": cfg.BackendType,
	}

	// Add backend config
	for key, value := range cfg.BackendConfig {
		data[fmt.Sprintf("backend_config.%s", key)] = value
	}

	// Add secrets
	for secretName, envName := range cfg.Secrets {
		data[fmt.Sprintf("secret.%s", secretName)] = envName
	}

	sendResponse(conn, Response{Success: true, Data: data})
}

// handleFindConfig handles the find_config command
func handleFindConfig(conn net.Conn, cmd Command) {
	// Find config
	configPath, err := config.FindConfig(cmd.CurrentDir, cmd.MaxLevels)
	if err != nil {
		sendResponse(conn, Response{Success: false, Error: fmt.Sprintf("Failed to find config: %v", err)})
		return
	}

	sendResponse(conn, Response{
		Success: true,
		Data: map[string]string{
			"config_path": configPath,
		},
	})
}

// sendResponse sends a response to the client
func sendResponse(conn net.Conn, resp Response) {
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(resp); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// runClient sends a command to the server and returns the response
func runClient(socketPath string, cmd Command) (*Response, error) {
	// Connect to server
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Send command
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(cmd); err != nil {
		return nil, fmt.Errorf("failed to encode command: %v", err)
	}

	// Read response
	decoder := json.NewDecoder(conn)
	var resp Response
	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &resp, nil
}

// findAndLoadConfig finds and loads the config file
func findAndLoadConfig(configPath string, maxLevels int) (string, *config.ImbuedConfig, error) {
	var configFilePath string
	if configPath != "" {
		configFilePath = configPath
	} else {
		currentDir, err := os.Getwd()
		if err != nil {
			return "", nil, fmt.Errorf("failed to get current directory: %v", err)
		}

		configFilePath, err = config.FindConfig(currentDir, maxLevels)
		if err != nil {
			return "", nil, fmt.Errorf("failed to find config file: %v", err)
		}
	}

	cfg, err := config.LoadConfig(configFilePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to load config: %v", err)
	}

	return configFilePath, cfg, nil
}

// initializeBackend initializes the secret backend
func initializeBackend(cfg *config.ImbuedConfig) (secrets.Backend, error) {
	backend, err := secrets.NewBackend(cfg.BackendType)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret backend: %v", err)
	}

	if err := backend.Initialize(cfg.BackendConfig); err != nil {
		return nil, fmt.Errorf("failed to initialize secret backend: %v", err)
	}

	return backend, nil
}

// setupCommonFlags adds common flags to a command
func setupCommonFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to the .imbued config file (default: search in current and parent directories)")
	cmd.PersistentFlags().IntVar(&maxLevels, "max-levels", 3, "Maximum number of directory levels to search up for .imbued file")
	cmd.PersistentFlags().StringVar(&logFile, "log-file", "", "Path to the log file (default: ~/.imbued/logs/imbued.log)")
	cmd.PersistentFlags().DurationVar(&authDuration, "auth-duration", 1*time.Hour, "Duration for which authentication is valid")
	cmd.PersistentFlags().StringVar(&socketPath, "socket", "", "Unix socket path for server (default: ~/.imbued/socket)")
}

// setupDefaultPaths sets up default paths for log file and socket
func setupDefaultPaths() error {
	// Set default log file path if not specified
	if logFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %v", err)
		}
		logFile = filepath.Join(homeDir, ".imbued", "logs", "imbued.log")
	}

	// Ensure log directory exists
	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// Set default socket path if not specified
	if socketPath == "" {
		var err error
		socketPath, err = getDefaultSocketPath()
		if err != nil {
			return fmt.Errorf("failed to get default socket path: %v", err)
		}
	}

	return nil
}

func main() {
	// Create root command
	rootCmd := &cobra.Command{
		Use:   "imbued",
		Short: "Imbued is a tool for managing secrets",
		Long:  `Imbued is a tool for managing secrets and injecting them into environment variables.`,
	}

	// Setup common flags
	setupCommonFlags(rootCmd)

	// Create server command
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Commands for server operations",
		Long:  `Commands for server operations, such as starting the daemon.`,
	}

	// Create daemon command
	daemonCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the imbued server daemon",
		Long:  `Start the imbued server daemon that listens for client requests.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setupDefaultPaths(); err != nil {
				return err
			}

			// Initialize tracker
			tracker, err := tracking.NewFileTracker(logFile)
			if err != nil {
				return fmt.Errorf("failed to initialize tracker: %v", err)
			}
			defer tracker.Close()

			// Initialize authenticator
			authenticator := auth.NewSimpleAuthenticator(authDuration)

			log.Printf("Running in server mode")
			if err := runServer(socketPath, tracker, authenticator); err != nil {
				return fmt.Errorf("server error: %v", err)
			}

			return nil
		},
	}

	// Add daemon command to server command
	serverCmd.AddCommand(daemonCmd)

	// Create check-auth command
	checkAuthCmd := &cobra.Command{
		Use:   "check-auth",
		Short: "Check if the current process is authenticated",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setupDefaultPaths(); err != nil {
				return err
			}

			processID := auth.GetParentProcessID()

			// Send check_auth command to server
			clientCmd := Command{
				Action:    "check_auth",
				ProcessID: processID,
			}

			resp, err := runClient(socketPath, clientCmd)
			if err != nil {
				return fmt.Errorf("failed to check auth: %v", err)
			}

			if !resp.Success {
				return fmt.Errorf("failed to check auth: %s", resp.Error)
			}

			lgr := getLogger()

			isAuthenticated := resp.Data["authenticated"] == "true"
			if isAuthenticated {
				lgr.Info("Process is authenticated")
			} else {
				lgr.Error("Process is not authenticated")
				os.Exit(1)
			}

			return nil
		},
	}

	// Create auth command
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate the current process",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setupDefaultPaths(); err != nil {
				return err
			}

			lgr := getLogger()
			processID := auth.GetParentProcessID()
			lgr.Info("Authenticating process ID: %s\n", processID)
			currentDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %v", err)
			}

			// Find config file
			var configFilePath string
			if configPath != "" {
				configFilePath = configPath
			} else {
				// Send find_config command to server
				clientCmd := Command{
					Action:     "find_config",
					CurrentDir: currentDir,
					MaxLevels:  maxLevels,
				}

				resp, err := runClient(socketPath, clientCmd)
				if err != nil {
					return fmt.Errorf("failed to find config: %v", err)
				}

				if !resp.Success {
					return fmt.Errorf("failed to find config: %s", resp.Error)
				}

				configFilePath = resp.Data["config_path"]
			}

			// Send authenticate command to server
			clientCmd := Command{
				Action:     "authenticate",
				ConfigPath: configFilePath,
				ProcessID:  processID,
			}

			resp, err := runClient(socketPath, clientCmd)
			if err != nil {
				return fmt.Errorf("failed to authenticate: %v", err)
			}

			if !resp.Success {
				return fmt.Errorf("authentication failed: %s", resp.Error)
			}
			lgr.Info("Authentication successful")
			return nil
		},
	}

	// Create get-secret command
	getSecretCmd := &cobra.Command{
		Use:   "get-secret [secret-name]",
		Short: "Get a secret by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setupDefaultPaths(); err != nil {
				return err
			}

			secretName := args[0]
			processID := auth.GetCurrentProcessID()
			currentDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %v", err)
			}

			// Find config file
			var configFilePath string
			if configPath != "" {
				configFilePath = configPath
			} else {
				// Send find_config command to server
				clientCmd := Command{
					Action:     "find_config",
					CurrentDir: currentDir,
					MaxLevels:  maxLevels,
				}

				resp, err := runClient(socketPath, clientCmd)
				if err != nil {
					return fmt.Errorf("failed to find config: %v", err)
				}

				if !resp.Success {
					return fmt.Errorf("failed to find config: %s", resp.Error)
				}

				configFilePath = resp.Data["config_path"]
			}

			// Send get_secret command to server
			clientCmd := Command{
				Action:     "get_secret",
				ConfigPath: configFilePath,
				SecretName: secretName,
				ProcessID:  processID,
			}

			resp, err := runClient(socketPath, clientCmd)
			if err != nil {
				return fmt.Errorf("failed to get secret: %v", err)
			}

			if !resp.Success {
				return fmt.Errorf("failed to get secret: %s", resp.Error)
			}

			fmt.Printf("%s=%s\n", resp.Data["env_name"], resp.Data["value"])
			return nil
		},
	}

	// Create list-secrets command
	listSecretsCmd := &cobra.Command{
		Use:   "list-secrets",
		Short: "List available secrets",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setupDefaultPaths(); err != nil {
				return err
			}

			currentDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %v", err)
			}

			// Find config file
			var configFilePath string
			if configPath != "" {
				configFilePath = configPath
			} else {
				// Send find_config command to server
				clientCmd := Command{
					Action:     "find_config",
					CurrentDir: currentDir,
					MaxLevels:  maxLevels,
				}

				resp, err := runClient(socketPath, clientCmd)
				if err != nil {
					return fmt.Errorf("failed to find config: %v", err)
				}

				if !resp.Success {
					return fmt.Errorf("failed to find config: %s", resp.Error)
				}

				configFilePath = resp.Data["config_path"]
			}

			// Send list_secrets command to server
			clientCmd := Command{
				Action:     "list_secrets",
				ConfigPath: configFilePath,
			}

			resp, err := runClient(socketPath, clientCmd)
			if err != nil {
				return fmt.Errorf("failed to list secrets: %v", err)
			}

			if !resp.Success {
				return fmt.Errorf("failed to list secrets: %s", resp.Error)
			}

			fmt.Println("Available secrets:")
			for secretName, envName := range resp.Data {
				fmt.Printf("%s (env: %s)\n", secretName, envName)
			}

			return nil
		},
	}

	// Create inject-env command
	injectEnvCmd := &cobra.Command{
		Use:   "inject-env",
		Short: "Inject secrets into environment variables",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setupDefaultPaths(); err != nil {
				return err
			}

			processID := auth.GetParentProcessID()
			currentDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %v", err)
			}

			// Find config file
			var configFilePath string
			if configPath != "" {
				configFilePath = configPath
			} else {
				// Send find_config command to server
				clientCmd := Command{
					Action:     "find_config",
					CurrentDir: currentDir,
					MaxLevels:  maxLevels,
				}

				resp, err := runClient(socketPath, clientCmd)
				if err != nil {
					return fmt.Errorf("failed to find config: %v", err)
				}

				if !resp.Success {
					return fmt.Errorf("failed to find config: %s", resp.Error)
				}

				configFilePath = resp.Data["config_path"]
			}

			// Send inject_env command to server
			clientCmd := Command{
				Action:     "inject_env",
				ConfigPath: configFilePath,
				ProcessID:  processID,
			}

			resp, err := runClient(socketPath, clientCmd)
			if err != nil {
				return fmt.Errorf("failed to inject env: %v", err)
			}

			if !resp.Success {
				return fmt.Errorf("failed to inject env: %s", resp.Error)
			}

			// Print environment variables
			for envName, value := range resp.Data {
				fmt.Printf("export %s=%s\n", envName, value)
			}

			return nil
		},
	}

	// Create clean-env command
	cleanEnvCmd := &cobra.Command{
		Use:   "clean-env",
		Short: "Clean environment variables set by imbued",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setupDefaultPaths(); err != nil {
				return err
			}

			currentDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %v", err)
			}

			// Find config file
			var configFilePath string
			if configPath != "" {
				configFilePath = configPath
			} else {
				// Send find_config command to server
				clientCmd := Command{
					Action:     "find_config",
					CurrentDir: currentDir,
					MaxLevels:  maxLevels,
				}

				resp, err := runClient(socketPath, clientCmd)
				if err != nil {
					return fmt.Errorf("failed to find config: %v", err)
				}

				if !resp.Success {
					return fmt.Errorf("failed to find config: %s", resp.Error)
				}

				configFilePath = resp.Data["config_path"]
			}

			// Send clean_env command to server
			clientCmd := Command{
				Action:     "clean_env",
				ConfigPath: configFilePath,
			}

			resp, err := runClient(socketPath, clientCmd)
			if err != nil {
				return fmt.Errorf("failed to clean env: %v", err)
			}

			if !resp.Success {
				return fmt.Errorf("failed to clean env: %s", resp.Error)
			}

			// Print environment variables to unset
			for envName := range resp.Data {
				fmt.Printf("unset %s\n", envName)
			}

			return nil
		},
	}

	// Create show-config command
	showConfigCmd := &cobra.Command{
		Use:   "show-config",
		Short: "Show the current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setupDefaultPaths(); err != nil {
				return err
			}

			currentDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %v", err)
			}

			// Find config file
			var configFilePath string
			if configPath != "" {
				configFilePath = configPath
			} else {
				// Send find_config command to server
				clientCmd := Command{
					Action:     "find_config",
					CurrentDir: currentDir,
					MaxLevels:  maxLevels,
				}

				resp, err := runClient(socketPath, clientCmd)
				if err != nil {
					return fmt.Errorf("failed to find config: %v", err)
				}

				if !resp.Success {
					return fmt.Errorf("failed to find config: %s", resp.Error)
				}

				configFilePath = resp.Data["config_path"]
			}

			// Send show_config command to server
			clientCmd := Command{
				Action:     "show_config",
				ConfigPath: configFilePath,
			}

			resp, err := runClient(socketPath, clientCmd)
			if err != nil {
				return fmt.Errorf("failed to show config: %v", err)
			}

			if !resp.Success {
				return fmt.Errorf("failed to show config: %s", resp.Error)
			}

			// Print config
			fmt.Printf("Config file: %s\n", resp.Data["config_file"])
			fmt.Printf("Valid depth: %s\n", resp.Data["valid_depth"])
			fmt.Printf("Backend type: %s\n", resp.Data["backend_type"])

			// Print backend config
			fmt.Println("Backend config:")
			for key, value := range resp.Data {
				if strings.HasPrefix(key, "backend_config.") {
					configKey := strings.TrimPrefix(key, "backend_config.")
					fmt.Printf("  %s: %s\n", configKey, value)
				}
			}

			// Print secrets
			fmt.Println("Secrets:")
			for key := range resp.Data {
				if strings.HasPrefix(key, "secret.") {
					secretName := strings.TrimPrefix(key, "secret.")
					fmt.Printf("  %s\n", secretName)
				}
			}

			return nil
		},
	}

	smeltCmd := &cobra.Command{
		Use:   "smelt [file-name]",
		Short: "Store all secrets from specified env file in the macOS Keychain, under the given prefix (if provided).",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setupDefaultPaths(); err != nil {
				return err
			}

			if len(args) != 1 {
				return fmt.Errorf("file-name argument is required")
			}

			fileName := args[0]
			prefix, _ := cmd.Flags().GetString("prefix")
			if len(prefix) == 0 {
				currentDir, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %v", err)
				}
				prefix = filepath.Dir(currentDir)
			}

			// Open and parse the env file
			file, err := os.Open(fileName)
			if err != nil {
				return fmt.Errorf("failed to open env file: %v", err)
			}
			defer file.Close()

			envVars := make(map[string]string)
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}

				parts := strings.SplitN(line, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid line in env file: %s", line)
				}

				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				envVars[key] = value
			}

			if err := scanner.Err(); err != nil {
				return fmt.Errorf("failed to read env file: %v", err)
			}

			wd, err := os.Getwd()
			var fig *config.ImbuedConfig
			if err != nil {
				return fmt.Errorf("failed to get current directory: %v", err)
			} else if configPath, err = config.FindConfig(wd, 1); err != nil {
				return fmt.Errorf("failed to find config file: %v", err)
			} else if fig, err = config.LoadConfig(configPath); err != nil {
				return fmt.Errorf("failed to load config file: %v", err)
			}

			clientCmd := Command{
				Action:      "store_secrets",
				Environment: envVars,
				ProcessID:   auth.GetParentProcessID(),
				ConfigPath:  configPath,
				BackendType: fig.BackendType,
			}
			clientCmd.Environment["prefix"] = prefix

			resp, err := runClient(socketPath, clientCmd)
			if err != nil {
				return fmt.Errorf("failed to store secrets: %v", err)
			} else if !resp.Success {
				return fmt.Errorf("failed to store secrets: %s", resp.Error)
			}

			fmt.Println("Secrets stored successfully in the macOS Keychain")
			return nil
		},
	}

	smeltCmd.Flags().String("prefix", "", "Optional prefix to prepend to each key before storing in the keychain")

	// Create client command
	clientCmd := &cobra.Command{
		Use:   "client",
		Short: "Commands for client operations",
		Long:  `Commands for client operations, such as authentication and secret management.`,
	}

	// Add client subcommands
	clientCmd.AddCommand(checkAuthCmd)
	clientCmd.AddCommand(authCmd)
	clientCmd.AddCommand(getSecretCmd)
	clientCmd.AddCommand(listSecretsCmd)
	clientCmd.AddCommand(injectEnvCmd)
	clientCmd.AddCommand(cleanEnvCmd)
	clientCmd.AddCommand(showConfigCmd)
	clientCmd.AddCommand(smeltCmd)

	// Create credentials command
	credentialsCmd := &cobra.Command{
		Use:   "credentials",
		Short: "Commands for managing backend credentials",
		Long:  `Commands for managing backend credentials, such as setting and clearing 1Password credentials.`,
	}

	// Create set-onepass-credentials command
	var accountToken, vaultID string
	setOnePassCredentialsCmd := &cobra.Command{
		Use:   "set-onepass",
		Short: "Set 1Password credentials in the macOS Keychain",
		Long:  `Set 1Password credentials (account token and vault ID) in the macOS Keychain for use by the onepass backend.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if accountToken == "" {
				return fmt.Errorf("account-token is required")
			}
			if vaultID == "" {
				return fmt.Errorf("vault-id is required")
			}

			if err := secrets.StoreOnePassCredentials(accountToken, vaultID); err != nil {
				return fmt.Errorf("failed to store 1Password credentials: %v", err)
			}

			fmt.Println("1Password credentials stored successfully in the macOS Keychain")
			return nil
		},
	}
	setOnePassCredentialsCmd.Flags().StringVar(&accountToken, "account-token", "", "1Password service account token")
	setOnePassCredentialsCmd.Flags().StringVar(&vaultID, "vault-id", "", "1Password vault ID or name")

	// Create check-onepass-credentials command
	checkOnePassCredentialsCmd := &cobra.Command{
		Use:   "check-onepass",
		Short: "Check if 1Password credentials are set in the macOS Keychain",
		Long:  `Check if 1Password credentials (account token and vault ID) are set in the macOS Keychain.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Try to get account token
			accountToken, err := secrets.GetKeychainItem(secrets.KeychainServiceName, secrets.KeychainAccountTokenKey)
			if err != nil {
				return fmt.Errorf("1Password account token not found in keychain: %v", err)
			}

			// Try to get vault ID
			vaultID, err := secrets.GetKeychainItem(secrets.KeychainServiceName, secrets.KeychainVaultIDKey)
			if err != nil {
				return fmt.Errorf("1Password vault ID not found in keychain: %v", err)
			}

			fmt.Println("1Password credentials found in the macOS Keychain:")
			fmt.Printf("Account token: %s...\n", accountToken[:10]+"***")
			fmt.Printf("Vault ID: %s\n", vaultID)
			return nil
		},
	}

	// Create clear-onepass-credentials command
	clearOnePassCredentialsCmd := &cobra.Command{
		Use:   "clear-onepass",
		Short: "Clear 1Password credentials from the macOS Keychain",
		Long:  `Clear 1Password credentials (account token and vault ID) from the macOS Keychain.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Delete account token
			accountTokenCmd := exec.Command("security", "delete-generic-password", "-s", secrets.KeychainServiceName, "-a", secrets.KeychainAccountTokenKey)
			if err := accountTokenCmd.Run(); err != nil {
				return fmt.Errorf("failed to delete 1Password account token from keychain: %v", err)
			}

			// Delete vault ID
			vaultIDCmd := exec.Command("security", "delete-generic-password", "-s", secrets.KeychainServiceName, "-a", secrets.KeychainVaultIDKey)
			if err := vaultIDCmd.Run(); err != nil {
				return fmt.Errorf("failed to delete 1Password vault ID from keychain: %v", err)
			}

			fmt.Println("1Password credentials cleared from the macOS Keychain")
			return nil
		},
	}

	// Add credential commands
	credentialsCmd.AddCommand(setOnePassCredentialsCmd)
	credentialsCmd.AddCommand(checkOnePassCredentialsCmd)
	credentialsCmd.AddCommand(clearOnePassCredentialsCmd)

	// Add commands to root command
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(clientCmd)
	rootCmd.AddCommand(credentialsCmd)

	// Execute root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
