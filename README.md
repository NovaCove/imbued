# Imbued

Imbued is a toolset for managing secrets in a development environment. It automatically injects secrets into your environment variables when you enter a directory with an `.imbued` configuration file, and removes them when you exit the directory.

## Quick start

### Installation
#### macOS

```sh
# Add the tap
brew tap novacove/in5

# Install imbued
brew install imbued
```

### Quick usage
See the [`examples/basic`](./examples/basic/README.md) for a quick setup example of how `imbued` works.

## Features
- Automatically injects secrets into environment variables when entering a directory with an `.imbued` configuration file
- Automatically removes environment variables when exiting the directory
- Authenticates users before allowing access to secrets
- Tracks all access and authentication requests
- Supports Bash, Zsh, and Fish shells
- Supports multiple secret backends:
  - MacOS keychain
  - 1Password (see [1Password Backend Documentation](pkg/secrets/onepass_README.md))
- Plan to add support for:
  - AWS Secret Manager
  - GCP Secret Manager
  - HashiCorp Vault

## Installation

### Using Homebrew (macOS)

```bash
# Add the tap
brew tap novacove/in5

# Install imbued
brew install imbued
```

After installation, follow the provided details for configuring your shell integration.

### Prerequisites for manual installation

- Go 1.16 or later
- Bash, Zsh, or Fish shell

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

NOTE: none of the below is needed if you source the provided shell script and `homebrew`'s bin path is on your `$PATH`.

Add the following to your `.bashrc` or `.bash_profile`:

```bash
# Set the path to the imbued binary (OPTIONAL if installed to /usr/local/bin)
export IMBUED_BIN=/path/to/imbued

# Set the socket path (OPTIONAL)
export IMBUED_SOCKET=$HOME/.imbued/imbued.sock

# Source the imbued script
source /path/to/imbued/scripts/bash/imbued.sh
```

#### Zsh

Add the following to your `.zshrc`:

```zsh
# Set the path to the imbued binary (optional if installed to /usr/local/bin)
export IMBUED_BIN=/path/to/imbued

# Set the socket path (optional)
export IMBUED_SOCKET=$HOME/.imbued/imbued.sock

# Source the imbued script
source /path/to/imbued/scripts/zsh/imbued.zsh
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
# Type of secret backend to use
backend_type = "macos_keychain_manager"

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
# Server mode
# Run in daemon mode as a server
imbued server start

# Client mode (communicates with the server)
imbued client show-config
imbued client list-secrets
imbued client get-secret DB_PASSWORD
imbued client authenticate
imbued client check-auth
imbued client inject-env
imbued client clean-env
```

## How it works

### Client

1. When you enter a directory, the shell integration script checks for an `.imbued` configuration file in the current directory or parent directories (up to a configurable number of levels).
2. If an `.imbued` file is found, the script checks if the current process has been authenticated to retrieve secrets.
3. If not authenticated, the script prompts for authentication.
4. Once authenticated, the script retrieves the secrets from the configured backend and injects them into environment variables.
5. When you exit the directory (or go beyond the valid depth), the script removes the environment variables.

### Server

1. The imbued server runs as a background service, listening on a Unix socket.
2. When you enter a directory, the shell integration script checks for an `.imbued` configuration file in the current directory or parent directories (up to a configurable number of levels).
3. If an `.imbued` file is found, the script sends a request to the server to check if the current process has been authenticated to retrieve secrets.
4. If not authenticated, the script sends a request to the server to authenticate the current process.
5. Once authenticated, the script sends a request to the server to retrieve the secrets from the configured backend and injects them into environment variables.
6. When you exit the directory (or go beyond the valid depth), the script sends a request to the server to remove the environment variables.

## Contributing

Feel free to propose changes and fix issues that you encounter! Our guiding principle is to make this took as friendly and approachable as possible. Our focus is on general support and security, not niche use-cases. We're more than happy to always have a public discussion in a GitHub issue, though!

## License

MIT
