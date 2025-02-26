package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	apperrors "tfapp/internal/errors"
	"tfapp/internal/models"
	"tfapp/internal/terraform"
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
	resources, err := a.tfPlan.CreatePlan(ctx, tmpPlanFile, flags.AdditionalFlags)
	if err != nil {
		return fmt.Errorf("Planning failed: %w", err)
	}

	// Show the menu for the user to choose an action
	return a.handleMenuSelection(ctx, tmpPlanFile, resources)
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
func (a *App) handleMenuSelection(ctx context.Context, planFile string, resources []models.Resource) error {
	selection, err := menu.Show()
	if err != nil {
		return apperrors.NewUserInteractionError("menu selection", "Failed to show menu", err)
	}

	switch selection {
	case "Apply Plan":
		return a.tfApply.Apply(ctx, planFile)
	case "Show Full Plan":
		return a.tfPlan.ShowPlan(ctx, planFile)
	case "Do a target apply":
		return a.handleTargetApply(ctx, resources)
	case "Exit":
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
func (a *App) handleTargetApply(ctx context.Context, resources []models.Resource) error {
	utils.ClearTerminal()

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
		fmt.Println("No resources selected for targeted apply.")
		return nil
	}

	// Extract the selected resource names
	targets := make([]string, 0, len(selectedOptions))
	for _, opt := range selectedOptions {
		targets = append(targets, opt.Name)
	}

	// Apply to the selected targets
	return a.tfApply.ApplyTargets(ctx, targets)
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
