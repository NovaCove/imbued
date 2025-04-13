# Imbued

Imbued is a toolset for managing secrets in a development environment. It automatically injects secrets into your environment variables when you enter a directory with an `.imbued` configuration file, and removes them when you exit the directory.

Imbued can run in two modes:
- **Direct mode**: The imbued binary is executed directly for each operation
- **Server mode**: The imbued server runs as a background service, and the shell scripts communicate with it via a Unix socket

## Features

- Automatically injects secrets into environment variables when entering a directory with an `.imbued` configuration file
- Automatically removes environment variables when exiting the directory
- Supports multiple secret backends:
  - Environment file
  - HashiCorp Vault
  - 1Password (see [1Password Backend Documentation](pkg/secrets/onepass_README.md))
  - AWS Secret Manager
  - GCP Secret Manager
- Authenticates users before allowing access to secrets
- Tracks all access and authentication requests
- Supports both Bash and Fish shells

## Installation

### Prerequisites

- Go 1.16 or later
- Bash or Fish shell

### Building from source

```bash
git clone https://github.com/novacove/imbued.git
cd imbued
go build -o bin/imbued cmd/imbued/main.go
```

### Installing as a launchd service (macOS)

Imbued includes a script to install it as a launchd service on macOS:

```bash
cd imbued
./scripts/macos/install.sh
```

This script will:
1. Build the imbued binary
2. Install it to `/usr/local/bin`
3. Create the necessary directory structure in `~/.imbued`
4. Install a launchd plist file to `~/Library/LaunchAgents`
5. Load the launchd service

The imbued server will now run automatically when you log in.

### Shell integration

#### Bash

Add the following to your `.bashrc` or `.bash_profile`:

```bash
# Set the path to the imbued binary (optional if installed to /usr/local/bin)
export IMBUED_BIN=/path/to/imbued

# Set the socket path (optional)
export IMBUED_SOCKET=$HOME/.imbued/imbued.sock

# Source the imbued script
source /path/to/imbued/scripts/bash/imbued.sh
```

#### Fish

Add the following to your `config.fish`:

```fish
# Set the path to the imbued binary (optional if installed to /usr/local/bin)
set -gx IMBUED_BIN /path/to/imbued

# Set the socket path (optional)
set -gx IMBUED_SOCKET $HOME/.imbued/imbued.sock

# Source the imbued script
source /path/to/imbued/scripts/fish/imbued.fish
```

## Usage

### Creating an `.imbued` configuration file

Create a file named `.imbued` in the root directory of your project. Here's an example:

```toml
# Number of child directories down that secrets are available for
valid_depth = 2

# Type of secret backend to use
backend_type = "env_file"

# Backend-specific configuration
[backend_config]
file_path = "/path/to/secrets.env"

# Secrets to retrieve
[secrets]
DB_PASSWORD = "DATABASE_PASSWORD"
API_KEY = "API_KEY"
GITHUB_TOKEN = "GITHUB_TOKEN"
```

See `docs/sample.imbued` for a more detailed example.

### Using the CLI

Imbued provides a command-line interface for managing secrets:

```bash
# Direct mode
# Show the current configuration
imbued --show-config

# List available secrets
imbued --list-secrets

# Get a secret by name
imbued --get-secret DB_PASSWORD

# Authenticate the current process
imbued --authenticate

# Check if the current process is authenticated
imbued --check-auth

# Inject secrets into environment variables
imbued --inject-env

# Clean environment variables set by imbued
imbued --clean-env

# Server mode
# Run in daemon mode as a server
imbued --daemon --socket ~/.imbued/imbued.sock

# Client mode (communicates with the server)
imbued --client --socket ~/.imbued/imbued.sock --show-config
imbued --client --socket ~/.imbued/imbued.sock --list-secrets
imbued --client --socket ~/.imbued/imbued.sock --get-secret DB_PASSWORD
imbued --client --socket ~/.imbued/imbued.sock --authenticate
imbued --client --socket ~/.imbued/imbued.sock --check-auth
imbued --client --socket ~/.imbued/imbued.sock --inject-env
imbued --client --socket ~/.imbued/imbued.sock --clean-env
```

## How it works

### Direct mode

1. When you enter a directory, the shell integration script checks for an `.imbued` configuration file in the current directory or parent directories (up to a configurable number of levels).
2. If an `.imbued` file is found, the script checks if the current process has been authenticated to retrieve secrets.
3. If not authenticated, the script prompts for authentication.
4. Once authenticated, the script retrieves the secrets from the configured backend and injects them into environment variables.
5. When you exit the directory (or go beyond the valid depth), the script removes the environment variables.

### Server mode

1. The imbued server runs as a background service, listening on a Unix socket.
2. When you enter a directory, the shell integration script checks for an `.imbued` configuration file in the current directory or parent directories (up to a configurable number of levels).
3. If an `.imbued` file is found, the script sends a request to the server to check if the current process has been authenticated to retrieve secrets.
4. If not authenticated, the script sends a request to the server to authenticate the current process.
5. Once authenticated, the script sends a request to the server to retrieve the secrets from the configured backend and injects them into environment variables.
6. When you exit the directory (or go beyond the valid depth), the script sends a request to the server to remove the environment variables.

The server mode has several advantages:
- The server can maintain authentication state across multiple shells
- The server can cache secrets, reducing the need to retrieve them from the backend
- The server can handle authentication and secret retrieval in a centralized way

## License

MIT
