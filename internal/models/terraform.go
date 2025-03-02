// Package models contains the domain models for the application.
package models

// Resource represents a Terraform resource from a plan.
type Resource struct {
	Name   string
	Action string
	Line   string
}

// Executor defines the interface for executing Terraform commands.
type Executor interface {
	// RunCommand executes a terraform command with the given arguments.
	// If redirectOutput is true, the command's output will be redirected to stdout/stderr.
	RunCommand(ctx interface{}, args []string, spinnerMsg string, redirectOutput bool) error
}

// PlanService defines operations related to Terraform plans.
type PlanService interface {
	// CreatePlan generates a Terraform plan and returns affected resources.
	CreatePlan(ctx interface{}, planFilePath string, args []string, targeted bool) ([]Resource, error)
	// ShowPlan displays the full details of a saved plan file.
	ShowPlan(ctx interface{}, planFilePath string) error
}

// ApplyService defines operations related to Terraform applies.
type ApplyService interface {
	// Apply executes terraform apply with the given plan file.
	Apply(ctx interface{}, planFilePath string) error
	// ApplyTargets applies the plan only to the selected resources.
	ApplyTargets(ctx interface{}, targets []string) error
	// Init runs the Terraform init command.
	Init(ctx interface{}, upgrade bool) error
}
