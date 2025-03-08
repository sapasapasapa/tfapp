// Package ui provides user interface components and utilities.
package ui

import (
	"fmt"
	"strconv"
	"strings"

	"tfapp/internal/config"
)

// Color constants for terminal output.
const (
	ColorReset = "\033[0m"
	TextBold   = "\033[1m"
)

var (
	// Default colors - will be overridden from config
	ColorError   = "\033[1;31m"
	ColorSuccess = "\033[32m"
	ColorWarning = "\033[33m"
	ColorInfo    = "\033[36m"

	// Additional stored colors
	ColorHighlight = "\033[38;2;130;57;243m"  // Purple for highlights (#8239F3)
	ColorFaint     = "\033[38;2;119;119;119m" // Gray for less important text (#777)

	// Store the loaded config
	appConfig *config.Config

	// Text formatting constants
	TextUnderline = "\033[4m" // ANSI escape sequence for underlined text
)

// InitColors initializes the colors from the provided configuration.
func InitColors(cfg *config.Config) {
	appConfig = cfg

	// Update the color variables based on the configuration
	ColorError = parseColorToAnsi(cfg.Colors.Error)
	ColorSuccess = parseColorToAnsi(cfg.Colors.Success)
	ColorWarning = parseColorToAnsi(cfg.Colors.Warning)
	ColorInfo = parseColorToAnsi(cfg.Colors.Info)
	ColorHighlight = parseColorToAnsi(cfg.Colors.Highlight)
	ColorFaint = parseColorToAnsi(cfg.Colors.Faint)
}

