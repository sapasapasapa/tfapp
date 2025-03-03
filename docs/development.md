# Development Guide

This guide is intended for developers who want to contribute to the TFApp project or customize it for their own needs.

## Project Structure

TFApp follows a modular Go project structure:

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
├── docs/             # Documentation
├── go.mod            # Go module definition
├── go.sum            # Go module checksums
└── README.md         # Project overview
```

## Setting Up Development Environment

### Prerequisites

- Go 1.24 or later
- Terraform CLI installed
- Git

### Clone the Repository

```bash
git clone https://github.com/yourusername/tfapp.git
cd tfapp
```

### Install Dependencies

```bash
go mod download
```

### Build for Development

```bash
go build -o build/tfapp ./cmd/tfapp
```

### Development Build with Debug Information

```bash
go build -gcflags="all=-N -l" -o build/tfapp ./cmd/tfapp
```

## Core Components

### Command Executor

The `CommandExecutor` in `internal/terraform/exec.go` handles executing Terraform commands. It's responsible for:
- Starting processes
- Handling input/output
- Displaying spinners during execution
- Error handling

### Models

Domain models in `internal/models/` define the core interfaces and types:
- `Resource`: Represents a Terraform resource
- `Executor`: Interface for executing commands
- `PlanService`: Interface for plan operations
- `ApplyService`: Interface for apply operations

### UI Components

UI components in `internal/ui/` provide interactive elements:
- `menu`: Interactive selection menu
- `checkbox`: Multi-selection for resource targeting
- `spinner`: Loading animations
- `colors`: Color management for output

### Configuration Management

The `config` package manages user configuration:
- Loading and saving configuration
- Default settings
- YAML parsing

## Adding New Features

### Adding a New Command-Line Flag

To add a new command-line flag, modify `internal/cli/flags.go`:

```go
// Define the flag in the Flags struct
type Flags struct {
    Init            bool
    InitUpgrade     bool
    YourNewFlag     bool
    AdditionalFlags []string
}

// Add the flag in ParseFlags()
func ParseFlags() *Flags {
    // Define command-line flags
    init := flag.Bool("init", false, "Run terraform init before planning")
    initUpgrade := flag.Bool("init-upgrade", false, "Run terraform init -upgrade before planning")
    yourNewFlag := flag.Bool("your-flag", false, "Description of your new flag")

    // Parse the flags
    flag.Parse()

    // Create the Flags struct
    flags := &Flags{
        Init:            *init,
        InitUpgrade:     *initUpgrade,
        YourNewFlag:     *yourNewFlag,
        AdditionalFlags: flag.Args(),
    }

    // Validate the flags
    if err := validateFlags(flags); err != nil {
        apperrors.ExitWithError(err, 1)
    }

    return flags
}
```

### Adding a New Menu Option

To add a new option to the main menu, modify `internal/ui/menu/menu.go`:

1. Add the new option to the options list
2. Handle the new selection in the menu handler

### Adding a New Terraform Command

To add support for a new Terraform command:

1. Create a new service interface in `internal/models/terraform.go`
2. Implement the service in `internal/terraform/`
3. Add relevant UI handling in `internal/cli/commands.go`

## Building 

### Building for Multiple Platforms

```bash
# Build for Linux
GOOS=linux GOARCH=amd64 go build -o build/tfapp-linux-amd64 ./cmd/tfapp

# Build for macOS
GOOS=darwin GOARCH=amd64 go build -o build/tfapp-darwin-amd64 ./cmd/tfapp

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o build/tfapp-windows-amd64.exe ./cmd/tfapp
```

## Style Guide

### Code Formatting

Format your code using gofmt:

```bash
gofmt -w .
```

### Code Style

- Follow Go's official [style guidelines](https://golang.org/doc/effective_go)
- Use descriptive variable and function names
- Add comments for public functions and types
- Keep functions small and focused

### Error Handling

Use the error package in `internal/errors/` for consistent error handling:

```go
import apperrors "tfapp/internal/errors"

// Instead of returning generic errors
if err != nil {
    return fmt.Errorf("something went wrong: %w", err)
}

// Use the specialized error types
if err != nil {
    return apperrors.NewValidationError(
        "component-name",
        "Description of what went wrong",
        err,
    )
}
```

## Customizing for Your Organization

### Custom Branding

Modify the UI elements in `internal/ui/colors.go` to use your organization's branding:

```go
// Default colors
func DefaultConfig() *Config {
    return &Config{
        Colors: ColorConfig{
            Info:      "#36c", // Replace with your brand colors
            Success:   "#2a2",
            Warning:   "#fa0",
            Error:     "#f33",
            Highlight: "#83f",
            Faint:     "#777",
        },
        // ...
    }
}
```

### Adding Support for Custom Terraform Commands

If your organization uses custom Terraform commands or workflows, extend the application by adding new services in the `terraform` package.

## Contributing Guidelines

When contributing to this project, please follow these guidelines:

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Ensure all tests pass
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

Please include tests for new features and ensure all existing tests pass. 