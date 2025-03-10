package terraform

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"tfapp/internal/models"
	"tfapp/internal/ui"
	"tfapp/internal/ui/plan"
)

// PlanManager handles Terraform plan operations.
type PlanManager struct {
	executor models.Executor
}

// NewPlanManager creates a new Terraform plan manager.
func NewPlanManager(executor models.Executor) *PlanManager {
	return &PlanManager{
		executor: executor,
	}
}

// CreatePlan generates a Terraform plan and returns a list of affected resources.
// It saves the plan to the specified file path and runs `terraform plan`.
func (p *PlanManager) CreatePlan(ctx interface{}, planFilePath string, args []string, targeted bool) ([]models.Resource, error) {
	ctxTyped, ok := ctx.(context.Context)
	if !ok {
		return nil, fmt.Errorf("context type assertion failed")
	}

	planArgs := []string{"plan", "-out", planFilePath}
	planArgs = append(planArgs, args...)
	resources := []models.Resource{}

	var printed_line string
	if !targeted {
		printed_line = "Creating terraform plan"
	} else {
		printed_line = "Creating terraform plan with targeted resources"
	}
	err := p.executor.RunCommand(ctx, planArgs, printed_line, false)
	if err != nil {
		return nil, fmt.Errorf("error executing terraform plan: %w", err)
	}

	fmt.Printf("%s%sTerraform plan has been successfully created!%s\n",
		ui.ColorSuccess, ui.TextBold, ui.ColorReset)

	// Get plan details
	tfshow := exec.CommandContext(ctxTyped, "terraform", "show", "-no-color", planFilePath)
	tfshow.Stderr = os.Stderr
	output, err := tfshow.Output()
	if err != nil {
		return nil, fmt.Errorf("error showing plan: %w", err)
	}

	if strings.Contains(string(output), "No changes.") {
		fmt.Printf("%s%sNo changes detected in plan. Your infrastructure matches the configuration.%s\n",
			ui.ColorInfo, ui.TextBold, ui.ColorReset)
		os.Exit(0)
	}

	fmt.Println("\nSummary of proposed changes:")

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "#") && (strings.Contains(line, "will be") || strings.Contains(line, "must be")) {
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

	fmt.Println()
	return resources, nil
}

// ShowPlan displays the full details of a saved plan file.
func (p *PlanManager) ShowPlan(ctx interface{}, planFilePath string) error {
	ctxTyped, ok := ctx.(context.Context)
	if !ok {
		return fmt.Errorf("context type assertion failed")
	}

	tfshow := exec.CommandContext(ctxTyped, "terraform", "show", planFilePath)
	tfshow.Stderr = os.Stderr
	output, err := tfshow.Output()
	if err != nil {
		return fmt.Errorf("error showing plan: %w", err)
	}

	// Use the interactive plan viewer
	return plan.Show(string(output))
}

// getResourceAction determines the action being performed on a resource from a plan line.
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
	return ""
}

// Ensure PlanManager implements the models.PlanService interface
var _ models.PlanService = (*PlanManager)(nil)
