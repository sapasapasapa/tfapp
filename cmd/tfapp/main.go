package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func tfinit(skip bool) {
	if skip {
		fmt.Printf("%s%sSkipping terraform init%s\n", colorYellow, textBold, colorReset)
		return
	}

	tfinit := exec.Command("terraform", "init")

	var stderr bytes.Buffer
	tfinit.Stderr = &stderr
	tfinit.Stdin = os.Stdin

	spinner := NewSpinner("Running terraform init")
	spinner.Start()

	err := tfinit.Run()
	spinner.Stop()

	if err != nil {
		fmt.Printf("Error executing terraform init: %v\n", stderr.String())
		os.Exit(1)
	}

	fmt.Printf("%s%sTerraform has been successfully initialized!%s\n", colorGreen, textBold, colorReset)
}

func tfplan(tmpPlanFile string, args []string) {
	planArgs := []string{"plan", "-out", tmpPlanFile}
	planArgs = append(planArgs, args...)
	tfplan := exec.Command("terraform", planArgs...)

	var stderr bytes.Buffer
	tfplan.Stderr = &stderr
	tfplan.Stdin = os.Stdin

	spinner := NewSpinner("Creating terraform plan")
	spinner.Start()

	err := tfplan.Run()
	spinner.Stop()
	fmt.Printf("%s%sTerraform plan has been successfully created!%s\n", colorGreen, textBold, colorReset)

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
	skipInit := flag.Bool("skip-init", false, "Skip terraform init")
	flag.Parse()
	tfinit(*skipInit)

	args := flag.Args()

	tmpPlanFile := fmt.Sprintf("/tmp/tfplan%d", os.Getpid())
	defer os.Remove(tmpPlanFile)

	tfplan(tmpPlanFile, args)
	tfapply(tmpPlanFile)
}
