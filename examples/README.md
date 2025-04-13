# Imbued Examples

This directory contains examples of how to use Imbued for managing secrets in your development environment.

## Directory Structure

- `.imbued` - Example Imbued configuration file
- `imbued-daemon.plist` - Example launchd plist file for running Imbued as a daemon on macOS
- `secrets/` - Directory containing example secret files
  - `secrets.env` - Example environment file with secrets

## Using the Examples

### 1. Set up the Secrets

First, create a secrets file based on the example in `secrets/secrets.env`. You can either:

- Copy the example file and replace the values with your actual secrets:
  ```bash
  cp secrets/secrets.env ~/.imbued/secrets.env
  # Edit ~/.imbued/secrets.env with your actual secrets
  ```
- Or create a new secrets file from scratch following the format in the example

### 2. Configure Imbued

Create an `.imbued` file in your project directory based on the example:

```bash
cp .imbued /path/to/your/project/.imbued
```

Edit the `.imbued` file to point to your secrets file and configure which secrets you want to use:

```toml
# Number of child directories down that secrets are available for
valid_depth = 2

# Type of secret backend to use
backend_type = "env_file"

# Backend-specific configuration
[backend_config]
# Update this path to point to your actual secrets file
file_path = "/path/to/your/secrets.env"

# Secrets to retrieve
# Format: secret_name = "environment_variable_name"
[secrets]
# Include only the secrets you need
DATABASE_PASSWORD = "DB_PASSWORD"
API_KEY = "API_KEY"
```

### 3. Run Imbued as a Daemon (Optional)

If you want to run Imbued as a daemon, you can use the example launchd plist file:

```bash
# Copy the plist file to the LaunchAgents directory
cp imbued-daemon.plist ~/Library/LaunchAgents/com.example.imbued.plist

# Edit the plist file if needed
# For example, you might want to change the socket path or log file location

# Load the launchd service
launchctl load ~/Library/LaunchAgents/com.example.imbued.plist
```

### 4. Set Up Shell Integration

Follow the instructions in the main README to set up shell integration for your preferred shell (Bash or Fish).

### 5. Test the Integration

Navigate to your project directory where you placed the `.imbued` file:

```bash
cd /path/to/your/project
```

The shell integration should automatically detect the `.imbued` file and inject the secrets into your environment variables.

You can verify that the secrets are available by running:

```bash
echo $DB_PASSWORD
echo $API_KEY
```

When you leave the directory, the environment variables should be automatically removed.

## Troubleshooting

If you encounter issues with the examples:

1. Check the Imbued log file:
   ```bash
   cat ~/.imbued/logs/imbued.log
   ```

2. Run Imbued in direct mode with verbose logging:
   ```bash
   imbued --verbose --show-config
   ```

3. Verify that your `.imbued` file is correctly formatted and points to a valid secrets file

4. Ensure that the shell integration is correctly set up in your shell configuration file

## Additional Examples

### Using HashiCorp Vault

To use HashiCorp Vault as a secrets backend, modify your `.imbued` file:

```toml
backend_type = "vault"

[backend_config]
address = "https://vault.example.com:8200"
token = "hvs.example"

[secrets]
DB_PASSWORD = "secret/data/myapp/database#password"
API_KEY = "secret/data/myapp/api#key"
```

### Using 1Password

To use 1Password as a secrets backend:

```toml
backend_type = "onepass"

[backend_config]
account_token = "example-token"
vault_id = "example-vault"

[secrets]
DB_PASSWORD = "Database/MyApp#password"
API_KEY = "API/MyApp#key"
```

### Using AWS Secret Manager

To use AWS Secret Manager:

```toml
backend_type = "aws_secret_manager"

[backend_config]
region = "us-west-2"
access_key = "AKIAEXAMPLE"
secret_key = "example-secret-key"

[secrets]
DB_PASSWORD = "myapp/database#password"
API_KEY = "myapp/api#key"
```

### Using GCP Secret Manager

To use GCP Secret Manager:

```toml
backend_type = "gcp_secret_manager"

[backend_config]
project_id = "example-project"
credentials = "/path/to/credentials.json"

[secrets]
DB_PASSWORD = "projects/example-project/secrets/database-password/versions/latest"
API_KEY = "projects/example-project/secrets/api-key/versions/latest"
```
