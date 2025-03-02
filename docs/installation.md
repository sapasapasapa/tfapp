# Installation Guide

## Requirements

### System Requirements

- **Go**: Version 1.24 or later
- **Terraform**: CLI installed and available in PATH
- **Operating Systems**: Compatible with Linux, macOS, and Windows

### Checking Requirements

Ensure Go is installed:
```bash
go version
```

Ensure Terraform is installed:
```bash
terraform version
```

## Installation Methods

### Install from Binary

1. Download the latest release from the GitHub releases page:
   ```bash
   curl -LO https://github.com/yourusername/tfapp/releases/latest/download/tfapp_$(uname -s)_$(uname -m).tar.gz
   ```

2. Extract the binary:
   ```bash
   tar -xzf tfapp_$(uname -s)_$(uname -m).tar.gz
   ```

3. Move the binary to a location in your PATH:
   ```bash
   sudo mv tfapp /usr/local/bin/
   ```

4. Make it executable (if needed):
   ```bash
   chmod +x /usr/local/bin/tfapp
   ```

5. Verify the installation:
   ```bash
   tfapp --version
   ```

### Build from Source

Building from source gives you the latest code and the ability to customize the build.

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/tfapp.git
   cd tfapp
   ```

2. Build the application:
   ```bash
   go build -o build/tfapp ./cmd/tfapp
   ```

3. (Optional) Install the binary to a location in your PATH:
   ```bash
   sudo cp build/tfapp /usr/local/bin/
   ```

4. Verify the installation:
   ```bash
   tfapp --version
   ```

### Using Go Install

If you're familiar with Go's toolchain, you can install directly using `go install`:

```bash
go install github.com/yourusername/tfapp/cmd/tfapp@latest
```

This will place the binary in your `$GOPATH/bin` directory, which should be in your PATH.

## Verifying Your Installation

After installation, run a simple command to verify everything is working properly:

```bash
tfapp -h
```

You should see the help text displaying available commands and options.

## Troubleshooting

### Common Issues

#### "Terraform executable not found in PATH"

This error occurs when tfapp cannot locate the terraform executable. Make sure terraform is installed and in your PATH:

```bash
which terraform
```

If it's not in your PATH, install Terraform following [the official Terraform installation guide](https://learn.hashicorp.com/tutorials/terraform/install-cli).

#### Permission Denied

If you encounter permission issues when running tfapp:

```bash
chmod +x /path/to/tfapp
```

#### Build Errors

If you encounter errors while building from source, ensure you have Go 1.24+ installed and try fetching dependencies explicitly:

```bash
go mod download
go build -o build/tfapp ./cmd/tfapp
``` 