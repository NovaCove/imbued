#!/bin/bash

# install.sh - Install imbued as a launchd service on macOS

set -e

# Get the bin and script dir from the first and second arguments to this script.
if [ $# -ne 2 ]; then
    echo "Usage: $0 <bin_dir> <script_dir>"
    exit 1
fi
BIN_DIR="$1"
SCRIPT_DIR="$2"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"


# Create the imbued directory structure
echo "Creating imbued directory structure..."
mkdir -p ~/.imbued/logs

# Install the launchd plist file
echo "Installing launchd plist file..."
# Replace ~ with the actual home directory in the plist file
# sed "s|~|$HOME|g" "$SCRIPT_DIR/com.novacove.imbued.plist" > /tmp/com.novacove.imbued.plist
cp "$SCRIPT_DIR/com.novacove.imbued.plist" ~/Library/LaunchAgents/com.novacove.imbued.plist
# rm /tmp/com.novacove.imbued.plist

# Load the launchd service
echo "Loading launchd service..."
launchctl load ~/Library/LaunchAgents/com.novacove.imbued.plist