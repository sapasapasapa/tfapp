// Package cli provides command-line interface functionality.
package cli

import (
	"flag"
	"os"
	"os/exec"

	apperrors "tfapp/internal/errors"
)

// Flags represents the command-line flags for the application.
type Flags struct {
	Init            bool
	InitUpgrade     bool
	AdditionalFlags []string
}

// ParseFlags parses the command-line flags and returns a Flags struct.
func ParseFlags() *Flags {
	// Define command-line flags
	init := flag.Bool("init", false, "Run terraform init before planning")
	initUpgrade := flag.Bool("init-upgrade", false, "Run terraform init -upgrade before planning")

	// Parse the flags
	flag.Parse()

	// Create the Flags struct
	flags := &Flags{
		Init:            *init,
		InitUpgrade:     *initUpgrade,
		AdditionalFlags: flag.Args(),
	}

	// Validate the flags
	if err := validateFlags(flags); err != nil {
		apperrors.ExitWithError(err, 1)
	}

	return flags
}

// validateFlags checks if the combination of flags is valid.
func validateFlags(flags *Flags) error {
	// Check if init and init-upgrade are used together
	if flags.Init && flags.InitUpgrade {
		return apperrors.NewValidationError(
			"init-flags",
			"-init and -init-upgrade cannot be used together",
			apperrors.ErrInvalidInput,
		)
	}

	// Check if Terraform is installed
	if _, err := os.Stat("/usr/local/bin/terraform"); os.IsNotExist(err) {
		if _, err = os.Stat("/usr/bin/terraform"); os.IsNotExist(err) {
			// Check the PATH for terraform
			_, err := exec.LookPath("terraform")
			if err != nil {
				return apperrors.NewConfigurationError(
					"dependencies",
					"Terraform executable not found in PATH",
					err,
				)
			}
		}
	}

	return nil
}
