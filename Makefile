# Makefile for Imbued

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=imbued
BINARY_DIR=bin
MAIN_PATH=cmd/imbued/main.go

# Shell scripts
BASH_SCRIPT=scripts/bash/imbued.sh
FISH_SCRIPT=scripts/fish/imbued.fish
MACOS_PLIST=scripts/macos/com.novacove.imbued.plist
MACOS_INSTALL_SCRIPT=scripts/macos/install.sh

# Installation paths
PREFIX?=/usr/local
INSTALL_BIN=$(PREFIX)/bin
INSTALL_SCRIPTS=$(PREFIX)/share/imbued/scripts
LAUNCHD_DIR=$(HOME)/Library/LaunchAgents

.PHONY: all build clean test deps install uninstall install-server uninstall-server homebrew-tap

all: deps build

build:
	mkdir -p $(BINARY_DIR)
	$(GOBUILD) -o $(BINARY_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	chmod +x $(BINARY_DIR)/$(BINARY_NAME)

clean:
	$(GOCLEAN)
	rm -rf $(BINARY_DIR)

test:
	$(GOTEST) -v ./...

deps:
	$(GOMOD) tidy

install: build
	mkdir -p $(INSTALL_BIN)
	mkdir -p $(INSTALL_SCRIPTS)/bash
	mkdir -p $(INSTALL_SCRIPTS)/fish
	install -m 755 $(BINARY_DIR)/$(BINARY_NAME) $(INSTALL_BIN)/$(BINARY_NAME)
	install -m 644 $(BASH_SCRIPT) $(INSTALL_SCRIPTS)/bash/imbued.sh
	install -m 644 $(FISH_SCRIPT) $(INSTALL_SCRIPTS)/fish/imbued.fish
	@echo "Installed imbued to $(INSTALL_BIN)/$(BINARY_NAME)"
	@echo "Installed bash script to $(INSTALL_SCRIPTS)/bash/imbued.sh"
	@echo "Installed fish script to $(INSTALL_SCRIPTS)/fish/imbued.fish"
	@echo ""
	@echo "To use imbued with bash, add the following to your .bashrc or .bash_profile:"
	@echo "  export IMBUED_BIN=$(INSTALL_BIN)/$(BINARY_NAME)"
	@echo "  source $(INSTALL_SCRIPTS)/bash/imbued.sh"
	@echo ""
	@echo "To use imbued with fish, add the following to your config.fish:"
	@echo "  set -gx IMBUED_BIN $(INSTALL_BIN)/$(BINARY_NAME)"
	@echo "  source $(INSTALL_SCRIPTS)/fish/imbued.fish"

uninstall:
	rm -f $(INSTALL_BIN)/$(BINARY_NAME)
	rm -f $(INSTALL_SCRIPTS)/bash/imbued.sh
	rm -f $(INSTALL_SCRIPTS)/fish/imbued.fish
	@echo "Uninstalled imbued"

# Install imbued as a launchd service on macOS
install-server: build
	mkdir -p $(INSTALL_BIN)
	mkdir -p $(HOME)/.imbued/logs
	mkdir -p $(LAUNCHD_DIR)
	install -m 755 $(BINARY_DIR)/$(BINARY_NAME) $(INSTALL_BIN)/$(BINARY_NAME)
	sed "s|~|$(HOME)|g" $(MACOS_PLIST) > /tmp/com.novacove.imbued.plist
	install -m 644 /tmp/com.novacove.imbued.plist $(LAUNCHD_DIR)/com.novacove.imbued.plist
	rm -f /tmp/com.novacove.imbued.plist
	launchctl load $(LAUNCHD_DIR)/com.novacove.imbued.plist
	@echo "Installed imbued server as a launchd service"
	@echo "The server will start automatically when you log in"
	@echo ""
	@echo "To use imbued with bash, add the following to your .bashrc or .bash_profile:"
	@echo "  export IMBUED_BIN=$(INSTALL_BIN)/$(BINARY_NAME)"
	@echo "  export IMBUED_SOCKET=$(HOME)/.imbued/imbued.sock"
	@echo "  source $(INSTALL_SCRIPTS)/bash/imbued.sh"
	@echo ""
	@echo "To use imbued with fish, add the following to your config.fish:"
	@echo "  set -gx IMBUED_BIN $(INSTALL_BIN)/$(BINARY_NAME)"
	@echo "  set -gx IMBUED_SOCKET $(HOME)/.imbued/imbued.sock"
	@echo "  source $(INSTALL_SCRIPTS)/fish/imbued.fish"

# Uninstall imbued server from launchd
uninstall-server:
	launchctl unload $(LAUNCHD_DIR)/com.novacove.imbued.plist
	rm -f $(LAUNCHD_DIR)/com.novacove.imbued.plist
	@echo "Uninstalled imbued server from launchd"

# Create Homebrew tap repository structure
homebrew-tap:
	@echo "Creating Homebrew tap repository structure..."
	@mkdir -p Formula
	@chmod +x scripts/homebrew/setup-tap.sh
	@./scripts/homebrew/setup-tap.sh
	@echo "Homebrew tap repository structure created."
	@echo "Follow the instructions above to publish the tap to GitHub."
