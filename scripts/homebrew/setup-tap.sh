#!/bin/bash
# Script to set up a Homebrew tap for Imbued

set -e

# Configuration
TAP_NAME="homebrew-in5"
GITHUB_USERNAME="novacove"
GITHUB_REPO="$GITHUB_USERNAME/$TAP_NAME"

# Create the tap directory structure
echo "Creating tap directory structure..."
mkdir -p "$TAP_NAME/Formula"

# Copy the formula file
echo "Copying formula file..."
cp Formula/imbued.rb "$TAP_NAME/Formula/"

# Copy the README file
echo "Copying README file..."
cp README.md.tap "$TAP_NAME/README.md"

echo "Homebrew tap structure created in ./$TAP_NAME/"
echo ""
echo "Next steps:"
echo "1. Create a new GitHub repository at https://github.com/new"
echo "   - Repository name: $TAP_NAME"
echo "   - Description: Homebrew tap for NovaCove tools"
echo "   - Visibility: Public"
echo ""
echo "2. Initialize the Git repository and push to GitHub:"
echo "   cd $TAP_NAME"
echo "   git init"
echo "   git add ."
echo "   git commit -m \"Initial commit\""
echo "   git branch -M main"
echo "   git remote add origin https://github.com/$GITHUB_REPO.git"
echo "   git push -u origin main"
echo ""
echo "3. Users can then install Imbued with:"
echo "   brew tap $GITHUB_USERNAME/in5"
echo "   brew install imbued"
echo ""
echo "Note: Make sure to create a release on the main Imbued repository with the tag 'v0.1.0'"
echo "      or update the formula to point to the correct tag."
echo ""
echo "To add additional NovaCove tools to this tap:"
echo "1. Create a new formula file for each tool in the $TAP_NAME/Formula directory"
echo "2. Update the README.md in the tap repository to list the new tool"
echo "3. Commit and push the changes to GitHub"
