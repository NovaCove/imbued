#!/usr/bin/env fish
# imbued.fish - Fish shell integration for imbued
# This script should be sourced in your config.fish

# Path to the imbued binary
if set -q IMBUED_BIN
    set -g _imbued_bin $IMBUED_BIN
else
    set -g _imbued_bin (which imbued)
    if test -z "$_imbued_bin"
        set _script_dir (dirname (status -f))
        if test -x "$_script_dir/../../cmd/imbued/imbued"
            set -g _imbued_bin "$_script_dir/../../cmd/imbued/imbued"
        else
            echo "imbued binary not found. Please install it or set IMBUED_BIN environment variable."
            exit 1
        end
    end
end

# Socket path for the imbued server
if set -q IMBUED_SOCKET
    set -g _imbued_socket $IMBUED_SOCKET
else
    set -g _imbued_socket $HOME/.imbued/imbued.sock
end

# Maximum number of directory levels to search up for .imbued file
if set -q IMBUED_MAX_LEVELS
    set -g _imbued_max_levels $IMBUED_MAX_LEVELS
else
    set -g _imbued_max_levels 3
end

# Current .imbued file path
set -g _imbued_current_config ""

# Function to clean environment variables set by imbued
function _imbued_clean_env
    if test -n "$_imbued_current_config"
        # Get the list of environment variables to unset
        set -l env_vars (eval "$_imbued_bin --client --socket $_imbued_socket --config $_imbued_current_config --clean-env")
        
        # Unset each environment variable
        for line in $env_vars
            if string match -q -r "^unset (.+)\$" -- $line
                set -l env_name (string match -r "^unset (.+)\$" -- $line)[2]
                set -e $env_name
            end
        end
        
        # Clear the current config
        set -g _imbued_current_config ""
        
        # Notify the user
        echo "Imbued: Cleaned environment variables"
    end
end

# Function to set environment variables from imbued
function _imbued_set_env
    set -l config_path $argv[1]
    
    # Check if we need to authenticate
    if not eval "$_imbued_bin --client --socket $_imbued_socket --config $config_path --check-auth" > /dev/null 2>&1
        echo "Imbued: Authentication required"
        if not eval "$_imbued_bin --client --socket $_imbued_socket --config $config_path --authenticate"
            echo "Imbued: Authentication failed"
            return 1
        end
    end
    
    # Get the list of environment variables to set
    set -l env_vars (eval "$_imbued_bin --client --socket $_imbued_socket --config $config_path --inject-env")
    
    # Set each environment variable
    for line in $env_vars
        if string match -q -r "^export (.+)=(.+)\$" -- $line
            set -l matches (string match -r "^export (.+)=(.+)\$" -- $line)
            set -l env_name $matches[2]
            set -l env_value $matches[3]
            set -gx $env_name $env_value
        end
    end
    
    # Set the current config
    set -g _imbued_current_config $config_path
    
    # Notify the user
    echo "Imbued: Set environment variables"
end

# Function to check for .imbued file and set environment variables
function _imbued_check
    # Find .imbued file
    set -l config_path (eval "$_imbued_bin --client --socket $_imbued_socket --max-levels $_imbued_max_levels --show-config 2>/dev/null | grep 'Config file:' | cut -d' ' -f3-")
    
    # If no .imbued file found, clean environment variables
    if test -z "$config_path"
        _imbued_clean_env
        return
    end
    
    # If the .imbued file is the same as the current one, do nothing
    if test "$config_path" = "$_imbued_current_config"
        return
    end
    
    # Clean environment variables from previous .imbued file
    _imbued_clean_env
    
    # Set environment variables from new .imbued file
    _imbued_set_env $config_path
end

# Function to run when changing directory
function _imbued_cd
    # Call the original cd function
    builtin cd $argv
    
    # Check for .imbued file
    _imbued_check
end

# Function to ensure the imbued server is running
function _imbued_ensure_server
    # Check if the socket exists and is a socket
    if test ! -S "$_imbued_socket"
        echo "Imbued: Starting server..."
        # Start the server in the background
        nohup "$_imbued_bin" --daemon --socket "$_imbued_socket" > /dev/null 2>&1 &
        # Wait a moment for the server to start
        sleep 1
    end
end

# Ensure the server is running
_imbued_ensure_server

# Override the cd function
function cd --wraps cd
    _imbued_cd $argv
end

# Check for .imbued file in the current directory on startup
_imbued_check

# Clean environment variables on exit
function _imbued_on_exit --on-process-exit %self
    _imbued_clean_env
end
