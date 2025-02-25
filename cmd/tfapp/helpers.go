package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
)

func clearTerminal() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func runTerraformCommand(args []string, spinnerMsg string, redirectOutput bool) error {
	cmd := exec.Command("terraform", args...)
	cmd.Stdin = os.Stdin

	var stdout, stderr bytes.Buffer
	var wg sync.WaitGroup

	if redirectOutput {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		// Set up pipes for capturing output
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("error creating stdout pipe: %v", err)
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("error creating stderr pipe: %v", err)
		}

		// Copy output to buffers
		wg.Add(2)
		go func() {
			defer wg.Done()
			io.Copy(&stdout, stdoutPipe)
		}()
		go func() {
			defer wg.Done()
			io.Copy(&stderr, stderrPipe)
		}()
	}

	spinner := NewSpinner(spinnerMsg)
	spinner.Start()

	err := cmd.Start()
	if err != nil {
		spinner.Stop()
		return fmt.Errorf("error starting command: %v", err)
	}

	if !redirectOutput {
		wg.Wait() // Wait for output copying to complete
	}

	err = cmd.Wait()
	spinner.Stop()

	if err != nil && !redirectOutput {
		// Include both stdout and stderr in the error message
		return fmt.Errorf("%s\n%s: %v", stdout.String(), stderr.String(), err)
	}

	return err
}
