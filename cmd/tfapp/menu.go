package main

import (
	"fmt"

	"github.com/manifoldco/promptui"
)

// MenuOption represents a single menu option
type MenuOption struct {
	Name        string
	Description string
	Selected    string
}

// String implements the fmt.Stringer interface
func (o MenuOption) String() string {
	return fmt.Sprintf("%s - %s", o.Name, o.Description)
}

// ShowMenu displays an interactive menu and returns the selected option
func ShowMenu() (string, error) {
	options := []MenuOption{
		{
			Name:        "Apply Plan",
			Description: "Execute the current plan",
			Selected:    "Applying plan...",
		},
		{
			Name:        "Show Full Plan",
			Description: "Display the complete plan",
			Selected:    "Retrieving full plan...",
		},
		{
			Name:        "Exit",
			Description: "Discard the plan and exit",
			Selected:    "The plan has been discarded.",
		},
	}

	prompt := promptui.Select{
		Label:    "Select Action",
		HideHelp: true,
		Items:    options,
		Size:     3,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "> {{ .Name | cyan }} - {{ .Description }}",
			Inactive: "  {{ .Name | faint }}",
			Selected: "{{ .Selected | green }}",
		},
	}

	index, _, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("prompt failed: %v", err)
	}

	return options[index].Name, nil
}
