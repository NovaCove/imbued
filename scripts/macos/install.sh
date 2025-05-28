#!/bin/bash

# install.sh - Install imbued as a launchd service on macOS

set -e

# Get the directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Get the directory of the installed imbued binary
IMBUED_BIN_DIR="$PROJECT_DIR/bin"
if [ ! -d "$IMBUED_BIN_DIR" ]; then
    echo "Error: The imbued binary directory does not exist: $IMBUED_BIN_DIR"
    exit 1
fi

# Install the binary to /usr/local/bin
echo "Installing imbued to /usr/local/bin..."
sudo cp "$PROJECT_DIR/bin/imbued" /usr/local/bin/
sudo chmod +x /usr/local/bin/imbued

# Create the imbued directory structure
echo "Creating imbued directory structure..."
mkdir -p ~/.imbued/logs

# Install the launchd plist file
echo "Installing launchd plist file..."
# Replace ~ with the actual home directory in the plist file
sed "s|~|$HOME|g" "$SCRIPT_DIR/com.novacove.imbued.plist" > /tmp/com.novacove.imbued.plist
cp /tmp/com.novacove.imbued.plist ~/Library/LaunchAgents/com.novacove.imbued.plist
rm /tmp/com.novacove.imbued.plist

# Load the launchd service
echo "Loading launchd service..."
launchctl load ~/Library/LaunchAgents/com.novacove.imbued.plist

echo "Installation complete!"
echo "The imbued server is now running as a launchd service."
echo "It will automatically start when you log in."
echo "You can use the imbued client by sourcing the appropriate shell script:"
echo "  For bash: source $PROJECT_DIR/scripts/bash/imbued.sh"
echo "  For fish: source $PROJECT_DIR/scripts/fish/imbued.fish"
