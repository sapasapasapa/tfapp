package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"tfapp/internal/models"
	"tfapp/internal/ui"
	"tfapp/internal/ui/plan"
)

// JSON structs for parsing terraform plan output
type TerraformPlan struct {
	ResourceChanges []ResourceChange `json:"resource_changes"`
	PlannedValues   PlannedValues    `json:"planned_values"`
}

type PlannedValues struct {
	RootModule RootModule `json:"root_module"`
}

type RootModule struct {
	Resources    []Resource    `json:"resources"`
	ChildModules []ChildModule `json:"child_modules"`
}

type ChildModule struct {
	Resources []Resource `json:"resources"`
	Address   string     `json:"address"`
}

type Resource struct {
	Address      string `json:"address"`
	Type         string `json:"type"`
	Name         string `json:"name"`
	ProviderName string `json:"provider_name"`
}

type ResourceChange struct {
	Address       string `json:"address"`
	ModuleAddress string `json:"module_address,omitempty"`
	Mode          string `json:"mode"`
	Type          string `json:"type"`
	Name          string `json:"name"`
	Change        Change `json:"change"`
}

type Change struct {
	Actions []string    `json:"actions"`
	Before  interface{} `json:"before"`
	After   interface{} `json:"after"`
}

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

	// Get plan details in JSON format
	tfshow := exec.CommandContext(ctxTyped, "terraform", "show", "-json", planFilePath)
	tfshow.Stderr = os.Stderr
	output, err := tfshow.Output()
	if err != nil {
		return nil, fmt.Errorf("error showing plan in JSON format: %w", err)
	}

	// Parse the JSON output
	var plan TerraformPlan
	if err := json.Unmarshal(output, &plan); err != nil {
		return nil, fmt.Errorf("error parsing plan JSON: %w", err)
	}

	// Check if there are no changes
	if len(plan.ResourceChanges) == 0 {
		fmt.Printf("%s%sNo changes detected in plan. Your infrastructure is up-to-date.%s\n",
			ui.ColorInfo, ui.TextBold, ui.ColorReset)
		os.Exit(0)
	}

	fmt.Println("\nSummary of proposed changes:")

	// Count actions for summary
	creates := 0
	updates := 0
	destroys := 0
	replaces := 0

	// Process each resource change
	for _, change := range plan.ResourceChanges {
		if len(change.Change.Actions) == 0 {
			continue
		}

		resourceName := change.Address
		action := mapActions(change.Change.Actions)

		// Update counts for summary
		for _, a := range change.Change.Actions {
			switch a {
			case "create":
				creates++
			case "update":
				updates++
			case "delete":
				destroys++
			case "replace":
				replaces++
			}
		}

		// Generate a human-friendly line similar to the text output
		line := formatResourceChangeLine(resourceName, action)

		resources = append(resources, models.Resource{
			Name:   resourceName,
			Action: action,
			Line:   line,
		})

		colorizedLine := ui.Colorize(line)
		fmt.Println(colorizedLine)
	}

	// Display plan summary
	summary := fmt.Sprintf("Plan: %d to add, %d to change, %d to destroy.", creates, updates, destroys)
	fmt.Println(ui.Colorize(summary))

	fmt.Println()
	return resources, nil
}

// formatResourceChangeLine generates a human-readable line for a resource change
func formatResourceChangeLine(resourceName, action string) string {
	var line string
	switch action {
	case "create":
		line = fmt.Sprintf("# %s will be created", resourceName)
	case "destroy":
		line = fmt.Sprintf("# %s will be destroyed", resourceName)
	case "update":
		line = fmt.Sprintf("# %s will be updated in-place", resourceName)
	case "replace":
		line = fmt.Sprintf("# %s must be replaced", resourceName)
	default:
		line = fmt.Sprintf("# %s will be %s", resourceName, action)
	}
	return line
}

// mapActions maps the array of actions to a single action string
func mapActions(actions []string) string {
	if contains(actions, "create") && contains(actions, "delete") {
		return "replace"
	} else if contains(actions, "create") {
		return "create"
	} else if contains(actions, "delete") {
		return "destroy"
	} else if contains(actions, "update") {
		return "update"
	}
	return strings.Join(actions, "/")
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
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
