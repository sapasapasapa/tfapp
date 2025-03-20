package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"tfapp/internal/models"
	"tfapp/internal/ui"

	"github.com/charmbracelet/lipgloss"
)

// DisplayPlanSummary displays a summary of a Terraform plan and returns the identified resources.
// It supports both regular and drifted resources with consistent styling.
func DisplayPlanSummary(ctx context.Context, planFilePath string) ([]models.Resource, error) {
	// Get plan details in JSON format
	tfshow := exec.CommandContext(ctx, "terraform", "show", "-json", planFilePath)
	output, err := tfshow.Output()
	if err != nil {
		return nil, fmt.Errorf("error showing plan in JSON format: %w", err)
	}

	// Parse the JSON output
	var plan TerraformPlan
	if err := json.Unmarshal(output, &plan); err != nil {
		return nil, fmt.Errorf("error parsing plan JSON: %w", err)
	}

	var resources []models.Resource

	// Process resource drift if present
	if len(plan.ResourceDrift) > 0 {
		fmt.Printf("\n%s%sResources that have changed outside of Terraform:%s\n",
			ui.TextBold,
			ui.ColorCyan,
			ui.ColorReset)

		for _, drift := range plan.ResourceDrift {
			if len(drift.Change.Actions) == 0 {
				continue
			}

			resourceName := drift.Address
			action := "drift:" + mapActions(drift.Change.Actions)

			// Generate a human-friendly line for drift
			line := fmt.Sprintf("# %s has drifted", resourceName)

			resources = append(resources, models.Resource{
				Name:   resourceName,
				Action: action,
				Line:   line,
			})

			// Apply the special drift styling only to the "has drifted" part
			resourcePrefix := fmt.Sprintf("# %s ", resourceName)
			driftText := "has drifted"
			colorizedDriftText := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF9900")). // Orange color for drift phrase only
				Render(driftText)
			colorizedLine := resourcePrefix + colorizedDriftText
			fmt.Println(colorizedLine)
		}
		fmt.Println()
	}

	fmt.Printf("\n%s%sSummary of proposed changes:%s\n",
		ui.TextBold,
		ui.ColorCyan,
		ui.ColorReset)

	// Count actions for summary
	creates := 0
	updates := 0
	destroys := 0
	replaces := 0
	moves := 0

	// Process each resource change
	for _, change := range plan.ResourceChanges {
		if len(change.Change.Actions) == 0 || change.Change.Actions[0] == "no-op" {
			continue
		}

		resourceName := change.Address
		action := mapActions(change.Change.Actions)

		// Check if this is a moved resource
		wasMoved := false
		if change.PreviousAddress != "" && change.PreviousAddress != change.Address {
			wasMoved = true
			moves++
		}

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
		var line string
		if wasMoved {
			// Add information about the move
			line = fmt.Sprintf("# %s will be %s (moved from %s)",
				resourceName, getGrammaticalAction(action), change.PreviousAddress)
		} else if change.ActionReason != "" {
			// Include action reason if available
			reasonText := getActionReasonText(change.ActionReason)
			line = fmt.Sprintf("# %s will be %s (%s)",
				resourceName, getGrammaticalAction(action), reasonText)
		} else {
			line = formatResourceChangeLine(resourceName, action)
		}

		resources = append(resources, models.Resource{
			Name:   resourceName,
			Action: action,
			Line:   line,
		})

		// Display the line with appropriate color
		fmt.Println(ui.Colorize(line))
	}

	// Display plan summary
	summary := fmt.Sprintf("Plan: %d to add, %d to change, %d to destroy.", creates, updates, destroys)
	if moves > 0 {
		summary += fmt.Sprintf(" (%d resources moved)", moves)
	}
	fmt.Println(ui.Colorize(summary))
	fmt.Println()

	return resources, nil
}

// getGrammaticalAction returns the grammatically correct form of an action
func getGrammaticalAction(action string) string {
	switch action {
	case "create":
		return "created"
	case "update":
		return "updated"
	case "replace":
		return "replaced"
	case "destroy":
		return "destroyed"
	case "move":
		return "moved"
	default:
		return action + "d" // Add 'd' as a general case
	}
}
