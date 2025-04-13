# 1Password Backend for Imbued

This document describes how to use the 1Password backend for Imbued to securely retrieve secrets from 1Password.

## Prerequisites

Before using the 1Password backend, you need to:

1. Install the 1Password CLI (Command Line Interface)
2. Set up a 1Password Service Account
3. Store your 1Password credentials in the macOS Keychain
4. Configure Imbued to use the 1Password backend

## 1Password CLI Installation

The 1Password backend requires the 1Password CLI (`op`) to be installed on your system.

### macOS

Using Homebrew:
```bash
brew install 1password-cli
```

### Linux

For Debian/Ubuntu:
```bash
curl -sS https://downloads.1password.com/linux/keys/1password.asc | sudo gpg --dearmor --output /usr/share/keyrings/1password-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/1password-archive-keyring.gpg] https://downloads.1password.com/linux/debian/$(dpkg --print-architecture) stable main" | sudo tee /etc/apt/sources.list.d/1password.list
sudo apt update && sudo apt install 1password-cli
```

For other Linux distributions, see the [1Password CLI documentation](https://developer.1password.com/docs/cli/get-started/).

### Windows

Using Chocolatey:
```powershell
choco install 1password-cli
```

Using Scoop:
```powershell
scoop install 1password-cli
```

## Setting Up a 1Password Service Account

To use the 1Password backend, you need to create a service account and obtain a token:

1. Sign in to your 1Password account at [1password.com](https://1password.com)
2. Go to Settings > Developer > Service Accounts
3. Click "New Service Account"
4. Give your service account a name (e.g., "Imbued")
5. Select the vault you want to use for your secrets
6. Click "Create"
7. Copy the service account token (you'll need this for the Imbued configuration)

## Storing 1Password Credentials in the macOS Keychain

Instead of storing sensitive credentials in the configuration file, Imbued stores them securely in the macOS Keychain. To set up your 1Password credentials:

```bash
# Store your 1Password credentials in the macOS Keychain
imbued credentials set-onepass --account-token "your-service-account-token" --vault-id "your-vault-id"
```

You can verify that your credentials are stored correctly:

```bash
# Check if your 1Password credentials are stored in the macOS Keychain
imbued credentials check-onepass
```

If you need to remove your credentials:

```bash
# Clear your 1Password credentials from the macOS Keychain
imbued credentials clear-onepass
```

## Configuring Imbued

To use the 1Password backend, update your `.imbued` configuration file:

```toml
# Type of secret backend to use
backend_type = "onepass"

# No backend-specific configuration needed here, as credentials are stored in the keychain

# Secrets to retrieve
# Format: secret_name = "environment_variable_name"
[secrets]
# The secret_name should match the item name in 1Password
DATABASE_PASSWORD = "DB_PASSWORD"
API_KEY = "API_KEY"
```

## Secret Structure in 1Password

The 1Password backend expects your secrets to be stored as items in your 1Password vault. The item name should match the secret name in your Imbued configuration.

For example, if your configuration includes `DATABASE_PASSWORD = "DB_PASSWORD"`, you should have an item named `DATABASE_PASSWORD` in your 1Password vault.

The backend will look for a field labeled "password" in the item and use its value as the secret.

## Security Considerations

- Credentials are stored securely in the macOS Keychain, not in plain text configuration files
- Limit the permissions of your service account to only the vault containing your secrets
- Regularly rotate your service account token
- Use the `imbued credentials set-onepass` command to update your credentials when they change

## Troubleshooting

If you encounter issues with the 1Password backend:

1. Verify that the 1Password CLI is installed and working by running `op --version`
2. Check that your credentials are stored in the keychain with `imbued credentials check-onepass`
3. If credentials are missing, set them with `imbued credentials set-onepass`
4. Verify that the item names in your 1Password vault match the secret names in your Imbued configuration
5. Check that each item has a field labeled "password"

For more information, see the [1Password CLI documentation](https://developer.1password.com/docs/cli/).
