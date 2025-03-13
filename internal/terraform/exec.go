// Package terraform provides functionality for executing Terraform commands.
package terraform

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"tfapp/internal/models"
	"tfapp/internal/ui/spinner"
)

// CommandExecutor handles executing Terraform commands.
type CommandExecutor struct {
	// Add fields if needed in the future for configuration
	progressCallbacks []ProgressCallback
}

// ProgressCallback is a function type that gets called with progress updates
type ProgressCallback func(status string)

// NewCommandExecutor creates a new Terraform command executor.
func NewCommandExecutor() *CommandExecutor {
	return &CommandExecutor{
		progressCallbacks: make([]ProgressCallback, 0),
	}
}

// RegisterProgressCallback registers a callback function to receive progress updates
func (e *CommandExecutor) RegisterProgressCallback(callback ProgressCallback) {
	e.progressCallbacks = append(e.progressCallbacks, callback)
}

// notifyProgress sends a status update to all registered callbacks
func (e *CommandExecutor) notifyProgress(status string) {
	for _, callback := range e.progressCallbacks {
		callback(status)
	}
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
	var progressWg sync.WaitGroup

	// Create pipes for real-time output processing
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("error creating stderr pipe: %w", err)
	}

	// Setup the multiplexing of outputs
	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()

	progressWg.Add(1)
	go func() {
		defer progressWg.Done()
		e.processOutputForProgress(stdoutReader, "stdout")
	}()

	progressWg.Add(1)
	go func() {
		defer progressWg.Done()
		e.processOutputForProgress(stderrReader, "stderr")
	}()

	// Set up output handling
	wg.Add(2)
	if redirectOutput {
		// Tee the output to both the console and our progress monitoring
		go func() {
			defer wg.Done()
			defer stdoutWriter.Close()
			mw := io.MultiWriter(os.Stdout, stdoutWriter, &stdout)
			io.Copy(mw, stdoutPipe)
		}()
		go func() {
			defer wg.Done()
			defer stderrWriter.Close()
			mw := io.MultiWriter(os.Stderr, stderrWriter, &stderr)
			io.Copy(mw, stderrPipe)
		}()
	} else {
		// Capture output for our buffers and progress monitoring
		go func() {
			defer wg.Done()
			defer stdoutWriter.Close()
			mw := io.MultiWriter(stdoutWriter, &stdout)
			io.Copy(mw, stdoutPipe)
		}()
		go func() {
			defer wg.Done()
			defer stderrWriter.Close()
			mw := io.MultiWriter(stderrWriter, &stderr)
			io.Copy(mw, stderrPipe)
		}()
	}

	// Start an enhanced spinner with status updates
	s := spinner.New(spinnerMsg)
	s.Start()

	// Start the command
	e.notifyProgress(fmt.Sprintf("Starting terraform %s", strings.Join(args, " ")))
	err = cmd.Start()
	if err != nil {
		s.Stop()
		return fmt.Errorf("error starting terraform command: %w", err)
	}

	// Start a goroutine to periodically update the spinner message with status
	statusCtx, statusCancel := context.WithCancel(ctxTyped)
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		counter := 0
		for {
			select {
			case <-statusCtx.Done():
				return
			case <-ticker.C:
				counter++
				s.UpdateMessage(fmt.Sprintf("%s (running for %ds)", spinnerMsg, counter*3))
			}
		}
	}()

	// Wait for the command to finish
	cmdErr := cmd.Wait()

	// Stop the status updates
	statusCancel()

	// Wait for the output processing to finish
	wg.Wait()

	// Close the readers to signal the progress processors to finish
	stdoutReader.Close()
	stderrReader.Close()

	// Wait for progress processors to finish
	progressWg.Wait()

	// Stop the spinner
	s.Stop()

	if cmdErr != nil {
		e.notifyProgress(fmt.Sprintf("Command failed: %v", cmdErr))
		if !redirectOutput {
			// Include both stdout and stderr in the error message
			return fmt.Errorf("%s\n%s: %w", stdout.String(), stderr.String(), cmdErr)
		}
		return cmdErr
	}

	e.notifyProgress("Command completed successfully")
	return nil
}

// processOutputForProgress monitors the command output for progress indicators
func (e *CommandExecutor) processOutputForProgress(reader io.Reader, source string) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// Look for specific Terraform progress indicators
		if strings.Contains(line, "Plan:") ||
			strings.Contains(line, "Apply complete!") ||
			strings.Contains(line, "Terraform will perform the following actions") ||
			strings.Contains(line, "Preparing the remote plan") ||
			strings.Contains(line, "Executing plan:") ||
			strings.Contains(line, "Still creating...") ||
			strings.Contains(line, "Still destroying...") ||
			strings.Contains(line, "Still modifying...") {
			e.notifyProgress(line)
		}
	}
}

// Ensure CommandExecutor implements the models.Executor interface
var _ models.Executor = (*CommandExecutor)(nil)
