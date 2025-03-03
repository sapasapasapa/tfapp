// Package version contains version information for the tfapp application.
package version

import (
	"fmt"
)

var (
	// Version holds the current version set by ldflags during build
	Version = "dev"
	// Commit holds the git commit hash set by ldflags during build
	Commit = "none"
	// Date holds the build date set by ldflags during build
	Date = "unknown"
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
// If Version is set via ldflags, it will use that value
// Otherwise, it will use the hardcoded version numbers
func Full() string {
	if Version != "dev" {
		return Version
	}
	return fmt.Sprintf("%d.%d.%d", Major, Minor, Patch)
}

// Info returns detailed version information as a multi-line string
func Info() string {
	if Version != "dev" {
		return fmt.Sprintf("%s\nCommit: %s\nBuilt: %s", Version, Commit, Date)
	}
	return Full()
}

// ShortInfo returns a short version string suitable for CLI output
func ShortInfo() string {
	return Full()
}
