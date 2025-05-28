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

# Check if the bin directory exists
if [ ! -d "$BIN_DIR" ]; then
    echo "Error: Bin directory '$BIN_DIR' does not exist."
    exit 1
fi
# Check if the script directory exists
if [ ! -d "$SCRIPT_DIR" ]; then
    echo "Error: Script directory '$SCRIPT_DIR' does not exist."
    exit 1
fi
# Check if the imbued binary exists
if [ ! -f "$BIN_DIR/imbued" ]; then
    echo "Error: imbued binary '$BIN_DIR/imbued' does not exist."
    exit 1
fi

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

echo "Installation complete!"
echo "The imbued server is now running as a launchd service."
echo "It will automatically start when you log in."
echo "You can use the imbued client by sourcing the appropriate shell script:"
echo "  For bash: source $PROJECT_DIR/scripts/bash/imbued.sh"
echo "  For fish: source $PROJECT_DIR/scripts/fish/imbued.fish"
