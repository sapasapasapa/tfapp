package errors

import (
	"fmt"
	"os"
	"strings"

	"tfapp/internal/ui"
)

// DisplayError formats and displays an error message to the user.
// It formats different error types appropriately and uses color coding.
func DisplayError(err error) {
	if err == nil {
		return
	}

	errMsg := err.Error()

	switch {
	case IsValidationError(err):
		// Validation errors are shown in yellow
		fmt.Fprintf(os.Stderr, "%sValidation Error:%s %s\n", ui.ColorYellow, ui.ColorReset, errMsg)

	case IsUserInteractionError(err):
		// User interaction errors are shown in yellow
		fmt.Fprintf(os.Stderr, "%sInput Error:%s %s\n", ui.ColorYellow, ui.ColorReset, errMsg)

	case IsConfigurationError(err):
		// Configuration errors are shown in red
		fmt.Fprintf(os.Stderr, "%sConfiguration Error:%s %s\n", ui.ColorRed, ui.ColorReset, errMsg)

	case IsErrUserAborted(err):
		// User aborted operations are shown in yellow
		fmt.Fprintf(os.Stderr, "%sOperation Aborted:%s %s\n", ui.ColorYellow, ui.ColorReset, errMsg)

	case strings.Contains(errMsg, "Planning failed"):
		// Planning errors are shown entirely in red
		fmt.Fprintf(os.Stderr, "%s%s%s\n\n", ui.ColorRed, errMsg, ui.ColorReset)

	default:
		// All other errors are shown in red
		fmt.Fprintf(os.Stderr, "%sError:%s %s\n", ui.ColorRed, ui.ColorReset, errMsg)
	}
}

// ExitWithError displays an error and exits with non-zero status code.
func ExitWithError(err error, code int) {
	DisplayError(err)
	os.Exit(code)
}
