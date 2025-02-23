package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func tfplan(tmpPlanFile string) {
	args := []string{"plan", "-out", tmpPlanFile}
	args = append(args, os.Args[1:]...)
	tfplan := exec.Command("terraform", args...)

	var stderr bytes.Buffer
	tfplan.Stderr = &stderr
	tfplan.Stdin = os.Stdin

	spinner := NewSpinner("Creating terraform plan")
	spinner.Start()

	err := tfplan.Run()
	spinner.Stop()

	if err != nil {
		fmt.Printf("Error executing terraform plan: %v\n", stderr.String())
		os.Exit(1)
	}

	fmt.Println("\nSummary of proposed changes:")
	tfshow := exec.Command("terraform", "show", "-no-color", tmpPlanFile)
	tfshow.Stderr = os.Stderr
	output, err := tfshow.Output()
	if err != nil {
		fmt.Printf("Error showing plan: %v\n", err)
		os.Exit(1)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "# module.") || strings.HasPrefix(line, "Plan: ") {
			fmt.Println(colorize(line))
		}
	}
	fmt.Println()
}

func tfapply(tmpPlanFile string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Proceed with applying this plan? [yes/No]: ")
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading input: %v\n", err)
		os.Exit(1)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "yes" {
		// Step 4: Apply the plan
		tfapply := exec.Command("terraform", "apply", tmpPlanFile)
		tfapply.Stdin = os.Stdin
		tfapply.Stdout = os.Stdout
		tfapply.Stderr = os.Stderr

		err = tfapply.Run()
		if err != nil {
			fmt.Printf("Error executing terraform apply: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Apply aborted.")
	}
}

func main() {
	tmpPlanFile := fmt.Sprintf("/tmp/tfplan%d", os.Getpid())
	defer os.Remove(tmpPlanFile)

	tfplan(tmpPlanFile)
	tfapply(tmpPlanFile)
}
