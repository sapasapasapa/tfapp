// Package cli provides command-line interface functionality.
package cli

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	apperrors "tfapp/internal/errors"
	"tfapp/internal/ui"
	"tfapp/internal/version"
)

// Flags represents the command-line flags for the application.
type Flags struct {
	Init            bool
	InitUpgrade     bool
	Version         bool
	Help            bool
	AdditionalFlags []string
}

// ParseFlags parses the command-line flags and returns a Flags struct.
func ParseFlags() *Flags {
	// Define command-line flags
	init := flag.Bool("init", false, "Run terraform init before planning")
	initUpgrade := flag.Bool("init-upgrade", false, "Run terraform init -upgrade before planning")
	showVersion := flag.Bool("version", false, "Show version information and exit")
	help := flag.Bool("help", false, "Display help information")

	// Create custom usage function
	flag.Usage = func() {
		DisplayHelp()
	}

	// Parse the flags
	flag.Parse()

	// Handle --version manually (Go's flag package only does single dash by default)
	hasLongVersion := false
	for _, arg := range os.Args[1:] {
		if arg == "--version" {
			hasLongVersion = true
			break
		}
	}

	// Show help if requested
	if *help {
		DisplayHelp()
		os.Exit(0)
	}

	// Show version if requested with either -version or --version
	if *showVersion || hasLongVersion {
		fmt.Println(version.Info())
		os.Exit(0)
	}

	// Create the Flags struct
	flags := &Flags{
		Init:            *init,
		InitUpgrade:     *initUpgrade,
		Version:         *showVersion || hasLongVersion,
		Help:            *help,
		AdditionalFlags: flag.Args(),
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
	fmt.Printf("Version: %s\n\n", version.Full())

	fmt.Println("USAGE:")
	fmt.Printf("  tfapp [tfapp-flags] -- [terraform-arguments]\n\n")

	fmt.Println("FLAGS:")
	fmt.Printf("  %-20s %s\n", "-init", "Run terraform init before creating a plan")
	fmt.Printf("  %-20s %s\n", "-init-upgrade", "Run terraform init -upgrade to update modules and providers")
	fmt.Printf("  %-20s %s\n", "-version, --version", "Show version information and exit")
	fmt.Printf("  %-20s %s\n\n", "-help, --help", "Display this help information")

	fmt.Println("EXAMPLES:")
	fmt.Printf("  # Basic usage\n")
	fmt.Printf("  tfapp\n\n")

	fmt.Printf("  # Initialize before planning\n")
	fmt.Printf("  tfapp -init\n\n")

	fmt.Printf("  # Initialize with module updates\n")
	fmt.Printf("  tfapp -init-upgrade\n\n")

	fmt.Printf("  # Show version information\n")
	fmt.Printf("  tfapp --version\n\n")

	fmt.Printf("  # Pass variables to terraform\n")
	fmt.Printf("  tfapp -- -var=\"environment=production\" -var=\"region=us-west-2\"\n\n")

	fmt.Printf("  # Use a variable file\n")
	fmt.Printf("  tfapp -- -var-file=production.tfvars\n\n")

	fmt.Printf("  # Use auto-approval (non-interactive mode)\n")
	fmt.Printf("  tfapp -- -auto-approve\n\n")

	fmt.Println("")

	fmt.Printf("For more detailed information, please see the documentation at: %s%shttps://github.com/sapasapasapa/tfapp/tree/master/docs%s\n",
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
