# tfapp - Terraform Interface

[![Go Version](https://img.shields.io/badge/Go-1.24-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

A user-friendly interface for managing Terraform infrastructure with enhanced features like colorized output, interactive selection, and targeted applies.

![TFApp Demo](https://via.placeholder.com/800x450.png?text=TFApp+Demo+Image)

## Features

- 🚀 **Interactive Interface**: Navigate your Terraform workflows with intuitive menus
- 🎯 **Resource Targeting**: Select specific resources for targeted applies
- 🌈 **Colorized Output**: Easily understand Terraform outputs with color-coded messages
- 🔄 **Initialization Support**: Initialize and upgrade Terraform modules with simple flags
- ⚙️ **Customizable UI**: Personalize colors and UI elements through configuration
- 💻 **Interactive Confirmation**: Maintains Terraform's interactive confirmation prompts
- 📊 **Plan Visualization**: Better visualization of plan outputs

## Installation

### Requirements

- Go 1.24 or later
- Terraform CLI installed and available in PATH

### Install from Binary

Download the latest release from the GitHub releases page and place it in your PATH.

### Build from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/tfapp.git
cd tfapp

# Build the application
go build -o build/tfapp ./cmd/tfapp

# Make it available in your PATH (optional)
sudo cp build/tfapp /usr/local/bin/
```

## Basic Usage

At its simplest, just run `tfapp` in a directory containing Terraform configuration:

```bash
cd /path/to/terraform/project
tfapp
```

### Command-line Flags

| Flag | Description |
|------|-------------|
| `-init` | Run `terraform init` before creating a plan |
| `-init-upgrade` | Run `terraform init -upgrade` to update modules and providers |

### Pass-through Arguments

You can pass additional arguments to the underlying Terraform commands:

```bash
# Pass variables to terraform
tfapp -- -var="environment=production" -var="region=us-west-2"

# Specify a different state file
tfapp -- -state=custom.tfstate
```

### Interactive Menu

After generating a plan, you'll be presented with an interactive menu:

- `Apply Plan`: Apply the generated plan
- `Show Full Plan`: Display the complete Terraform plan details
- `Do a target apply`: Select specific resources to apply changes to
- `Exit`: Exit without applying changes

## Configuration

TFApp can be customized through a YAML configuration file.

### Location

```
~/.config/tfapp/config.yaml
```

A default configuration file will be created on first run if it doesn't exist.

### Configuration Options

```yaml
colors:
  # Color values can be hex codes ('#36c') or named colors
  info: "#36c"       # Informational messages (cyan/blue)
  success: "#2a2"    # Success messages (green)
  warning: "#fa0"    # Warning messages (yellow/orange)
  error: "#f33"      # Error messages (red)
  highlight: "#83f"  # Highlighted elements (purple)
  faint: "#777"      # Less important text (gray)

ui:
  # For spinner_type, available options are:
  # MiniDot, Dot, Line, Jump, Pulse, Points, Globe, Moon, Monkey, Meter
  spinner_type: "MiniDot"
  cursor_char: ">"   # Character used for selection cursor
```

## Advanced Features

### Targeted Applies

Select specific resources to apply using the interactive checkbox menu:

1. Run `tfapp`
2. Choose `Do a target apply` from the menu
3. Use space to select resources and Enter to confirm your selection
4. Review the targeted plan and apply if desired

### Custom Workflow Examples

#### Development Workflow with Variable Files

```bash
# Initialize with latest modules and apply with dev variables
tfapp -init-upgrade -- -var-file=dev.tfvars
```

#### CI/CD Pipeline Usage

```bash
# Non-interactive mode with automatic approval (for CI/CD)
tfapp -- -auto-approve
```

#### Working with Workspaces

```bash
# Create and select a new workspace before planning
terraform workspace new staging
tfapp -- -var-file=staging.tfvars
```

#### Using with Remote State

```bash
# Initialize with backend configuration
tfapp -init -- -backend-config=backend.hcl
```

## Project Structure

For developers interested in contributing or understanding the codebase:

```
tfapp/
├── cmd/              # Application entry points
│   └── tfapp/        # Main application
├── internal/         # Internal packages
│   ├── cli/          # Command-line interface
│   ├── config/       # Configuration management
│   ├── errors/       # Error handling
│   ├── models/       # Domain models
│   ├── terraform/    # Terraform operations
│   ├── ui/           # User interface components
│   └── utils/        # Utility functions
├── build/            # Build artifacts
├── go.mod            # Go module definition
└── README.md         # This file
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Development Setup

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Built with [Go](https://go.dev/)
- UI components powered by [Charm](https://charm.sh/) libraries
- Inspired by the Terraform community