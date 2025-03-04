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

## Contributing Guidelines

When contributing to this project, please follow these guidelines:

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Ensure all tests pass
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request
