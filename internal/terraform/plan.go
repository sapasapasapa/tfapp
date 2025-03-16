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
	ResourceDrift   []ResourceChange `json:"resource_drift"`
	FormatVersion   string           `json:"format_version"`
	Applyable       bool             `json:"applyable"`
	Complete        bool             `json:"complete"`
	Errored         bool             `json:"errored"`
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
	Address         string      `json:"address"`
	PreviousAddress string      `json:"previous_address,omitempty"`
	ModuleAddress   string      `json:"module_address,omitempty"`
	Mode            string      `json:"mode"`
	Type            string      `json:"type"`
	Name            string      `json:"name"`
	Index           interface{} `json:"index,omitempty"` // Can be int or string
	Deposed         string      `json:"deposed,omitempty"`
	Change          Change      `json:"change"`
	ActionReason    string      `json:"action_reason,omitempty"`
}

type Change struct {
	Actions         []string    `json:"actions"`
	Before          interface{} `json:"before"`
	After           interface{} `json:"after"`
	AfterUnknown    interface{} `json:"after_unknown,omitempty"`
	BeforeSensitive interface{} `json:"before_sensitive,omitempty"`
	AfterSensitive  interface{} `json:"after_sensitive,omitempty"`
	ReplacePaths    [][]string  `json:"replace_paths,omitempty"`
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

	// Check plan metadata
	if plan.Errored {
		fmt.Printf("%s%sWarning: The plan has errors and may be incomplete.%s\n",
			ui.ColorWarning, ui.TextBold, ui.ColorReset)
	}

	if !plan.Applyable {
		fmt.Printf("%s%sWarning: This plan is not applyable according to Terraform.%s\n",
			ui.ColorWarning, ui.TextBold, ui.ColorReset)
	}

	if !plan.Complete {
		fmt.Printf("%s%sNote: This plan is incomplete. After applying, you will need to run plan again.%s\n",
			ui.ColorInfo, ui.TextBold, ui.ColorReset)
	}

	// Check if there are no changes
	if len(plan.ResourceChanges) == 0 {
		fmt.Printf("%s%sNo changes detected in plan. Your infrastructure is up-to-date.%s\n",
			ui.ColorInfo, ui.TextBold, ui.ColorReset)
		os.Exit(0)
	}

	changing := false
	for _, resource := range plan.ResourceChanges {
		if len(resource.Change.Actions) > 0 && resource.Change.Actions[0] != "no-op" {
			changing = true
			break
		}
	}

	if !changing {
		fmt.Printf("%s%sNo changes detected in plan. Your infrastructure is up-to-date.%s\n",
			ui.ColorInfo, ui.TextBold, ui.ColorReset)
		os.Exit(0)
	}

	// Use the unified DisplayPlanSummary function to show and return resources
	return DisplayPlanSummary(ctxTyped, planFilePath)
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

// getActionReasonText returns a human-readable description of the action reason
func getActionReasonText(reason string) string {
	switch reason {
	case "replace_because_tainted":
		return "tainted, so must be replaced"
	case "replace_because_cannot_update":
		return "cannot be updated in-place"
	case "replace_by_request":
		return "replacement requested"
	case "delete_because_no_resource_config":
		return "no resource configuration found"
	case "delete_because_no_module":
		return "containing module is gone"
	case "delete_because_wrong_repetition":
		return "wrong repetition mode"
	case "delete_because_count_index":
		return "count index out of range"
	case "delete_because_each_key":
		return "for_each key not found"
	case "read_because_config_unknown":
		return "configuration contains unknown values"
	case "read_because_dependency_pending":
		return "has pending dependent resources"
	default:
		return reason
	}
}

// ShowPlan displays the full details of a saved plan file.
func (p *PlanManager) ShowPlan(ctx interface{}, planFilePath string) error {
	ctxTyped, ok := ctx.(context.Context)
	if !ok {
		return fmt.Errorf("context type assertion failed")
	}

	tfshow := exec.CommandContext(ctxTyped, "terraform", "show", "-json", planFilePath)
	tfshow.Stderr = os.Stderr
	output, err := tfshow.Output()
	if err != nil {
		return fmt.Errorf("error showing plan: %w", err)
	}

	// Use the interactive plan viewer
	return plan.Show(string(output))
}

var _ models.PlanService = (*PlanManager)(nil)
