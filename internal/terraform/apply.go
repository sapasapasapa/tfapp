package terraform

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"tfapp/internal/models"
	"tfapp/internal/ui"
)

// ApplyManager handles Terraform apply operations.
type ApplyManager struct {
	executor models.Executor
}

// NewApplyManager creates a new Terraform apply manager.
func NewApplyManager(executor models.Executor) *ApplyManager {
	// Register progress callback with the executor if it's a CommandExecutor
	applyManager := &ApplyManager{
		executor: executor,
	}

	// Try to register progress callback if the executor supports it
	if cmdExecutor, ok := executor.(*CommandExecutor); ok {
		cmdExecutor.RegisterProgressCallback(applyManager.displayProgress)
	}

	return applyManager
}

// displayProgress outputs progress updates to the user
func (a *ApplyManager) displayProgress(status string) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("%s[%s] %s%s\n", ui.ColorHighlight, timestamp, status, ui.ColorReset)
}

// Apply executes `terraform apply` with the given plan file.
// It prompts for confirmation before proceeding.
func (a *ApplyManager) Apply(ctx interface{}, planFilePath string) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Proceed with applying this plan? [yes/No]: ")
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "yes" {
		fmt.Printf("%sStarting terraform apply operation...%s\n", ui.ColorInfo, ui.ColorReset)
		fmt.Printf("%sThis may take several minutes. Progress updates will be displayed.%s\n", ui.ColorInfo, ui.ColorReset)

		if err := a.executor.RunCommand(ctx, []string{"apply", planFilePath}, "Applying terraform plan", false); err != nil {
			return fmt.Errorf("error executing terraform apply: %w", err)
		}
		fmt.Printf("%s%sTerraform apply completed successfully!%s\n",
			ui.ColorSuccess, ui.TextBold, ui.ColorReset)
		return nil
	}

	fmt.Printf("%sApply aborted.%s\n", ui.ColorWarning, ui.ColorReset)
	return nil
}

// ApplyTargets applies the plan only to the selected resources.
// It takes a list of resource targets to apply.
func (a *ApplyManager) ApplyTargets(ctx interface{}, targets []string) error {
	if len(targets) == 0 {
		return fmt.Errorf("no targets specified for apply")
	}

	args := []string{"apply"}
	for _, target := range targets {
		args = append(args, "-target="+target)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Apply to %d selected resources? [yes/No]: ", len(targets))
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "yes" {
		fmt.Printf("%sStarting targeted terraform apply operation...%s\n", ui.ColorInfo, ui.ColorReset)
		fmt.Printf("%sThis may take several minutes. Progress updates will be displayed.%s\n", ui.ColorInfo, ui.ColorReset)

		if err := a.executor.RunCommand(ctx, args, "Applying terraform to selected resources", false); err != nil {
			return fmt.Errorf("error executing targeted terraform apply: %w", err)
		}
		fmt.Printf("%s%sTargeted terraform apply completed successfully!%s\n",
			ui.ColorSuccess, ui.TextBold, ui.ColorReset)
		return nil
	}

	fmt.Printf("%sTargeted apply aborted.%s\n", ui.ColorWarning, ui.ColorReset)
	return nil
}

// Init runs the Terraform init command.
// If upgrade is true, it runs with the -upgrade flag.
func (a *ApplyManager) Init(ctx interface{}, upgrade bool) error {
	if upgrade {
		return a.initUpgrade(ctx)
	}
	return a.initOnly(ctx)
}

// initOnly runs a basic terraform init.
func (a *ApplyManager) initOnly(ctx interface{}) error {
	fmt.Printf("%sStarting terraform init...%s\n", ui.ColorInfo, ui.ColorReset)

	if err := a.executor.RunCommand(ctx, []string{"init"}, "Running terraform init...", false); err != nil {
		return fmt.Errorf("error executing terraform init: %w", err)
	}
	fmt.Printf("%s%sTerraform has been successfully initialized!%s\n",
		ui.ColorSuccess, ui.TextBold, ui.ColorReset)
	return nil
}

// initUpgrade runs terraform init with the -upgrade flag.
// It prompts for confirmation before proceeding.
func (a *ApplyManager) initUpgrade(ctx interface{}) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Using `%s-init-upgrade%s` will run `%sterraform init -upgrade%s`.\n",
		ui.ColorWarning, ui.ColorReset, ui.ColorWarning, ui.ColorReset)
	fmt.Println("This will update providers to the latest version, within the specified version constraints, and could potentially cause breaking changes.")
	fmt.Print("Do you wish to proceed? [yes/No]: ")
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "yes" {
		fmt.Printf("%sStarting terraform init with upgrade...%s\n", ui.ColorInfo, ui.ColorReset)

		if err := a.executor.RunCommand(ctx, []string{"init", "-upgrade"}, "Running terraform init -upgrade...", false); err != nil {
			return fmt.Errorf("error executing terraform init -upgrade: %w", err)
		}
		fmt.Printf("%s%sTerraform has been successfully initialized and upgraded!%s\n",
			ui.ColorSuccess, ui.TextBold, ui.ColorReset)
		return nil
	}

	fmt.Printf("%sCommand aborted.%s\n", ui.ColorWarning, ui.ColorReset)
	return nil
}

// Ensure ApplyManager implements the models.ApplyService interface
var _ models.ApplyService = (*ApplyManager)(nil)
