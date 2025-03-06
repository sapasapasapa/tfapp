package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	apperrors "tfapp/internal/errors"
	"tfapp/internal/models"
	"tfapp/internal/terraform"
	"tfapp/internal/ui"
	"tfapp/internal/ui/checkbox"
	"tfapp/internal/ui/menu"
	"tfapp/internal/utils"
)

// App represents the tfapp application.
type App struct {
	tfExecutor models.Executor
	tfPlan     models.PlanService
	tfApply    models.ApplyService
}

// NewApp creates a new instance of the application.
func NewApp() *App {
	executor := terraform.NewCommandExecutor()
	return &App{
		tfExecutor: executor,
		tfPlan:     terraform.NewPlanManager(executor),
		tfApply:    terraform.NewApplyManager(executor),
	}
}

// Run executes the main application logic.
func (a *App) Run(ctx context.Context, flags *Flags) error {
	// Create a temporary file for the plan
	tmpPlanFile, err := createTempPlanFile()
	if err != nil {
		return fmt.Errorf("Failed to create temporary plan file: %w", err)
	}
	defer os.Remove(tmpPlanFile) // Clean up the temporary file when done

	// Handle initialization if requested
	if flags.Init || flags.InitUpgrade {
		if err := a.handleInit(ctx, flags.Init, flags.InitUpgrade); err != nil {
			return fmt.Errorf("Initialization failed: %w", err)
		}
	}

	// Generate the plan
	resources, err := a.tfPlan.CreatePlan(ctx, tmpPlanFile, flags.AdditionalFlags, false)
	if err != nil {
		return fmt.Errorf("Planning failed: %w", err)
	}

	// Show the menu for the user to choose an action
	return a.handleMenuSelection(ctx, tmpPlanFile, resources, flags)
}

// handleInit processes the initialization flags.
func (a *App) handleInit(ctx context.Context, performInit, performUpgrade bool) error {
	if !performInit && !performUpgrade {
		return nil
	}

	if performInit && performUpgrade {
		return apperrors.NewValidationError(
			"init-flags",
			"-init and -init-upgrade cannot be used together",
			apperrors.ErrInvalidInput,
		)
	}

	return a.tfApply.Init(ctx, performUpgrade)
}

// handleMenuSelection displays the menu and processes the user's selection.
func (a *App) handleMenuSelection(ctx context.Context, planFile string, resources []models.Resource, flags *Flags) error {
	selection, err := menu.Show()
	if err != nil {
		return apperrors.NewUserInteractionError("menu selection", "Failed to show menu", err)
	}

	switch selection {
	case "Apply Plan":
		menu.ClearMenuOutput()
		return a.tfApply.Apply(ctx, planFile)
	case "Show Full Plan":
		utils.ClearTerminal()
		err := a.tfPlan.ShowPlan(ctx, planFile)
		if err != nil {
			return err
		}
		printSummary(ctx, planFile)
		return a.handleMenuSelection(ctx, planFile, resources, flags)
	case "Do a target apply":
		menu.ClearMenuOutput()
		return a.handleTargetApply(ctx, resources, flags)
	case "Exit":
		menu.ClearMenuOutput()
		fmt.Println("Exiting without applying changes.")
		return nil
	default:
		return apperrors.NewUserInteractionError(
			"menu selection",
			fmt.Sprintf("Unknown selection: %s", selection),
			nil,
		)
	}
}

// handleTargetApply processes targeted resource application.
func (a *App) handleTargetApply(ctx context.Context, resources []models.Resource, flags *Flags) error {
	// Convert resources to checkbox options
	checkboxOptions := make([]checkbox.Option, 0, len(resources))
	for _, resource := range resources {
		checkboxOptions = append(checkboxOptions, checkbox.Option{
			Name:        resource.Name,
			Description: resource.Action,
			Checked:     false,
		})
	}

	// Show checkbox menu
	selectedOptions, err := checkbox.Show(checkboxOptions)
	if err != nil {
		return apperrors.NewUserInteractionError("resource selection", "Failed to show resource selection menu", err)
	}

	if selectedOptions == nil || len(selectedOptions) == 0 {
		utils.ClearTerminal()
		fmt.Printf("%sNo resources selected for targeted apply.%s\n", ui.ColorInfo, ui.ColorReset)
		return nil
	}

	utils.ClearTerminal()
	for _, opt := range selectedOptions {
		flags.AdditionalFlags = append(flags.AdditionalFlags, "-target="+opt.Name)
	}

	tmpPlanFile, err := createTempPlanFile()
	if err != nil {
		return fmt.Errorf("Failed to create temporary plan file: %w", err)
	}
	defer os.Remove(tmpPlanFile) // Clean up the temporary file when done

	// Generate the plan
	new_resources, err := a.tfPlan.CreatePlan(ctx, tmpPlanFile, flags.AdditionalFlags, true)
	if err != nil {
		return fmt.Errorf("Planning failed: %w", err)
	}

	// Show the menu for the user to choose an action
	return a.handleMenuSelection(ctx, tmpPlanFile, new_resources, flags)
}

// createTempPlanFile creates a temporary file for the Terraform plan.
func createTempPlanFile() (string, error) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "tfapp")
	if err != nil {
		return "", apperrors.NewConfigurationError(
			"temp-file",
			"Failed to create temporary directory",
			err,
		)
	}

	// Create a temporary file path
	return filepath.Join(tempDir, "terraform.tfplan"), nil
}

func printSummary(ctx context.Context, planFilePath string) ([]models.Resource, error) {
	ctxTyped, ok := ctx.(context.Context)
	if !ok {
		return nil, fmt.Errorf("context type assertion failed")
	}

	tfshow := exec.CommandContext(ctxTyped, "terraform", "show", "-no-color", planFilePath)
	tfshow.Stderr = os.Stderr
	output, err := tfshow.Output()
	if err != nil {
		return nil, fmt.Errorf("error showing plan: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var resources []models.Resource

	fmt.Println("Summary of proposed changes:")

	for _, line := range lines {
		if strings.Contains(line, "# module.") {
			action := getResourceAction(line)
			// Clean up the name by removing leading # and whitespace
			name := strings.TrimPrefix(strings.TrimSpace(strings.Split(strings.Split(line, " will be")[0], " must be")[0]), "#")

			resources = append(resources, models.Resource{
				Name:   name,
				Action: action,
				Line:   line,
			})

			colorizedLine := ui.Colorize(line)
			fmt.Println(colorizedLine)
		} else if strings.Contains(line, "Plan:") {
			fmt.Println(ui.Colorize(line))
		}
	}

	return resources, nil
}

// getResourceAction determines the action type from a terraform plan line
func getResourceAction(line string) string {
	if strings.Contains(line, "will be created") {
		return "create"
	} else if strings.Contains(line, "will be destroyed") {
		return "destroy"
	} else if strings.Contains(line, "will be updated in-place") {
		return "update"
	} else if strings.Contains(line, "must be replaced") {
		return "replace"
	}
	return "unknown"
}
