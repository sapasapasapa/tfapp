// Package cli provides command-line interface functionality.
package cli

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	apperrors "tfapp/internal/errors"
	"tfapp/internal/ui"
)

// Flags represents the command-line flags for the application.
type Flags struct {
	Init            bool
	InitUpgrade     bool
	Help            bool
	AdditionalFlags []string
}

// ParseFlags parses the command-line flags and returns a Flags struct.
func ParseFlags() *Flags {
	// Define command-line flags
	init := flag.Bool("init", false, "Run terraform init before planning")
	initUpgrade := flag.Bool("init-upgrade", false, "Run terraform init -upgrade before planning")
	help := flag.Bool("help", false, "Display help information")

	// Parse the flags
	flag.Parse()

	// Create the Flags struct
	flags := &Flags{
		Init:            *init,
		InitUpgrade:     *initUpgrade,
		Help:            *help,
		AdditionalFlags: flag.Args(),
	}

	// Display help if requested
	if flags.Help {
		DisplayHelp()
		os.Exit(0)
	}

	// Validate the flags
	if err := validateFlags(flags); err != nil {
		apperrors.ExitWithError(err, 1)
	}

	return flags
}

// DisplayHelp shows the application usage and help information
func DisplayHelp() {
	fmt.Printf("%s%sTFApp - Enhanced Terraform Experience%s\n\n", ui.ColorInfo, ui.TextBold, ui.ColorReset)

	fmt.Println("USAGE:")
	fmt.Printf("  tfapp [tfapp-flags] -- [terraform-arguments]\n\n")

	fmt.Println("FLAGS:")
	fmt.Printf("  %-20s %s\n", "-init", "Run terraform init before creating a plan")
	fmt.Printf("  %-20s %s\n", "-init-upgrade", "Run terraform init -upgrade to update modules and providers")
	fmt.Printf("  %-20s %s\n\n", "-help, --help", "Display this help information")

	fmt.Println("EXAMPLES:")
	fmt.Printf("  # Basic usage\n")
	fmt.Printf("  tfapp\n\n")

	fmt.Printf("  # Initialize before planning\n")
	fmt.Printf("  tfapp -init\n\n")

	fmt.Printf("  # Initialize with module updates\n")
	fmt.Printf("  tfapp -init-upgrade\n\n")

	fmt.Printf("  # Pass variables to terraform\n")
	fmt.Printf("  tfapp -- -var=\"environment=production\" -var=\"region=us-west-2\"\n\n")

	fmt.Printf("  # Use a variable file\n")
	fmt.Printf("  tfapp -- -var-file=production.tfvars\n\n")

	fmt.Printf("  # Use auto-approval (non-interactive mode)\n")
	fmt.Printf("  tfapp -- -auto-approve\n\n")

	fmt.Println("")

	fmt.Printf("For more detailed information, please see the documentation at: %s%shttps://github.com/sapasapasapa/tfapp/tree/refactor/refactor-project/docs%s\n",
		ui.ColorInfo, ui.TextUnderline, ui.ColorReset)
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
