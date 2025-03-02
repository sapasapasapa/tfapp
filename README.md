# tfapp - Enhanced Terraform Experience

[![Go Version](https://img.shields.io/badge/Go-1.24-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

A delightful, feature-rich interface for Terraform that makes infrastructure management more intuitive and efficient.

![TFApp Demo](https://via.placeholder.com/800x450.png?text=TFApp+Demo+Image)

## ✨ Key Features

- 🚀 **Interactive Interface** - Navigate infrastructure workflows with intuitive menus
- 🎯 **Resource Targeting** - Select specific resources for targeted applies
- 🌈 **Colorized Output** - Clear, color-coded information for better readability
- ⚙️ **Customizable UI** - Personalize colors and UI elements to match your preferences

## 🚀 Quick Start

```bash
# Make sure you have Go and Terraform installed
# Go 1.24+ required

# Install from source
git clone https://github.com/yourusername/tfapp.git
cd tfapp
go build -o build/tfapp ./cmd/tfapp
sudo cp build/tfapp /usr/local/bin/

# Use it in your Terraform project
cd /path/to/terraform/project
tfapp
```

## 📚 Documentation

For detailed information, check our documentation:

- [Installation Guide](docs/installation.md) - Detailed installation instructions
- [Usage Guide](docs/usage.md) - How to use TFApp effectively
- [Configuration](docs/configuration.md) - Customizing TFApp to your preferences
- [Advanced Features](docs/advanced.md) - Power user features and techniques
- [Development](docs/development.md) - Contributing to TFApp

## 🖼️ Screenshots

<table>
  <tr>
    <td width="50%">
      <img src="https://via.placeholder.com/400x300.png?text=Interactive+Menu" alt="Interactive Menu" />
      <p align="center"><em>Interactive Menu</em></p>
    </td>
    <td width="50%">
      <img src="https://via.placeholder.com/400x300.png?text=Resource+Selection" alt="Resource Selection" />
      <p align="center"><em>Resource Selection</em></p>
    </td>
  </tr>
  <tr>
    <td width="50%">
      <img src="https://via.placeholder.com/400x300.png?text=Colorized+Output" alt="Colorized Output" />
      <p align="center"><em>Colorized Output</em></p>
    </td>
    <td width="50%">
      <img src="https://via.placeholder.com/400x300.png?text=Plan+Overview" alt="Plan Overview" />
      <p align="center"><em>Plan Overview</em></p>
    </td>
  </tr>
</table>

## 🔄 Version Compatibility

- Terraform: 0.12.x and above
- Go: 1.24 and above
- Platforms: Linux, macOS, Windows

## 👥 Contributing

Contributions are welcome! Check out our [Development Guide](docs/development.md) to get started.

## 📜 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- Built with [Go](https://go.dev/)
- UI components powered by [Charm](https://charm.sh/) libraries
- Inspired by the Terraform community