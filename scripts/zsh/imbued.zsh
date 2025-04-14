#!/bin/zsh

# Ensure the script is running in Zsh
if [[ -z "$ZSH_VERSION" ]]; then
    echo "This script must be run in Zsh." >&2
    return 1
fi

setopt extended_glob

# imbued.zsh - Zsh integration for imbued
# This script should be sourced in your .zshrc

# Path to the imbued binary
IMBUED_BIN="${IMBUED_BIN:-$(which imbued)}"

# If imbued binary is not found, try to use the one in the same directory as this script
if [[ -z "$IMBUED_BIN" ]]; then
    SCRIPT_DIR="$(cd "$(dirname "${(%):-%x}")" &> /dev/null && pwd)"
    if [[ -x "$SCRIPT_DIR/../../cmd/imbued/imbued" ]]; then
        IMBUED_BIN="$SCRIPT_DIR/../../cmd/imbued/imbued"
    else
        echo "imbued binary not found. Please install it or set IMBUED_BIN environment variable."
        return 1
    fi
fi

# Socket path for the imbued server
IMBUED_SOCKET="${IMBUED_SOCKET:-$HOME/.imbued/imbued.sock}"

# Maximum number of directory levels to search up for .imbued file
IMBUED_MAX_LEVELS="${IMBUED_MAX_LEVELS:-3}"

# Current .imbued file path
IMBUED_CURRENT_CONFIG=""

# Function to clean environment variables set by imbued
imbued_clean_env() {
    if [[ -n "$IMBUED_CURRENT_CONFIG" ]]; then
        # Get the list of environment variables to unset
        local env_vars="$("$IMBUED_BIN" client clean-env --socket "$IMBUED_SOCKET" --config "$IMBUED_CURRENT_CONFIG")"
        
        # Unset each environment variable
        while read -r line; do
            if [[ "$line" =~ ^unset\ (.+)$ ]]; then
                local env_name="${match[1]}"
                unset "$env_name"
            fi
        done <<< "$env_vars"
        
        # Clear the current config
        IMBUED_CURRENT_CONFIG=""
        
        # Notify the user
        echo "Imbued: Cleaned environment variables"
    fi
}

# Function to set environment variables from imbued
imbued_set_env() {
    local config_path="$1"
    
    # Check if we need to authenticate
    if ! "$IMBUED_BIN" client check-auth --socket "$IMBUED_SOCKET" --config "$config_path" &> /dev/null; then
        echo "Imbued: Authentication required"
        if ! "$IMBUED_BIN" client auth --socket "$IMBUED_SOCKET" --config "$config_path"; then
            echo "Imbued: Authentication failed"
            return 1
        fi
    fi
    
    # Get the list of environment variables to set
    local env_vars="$("$IMBUED_BIN" client inject-env --socket "$IMBUED_SOCKET" --config "$config_path")"

    echo "Imbued: Setting environment variables from $config_path"
    echo "$env_vars"
    
    # Debug: Print raw environment variables
    echo "Imbued: Raw environment variables:"
    echo "$env_vars"

    # Set each environment variable
    while IFS= read -r line; do
        echo "Imbued: Processing line: $line"
        # Trim leading and trailing whitespace from the line
        line="${line## }"
        line="${line%% }"
        if [[ "$line" =~ ^export\ ([^=]+)=(.+)$ ]]; then
            local env_name="${match[1]}"
            local env_value="${match[2]}"
            # Remove surrounding quotes from env_value if present
            env_value="${env_value%\"}"
            env_value="${env_value#\"}"
            echo "Imbued: Setting $env_name to $env_value"
            export "$env_name"="$env_value"
        else
            echo "Imbued: Skipping invalid line: $line"
        fi
    done <<< "$env_vars"
    
    # Set the current config
    IMBUED_CURRENT_CONFIG="$config_path"
    
    # Notify the user
    echo "Imbued: Set environment variables"
}

# Function to check for .imbued file and set environment variables
imbued_check() {
    # Find .imbued file
    local config_path="$("$IMBUED_BIN" client show-config --socket "$IMBUED_SOCKET" --max-levels "$IMBUED_MAX_LEVELS" 2>/dev/null | grep "Config file:" | cut -d' ' -f3-)"
    
    # If no .imbued file found, clean environment variables
    if [[ -z "$config_path" ]]; then
        echo "Imbued: No .imbued file found"
        imbued_clean_env
        return
    fi
    
    # If the .imbued file is the same as the current one, do nothing
    if [[ "$config_path" == "$IMBUED_CURRENT_CONFIG" ]]; then
        echo "Imbued: Already using the current .imbued file"
        return
    fi
    
    # Clean environment variables from previous .imbued file
    echo "Imbued: Switching to new .imbued file"
    imbued_clean_env
    
    # Set environment variables from new .imbued file
    echo "Imbued: Setting environment variables from $config_path"
    imbued_set_env "$config_path"
}

# Function to run when changing directory
imbued_cd() {
    # Call the original cd command
    builtin cd "$@"
    
    # Check for .imbued file
    imbued_check
}

# Function to ensure the imbued server is running
imbued_ensure_server() {
    # Check if the socket exists and is a socket
    if [[ ! -S "$IMBUED_SOCKET" ]]; then
        echo "Imbued: Starting server..."
        # Start the server in the background
        nohup "$IMBUED_BIN" server start --socket "$IMBUED_SOCKET" > /dev/null 2>&1 &
        # Wait a moment for the server to start
        sleep 1
    fi
}

# Ensure the server is running
imbued_ensure_server

# Override the cd command
alias cd="imbued_cd"

# Check for .imbued file in the current directory on startup
imbued_check

# Clean environment variables on exit
trap imbued_clean_env EXIT