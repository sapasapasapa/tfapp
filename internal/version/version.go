// Package version contains version information for the tfapp application.
package version

import (
	"fmt"
)

// Version information
const (
	// Major version component
	Major = 0
	// Minor version component
	Minor = 1
	// Patch version component
	Patch = 0
)

// Full returns the full version string
func Full() string {
	return fmt.Sprintf("%d.%d.%d", Major, Minor, Patch)
}

// Info returns detailed version information as a multi-line string
func Info() string {
	return Full()
}
