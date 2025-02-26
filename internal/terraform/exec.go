// Package terraform provides functionality for executing Terraform commands.
package terraform

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"tfapp/internal/models"
	"tfapp/internal/ui/spinner"
)

// CommandExecutor handles executing Terraform commands.
type CommandExecutor struct {
	// Add fields if needed in the future for configuration
}

// NewCommandExecutor creates a new Terraform command executor.
func NewCommandExecutor() *CommandExecutor {
	return &CommandExecutor{}
}

// RunCommand executes a terraform command with the given arguments.
// If redirectOutput is true, the command's output will be redirected to stdout/stderr.
// Otherwise, it captures the output and returns any errors that occurred.
func (e *CommandExecutor) RunCommand(ctx interface{}, args []string, spinnerMsg string, redirectOutput bool) error {
	ctxTyped, ok := ctx.(context.Context)
	if !ok {
		return fmt.Errorf("context type assertion failed")
	}

	cmd := exec.CommandContext(ctxTyped, "terraform", args...)
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
			return fmt.Errorf("error creating stdout pipe: %w", err)
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("error creating stderr pipe: %w", err)
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

	s := spinner.New(spinnerMsg)
	s.Start()

	err := cmd.Start()
	if err != nil {
		s.Stop()
		return fmt.Errorf("error starting terraform command: %w", err)
	}

	if !redirectOutput {
		wg.Wait() // Wait for output copying to complete
	}

	err = cmd.Wait()
	s.Stop()

	if err != nil && !redirectOutput {
		// Include both stdout and stderr in the error message
		return fmt.Errorf("%s\n%s: %w", stdout.String(), stderr.String(), err)
	}

	return err
}

// Ensure CommandExecutor implements the models.Executor interface
var _ models.Executor = (*CommandExecutor)(nil)
