# Configuration Guide

TFApp allows extensive customization through its configuration file. These settings control appearance, behavior, and UI elements.

## Configuration File Location

The configuration file is stored at:
```
~/.config/tfapp/config.yaml
```

A default configuration file is created automatically on the first run if it doesn't exist.

## Configuration Format

TFApp uses YAML for its configuration. The file is organized into sections for different aspects of the application:

```yaml
colors:
  # Color settings
  info: "#36c"
  # ...

ui:
  # UI component settings
  spinner_type: "MiniDot"
  # ...
```

## Color Configuration

The `colors` section customizes the colors used throughout the application. Each setting accepts:
- Hex color codes (e.g., `#FF0000`)
- Named colors (e.g., `red`, `blue`)

```yaml
colors:
  # Color values can be hex codes ('#36c') or named colors
  info: "#36c"       # Informational messages (cyan/blue)
  success: "#2a2"    # Success messages (green)
  warning: "#fa0"    # Warning messages (yellow/orange)
  error: "#f33"      # Error messages (red)
  highlight: "#83f"  # Highlighted elements (purple)
  faint: "#777"      # Less important text (gray)
```

### Color Usage

| Setting | Purpose | Default | Used For |
|---------|---------|---------|----------|
| `info` | Informational text | `#36c` (blue) | Status messages, general information |
| `success` | Success indicators | `#2a2` (green) | Completion messages, successful operations |
| `warning` | Warning messages | `#fa0` (yellow) | Non-critical warnings, cautions |
| `error` | Error messages | `#f33` (red) | Critical errors, failures |
| `highlight` | Highlighted elements | `#83f` (purple) | Important information, key data points |
| `faint` | De-emphasized text | `#777` (gray) | Less important details, secondary information |

## UI Configuration

The `ui` section controls how interactive UI elements behave and appear:

```yaml
ui:
  # For spinner_type, available options are:
  # MiniDot, Dot, Line, Jump, Pulse, Points, Globe, Moon, Monkey, Meter
  spinner_type: "MiniDot"
  cursor_char: ">"   # Character used for selection cursor
```

### UI Settings

| Setting | Purpose | Default | Options |
|---------|---------|---------|---------|
| `spinner_type` | Loading animation style | `MiniDot` | `MiniDot`, `Dot`, `Line`, `Jump`, `Pulse`, `Points`, `Globe`, `Moon`, `Monkey`, `Meter` |
| `cursor_char` | Character for menu selection | `>` | Any character |

## Spinner Types

TFApp uses the [Charm](https://charm.sh/) library for spinners. Available options:

| Type | Description | Appearance |
|------|-------------|------------|
| `MiniDot` | Minimal spinning dot | A simple rotating dot |
| `Dot` | Standard dot spinner | A larger dot animation |
| `Line` | Line-based spinner | A rotating line |
| `Jump` | Bouncing animation | Characters that bounce up and down |
| `Pulse` | Pulsing animation | A pulsing element |
| `Points` | Multiple dots | Multiple animated dots |
| `Globe` | Rotating globe | A globe-like animation |
| `Moon` | Moon phases | Cycles through moon phases |
| `Monkey` | Monkey animation | An animated monkey face |
| `Meter` | Progress meter | A horizontal progress indicator |

## Advanced Configuration

### Multiple Configuration Profiles

While not directly supported in the application, you can maintain multiple configuration files and use symbolic links to switch between them:

```bash
# Create different configuration files
cp ~/.config/tfapp/config.yaml ~/.config/tfapp/config-dark.yaml
cp ~/.config/tfapp/config.yaml ~/.config/tfapp/config-light.yaml

# Edit the files with different color schemes
vim ~/.config/tfapp/config-dark.yaml
vim ~/.config/tfapp/config-light.yaml

# Switch between them by creating a symbolic link
ln -sf ~/.config/tfapp/config-dark.yaml ~/.config/tfapp/config.yaml
# or
ln -sf ~/.config/tfapp/config-light.yaml ~/.config/tfapp/config.yaml
```

## Troubleshooting Configuration

If TFApp encounters issues with your configuration:

1. It will display a warning message with details about the problem
2. Continue running with default settings
3. You can fix the configuration file and restart the application

### Common Configuration Issues

- **Invalid YAML syntax**: Ensure your YAML is properly formatted
- **Unknown color names**: Use hex codes if named colors don't work
- **Invalid spinner type**: Check the spelling of spinner types

### Resetting to Default Configuration

If your configuration becomes corrupted or you want to start fresh:

```bash
# Remove the existing configuration
rm ~/.config/tfapp/config.yaml

# Run tfapp to generate a new default configuration
tfapp
``` 
