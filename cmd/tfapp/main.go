package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func tfonlyinit() {
	if err := runTerraformCommand([]string{"init"}, "Running terraform init...", false); err != nil {
		fmt.Printf("%sError executing terraform init: \n%v\n", colorRed, err)
		os.Exit(1)
	}
	fmt.Printf("%s%sTerraform has been successfully initialized!%s\n", colorGreen, textBold, colorReset)
}

func tfinitupgrade() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Using `%s-init-upgrade%s` will run `%sterraform init -upgrade%s`.\n", colorYellow, colorReset, colorYellow, colorReset)
	fmt.Println("This will update providers to the latest version, within the specified version constraints, and could potentially cause breaking changes.", colorYellow, colorReset)
	fmt.Print("Do you wish to proceed? [yes/No]: ")
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("%sError reading input: \n%v\n", colorRed, err)
		os.Exit(1)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "yes" {
		if err := runTerraformCommand([]string{"init", "-upgrade"}, "Running terraform init -upgrade...", false); err != nil {
			fmt.Printf("%sError executing terraform init -upgrade: \n%v\n", colorRed, err)
			os.Exit(1)
		}
		fmt.Printf("%s%sTerraform has been successfully initialized and upgraded!%s\n", colorGreen, textBold, colorReset)
	} else {
		fmt.Printf("%sCommand aborted.%s\n", colorYellow, colorReset)
		os.Exit(0)
	}
}

func tfinit(performInit bool, performUpgrade bool) {
	if !performInit && !performUpgrade {
		fmt.Printf("%sTerraform init has been skipped.%s\n", colorYellow, colorReset)
		return
	}

	if performInit && performUpgrade {
		fmt.Printf("%sError: -init and -init-upgrade cannot be used together.%s\n", colorRed, colorReset)
		os.Exit(1)
	}

	if performInit {
		tfonlyinit()
	} else if performUpgrade {
		tfinitupgrade()
	}
}

func tfplan(tmpPlanFile string, args []string) []string {
	planArgs := []string{"plan", "-out", tmpPlanFile}
	planArgs = append(planArgs, args...)
	resources := []string{}

	if err := runTerraformCommand(planArgs, "Creating terraform plan", false); err != nil {
		fmt.Printf("%sError executing terraform plan: \n%v\n", colorRed, err)
		os.Exit(1)
	}

	fmt.Printf("%s%sTerraform plan has been successfully created!%s\n", colorGreen, textBold, colorReset)

	fmt.Println("\nSummary of proposed changes:")
	tfshow := exec.Command("terraform", "show", "-no-color", tmpPlanFile)
	tfshow.Stderr = os.Stderr
	output, err := tfshow.Output()
	if err != nil {
		fmt.Printf("%sError showing plan: \n%v\n", colorRed, err)
		os.Exit(1)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "# module.") {
			resources = append(resources, line)
			colorizedLine := colorize(line)
			fmt.Println(colorizedLine)
		} else if strings.Contains(line, "Plan:") {
			fmt.Println(colorize(line))
		}
	}
	fmt.Println()
	return resources
}

func tfapply(tmpPlanFile string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Proceed with applying this plan? [yes/No]: ")
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("%sError reading input: \n%v\n", colorRed, err)
		os.Exit(1)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "yes" {
		if err := runTerraformCommand([]string{"apply", tmpPlanFile}, "Applying terraform plan", true); err != nil {
			fmt.Printf("%sError executing terraform apply: \n%v\n", colorRed, err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("%sApply aborted.%s\n", colorYellow, colorReset)
	}
}

func tfshow(tmpPlanFile string) {
	tfshow := exec.Command("terraform", "show", "-no-color", tmpPlanFile)
	tfshow.Stderr = os.Stderr
	output, err := tfshow.Output()
	if err != nil {
		fmt.Printf("%sError showing plan: \n%v\n", colorRed, err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}

func getResourceAction(line string) string {
	if strings.Contains(line, "will be created") {
		return "create"
	} else if strings.Contains(line, "will be destroyed") {
		return "destroy"
	} else if strings.Contains(line, "will be updated in-place") {
		return "update"
	}
	return ""
}

func tftargetapply(resources []string) []string {
	clearTerminal()

	// Example resources - these should be replaced with actual resources from your Terraform plan
	checkboxOptions := []CheckboxMenuOption{}
	for _, resource := range resources {
		action := getResourceAction(resource)
		// Clean up the name by removing leading # and whitespace
		name := strings.TrimPrefix(strings.TrimSpace(strings.Split(resource, " will be")[0]), "#")
		checkboxOptions = append(checkboxOptions, CheckboxMenuOption{
			Name:        name,
			Description: action,
		})
	}

	selected, err := ShowCheckboxMenu(checkboxOptions)
	if err != nil {
		fmt.Printf("%sError showing checkbox menu: \n%v\n", colorRed, err)
		os.Exit(1)
	}

	if len(selected) == 0 {
		fmt.Printf("%sNo resources selected for targeted apply.%s\n", colorYellow, colorReset)
		os.Exit(0)
	}

	// Create target arguments for selected resources
	var targetArgs []string
	for _, opt := range selected {
		targetArgs = append(targetArgs, "-target="+opt.Name)
	}

	return targetArgs
}

func menu(tmpPlanFile string, resources []string, args []string) {
	action, err := ShowMenu()
	if err != nil {
		fmt.Printf("%sError showing menu: \n%v\n", colorRed, err)
		os.Exit(1)
	}

	switch action {
	case "Apply Plan":
		tfapply(tmpPlanFile)
		return
	case "Show Full Plan":
		tfshow(tmpPlanFile)
		menu(tmpPlanFile, resources, args)
		return
	case "Do a target apply":
		targetArgs := tftargetapply(resources)
		os.Remove(tmpPlanFile)
		newArgs := append(targetArgs, args...)
		mainLoop(newArgs)
		return
	case "Exit":
		clearTerminal()
		fmt.Printf("%sCommand aborted.%s\n", colorYellow, colorReset)
		os.Exit(0)
	}
}

func mainLoop(args []string) {
	tmpPlanFile := fmt.Sprintf("/tmp/tfplan%d", os.Getpid())
	defer os.Remove(tmpPlanFile)

	resources := tfplan(tmpPlanFile, args)
	menu(tmpPlanFile, resources, args)
}

func main() {
	clearTerminal()

	performInit := flag.Bool("init", false, "Run `terraform init`")
	performUpgrade := flag.Bool("init-upgrade", false, "Run `terraform init -upgrade`")
	flag.Parse()
	tfinit(*performInit, *performUpgrade)

	args := flag.Args()
	mainLoop(args)
}
