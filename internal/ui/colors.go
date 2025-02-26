// Package ui provides user interface components and utilities.
package ui

// Color constants for terminal output.
const (
	ColorRed    = "\033[1;31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorReset  = "\033[0m"
	TextBold    = "\033[1m"
)

// Colorize adds ANSI color codes to terraform plan output.
func Colorize(line string) string {
	if len(line) == 0 {
		return line
	}

	line = replaceIfContains(line, "destroyed", ColorRed+"destroyed"+ColorReset)
	line = replaceIfContains(line, "replaced", ColorRed+"replaced"+ColorReset)
	line = replaceIfContains(line, "created", ColorGreen+"created"+ColorReset)
	line = replaceIfContains(line, "updated in-place", ColorYellow+"updated in-place"+ColorReset)
	return line
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