// parseColorToAnsi converts a hex color string to an ANSI color code.
func parseColorToAnsi(hexColor string) string {
	// Strip the leading # if present
	hexColor = strings.TrimPrefix(hexColor, "#")

	// Handle simple 3-character hex colors
	if len(hexColor) == 3 {
		r := hexColor[0:1]
		g := hexColor[1:2]
		b := hexColor[2:3]
		hexColor = r + r + g + g + b + b
	}

	// Parse the hex values
	if len(hexColor) != 6 {
		// Fall back to default if invalid
		return "\033[37m" // White as fallback
	}

	r, _ := strconv.ParseInt(hexColor[0:2], 16, 0)
	g, _ := strconv.ParseInt(hexColor[2:4], 16, 0)
	b, _ := strconv.ParseInt(hexColor[4:6], 16, 0)

	// Return the 24-bit color ANSI escape sequence
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

// GetHexColorByName returns the hex color string for use with lipgloss.
// This is preferred for lipgloss styling over GetColorByName which returns ANSI codes.
func GetHexColorByName(name string) string {
	if appConfig == nil {
		switch strings.ToLower(name) {
		case "highlight":
			return "#8239F3" // Purple for highlights
		case "faint":
			return "#777777" // Gray for less important text
		case "info":
			return "#36c" // Cyan/Blue
		case "success":
			return "#2a2" // Green
		case "warning":
			return "#fa0" // Yellow/Orange
		case "error":
			return "#f33" // Red
		default:
			return "" // No color
		}
	}

	switch strings.ToLower(name) {
	case "info":
		return appConfig.Colors.Info
	case "success":
		return appConfig.Colors.Success
	case "warning":
		return appConfig.Colors.Warning
	case "error":
		return appConfig.Colors.Error
	case "highlight":
		return appConfig.Colors.Highlight
	case "faint":
		return appConfig.Colors.Faint
	default:
		return ""
	}
}

// Colorize adds ANSI color codes to terraform plan output.
func Colorize(line string) string {
	if len(line) == 0 {
		return line
	}

	// Handle specific operation patterns more precisely
	// Destroy operations - red
	if strings.Contains(line, "will be destroyed") {
		return replaceIfContains(line, "will be destroyed", ColorError+"will be destroyed"+ColorReset)
	} else if strings.Contains(line, "destroyed") {
		line = replaceIfContains(line, "destroyed", ColorError+"destroyed"+ColorReset)
	}

	// Replace/recreate operations - red
	if strings.Contains(line, "must be replaced") {
		return replaceIfContains(line, "must be replaced", ColorError+"must be replaced"+ColorReset)
	} else if strings.Contains(line, "must be recreated") {
		return replaceIfContains(line, "must be recreated", ColorError+"must be recreated"+ColorReset)
	} else if strings.Contains(line, "replaced") {
		line = replaceIfContains(line, "replaced", ColorError+"replaced"+ColorReset)
	}

	// Create operations - green
	if strings.Contains(line, "will be created") {
		return replaceIfContains(line, "will be created", ColorSuccess+"will be created"+ColorReset)
	} else if strings.Contains(line, "created") {
		line = replaceIfContains(line, "created", ColorSuccess+"created"+ColorReset)
	}

	// Update operations - yellow
	if strings.Contains(line, "will be updated in-place") {
		return replaceIfContains(line, "will be updated in-place", ColorWarning+"will be updated in-place"+ColorReset)
	} else if strings.Contains(line, "updated in-place") {
		line = replaceIfContains(line, "updated in-place", ColorWarning+"updated in-place"+ColorReset)
	}

	// Read operations - blue
	if strings.Contains(line, "will be read during apply") {
		return replaceIfContains(line, "will be read during apply", ColorInfo+"will be read during apply"+ColorReset)
	}

	return line
}

// GetColorByName returns the ANSI color code for a named color.
func GetColorByName(name string) string {
	if appConfig == nil {
		switch strings.ToLower(name) {
		case "highlight":
			return ColorHighlight
		case "faint":
			return ColorFaint
		case "info":
			return ColorInfo
		case "success":
			return ColorSuccess
		case "warning":
			return ColorWarning
		case "error":
			return ColorError
		default:
			return ColorReset
		}
	}

	switch strings.ToLower(name) {
	case "info":
		return parseColorToAnsi(appConfig.Colors.Info)
	case "success":
		return parseColorToAnsi(appConfig.Colors.Success)
	case "warning":
		return parseColorToAnsi(appConfig.Colors.Warning)
	case "error":
		return parseColorToAnsi(appConfig.Colors.Error)
	case "highlight":
		return parseColorToAnsi(appConfig.Colors.Highlight)
	case "faint":
		return parseColorToAnsi(appConfig.Colors.Faint)
	default:
		return ColorReset
	}
}

// Helper function to replace text only if it contains the substring.
func replaceIfContains(text, substr, replacement string) string {
	if Contains(text, substr) {
		return Replace(text, substr, replacement)
	}
	return text
}

// Contains reports whether substr is within s.
func Contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Replace returns a copy of s with the first n non-overlapping instances of old
// replaced by new. If old is empty, it matches at the beginning of the string
// and after each UTF-8 sequence, yielding up to k+1 replacements for a k-rune
// string. If n < 0, there is no limit on the number of replacements.
func Replace(s, old, new string) string {
	return doReplace(s, old, new, 1)
}

func doReplace(s, old, new string, n int) string {
	if old == new || n == 0 {
		return s // avoid allocation
	}

	if len(old) == 0 {
		if len(s) == 0 {
			return new
		}
		result := make([]byte, len(s)*(len(new)+1))
		copy(result, new)
		j := len(new)
		for i := 0; i < len(s); i++ {
			result[j] = s[i]
			j++
			if n > 0 && j < len(result) && n > i+1 {
				copy(result[j:], new)
				j += len(new)
			}
		}
		return string(result[:j])
	}

	// Count occurrences of old.
	m := 0
	for i := 0; i < len(s)-len(old)+1; i++ {
		if s[i:i+len(old)] == old {
			m++
			i += len(old) - 1
			if m == n {
				break
			}
		}
	}

	if m == 0 {
		return s // avoid allocation
	}

	result := make([]byte, len(s)+(m*len(new))-m*len(old))
	j := 0
	for i := 0; i < len(s); {
		if i <= len(s)-len(old) && s[i:i+len(old)] == old {
			copy(result[j:], new)
			j += len(new)
			i += len(old)
			m--
			if m == 0 {
				copy(result[j:], s[i:])
				break
			}
		} else {
			result[j] = s[i]
			j++
			i++
		}
	}
	return string(result)
}

// GetSpinnerType returns the configured spinner type or the default.
func GetSpinnerType() string {
	if appConfig == nil {
		return "MiniDot" // Default spinner type
	}
	return appConfig.UI.SpinnerType
}

// GetCursorChar returns the configured cursor character or the default.
func GetCursorChar() string {
	if appConfig == nil {
		return ">" // Default cursor character
	}
	return appConfig.UI.CursorChar
}
