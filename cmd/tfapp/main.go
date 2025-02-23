package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	tmpPlanFile := fmt.Sprintf("/tmp/tfplan.%d", os.Getpid())
	// Create the base command with the required arguments
	args := []string{"plan", "-out", tmpPlanFile}
	// Append any additional arguments passed to our program
	args = append(args, os.Args[1:]...)
	tfplan := exec.Command("terraform", args...)

	// Set the command to use the current terminal's stdin, stdout, and stderr
	tfplan.Stdin = os.Stdin
	tfplan.Stdout = os.Stdout
	tfplan.Stderr = os.Stderr

	// Execute the command
	err := tfplan.Run()
	if err != nil {
		fmt.Printf("Error executing terraform apply: %v\n", err)
		os.Exit(1)
	}
}
