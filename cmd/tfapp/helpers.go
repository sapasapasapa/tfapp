package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func runTerraformCommand(args []string, spinnerMsg string, redirectOutput bool) error {
	cmd := exec.Command("terraform", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdin = os.Stdin

	if redirectOutput {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	spinner := NewSpinner(spinnerMsg)
	spinner.Start()

	err := cmd.Run()
	spinner.Stop()

	if err != nil && !redirectOutput {
		return fmt.Errorf("%s: %v", stderr.String(), err)
	}

	return err
}
