package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	// Create the terraform apply command
	cmd := exec.Command("terraform", "apply")
	
	// Set the command to use the current terminal's stdin, stdout, and stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	// Execute the command
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error executing terraform apply: %v\n", err)
		os.Exit(1)
	}
} 