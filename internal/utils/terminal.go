// Package utils provides general utility functions.
package utils

import (
	"os"
	"os/exec"
	"runtime"
)

// ClearTerminal clears the terminal screen based on the current operating system.
func ClearTerminal() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}
