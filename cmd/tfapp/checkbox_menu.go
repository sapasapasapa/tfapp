package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CheckboxMenuOption represents a single checkbox menu option
type CheckboxMenuOption struct {
	Name        string
	Description string
	Checked     bool
}

// String implements the fmt.Stringer interface
func (o CheckboxMenuOption) String() string {
	return fmt.Sprintf("%s - %s", o.Name, o.Description)
}

// checkboxModel represents the checkbox menu state
type checkboxModel struct {
	options  []CheckboxMenuOption
	cursor   int
	quitting bool
}

// Init implements tea.Model
func (m checkboxModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m checkboxModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.options) - 1
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
		case " ":
			// Toggle the selected item
			m.options[m.cursor].Checked = !m.options[m.cursor].Checked
		case "enter":
			return m, tea.Quit
		}
	}
	return m, nil
}

var (
	checkboxActiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8239F3")).Bold(true)
	checkboxFaintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	checkboxCursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8239F3"))
	checkboxXStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#8239F3"))
)

// View implements tea.Model
func (m checkboxModel) View() string {
	var s strings.Builder

	s.WriteString("Select Resources to Apply\n\n")

	for i, option := range m.options {
		var cursor string
		var checkbox string
		nameStyle := checkboxFaintStyle
		descStyle := checkboxFaintStyle

		if m.cursor == i {
			cursor = checkboxCursorStyle.Render("> ")
			nameStyle = checkboxActiveStyle
			// Color the description based on the action type when active
			switch option.Description {
			case "create":
				descStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#32CD32")) // Bright green
			case "destroy":
				descStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4040")) // Bright red
			case "update":
				descStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")) // Bright yellow
			default:
				descStyle = lipgloss.NewStyle() // Default style for unknown actions
			}
		} else {
			cursor = "  "
		}

		if option.Checked {
			checkbox = "[" + checkboxXStyle.Render("x") + "] "
			if m.cursor != i {
				nameStyle = checkboxActiveStyle
			}
			// Color the description based on the action type when checked
			switch option.Description {
			case "create":
				descStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#32CD32")) // Bright green
			case "destroy":
				descStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4040")) // Bright red
			case "update":
				descStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700")) // Bright yellow
			default:
				descStyle = lipgloss.NewStyle() // Default style for unknown actions
			}
		} else {
			checkbox = "[ ] "
			if m.cursor != i {
				descStyle = checkboxFaintStyle // Only use faint style if not active and not checked
			}
		}

		// Render name and description separately
		s.WriteString(fmt.Sprintf("%s%s%s - %s\n",
			cursor,
			checkbox,
			nameStyle.Render(option.Name),
			descStyle.Render(option.Description)))
	}

	s.WriteString("\n(space to toggle, enter to confirm)")

	return s.String()
}

// ShowCheckboxMenu displays an interactive checkbox menu and returns the selected options
func ShowCheckboxMenu(options []CheckboxMenuOption) ([]CheckboxMenuOption, error) {
	m := checkboxModel{
		options: options,
	}

	p := tea.NewProgram(m)
	model, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("error running checkbox menu: %v", err)
	}

	finalModel := model.(checkboxModel)
	if finalModel.quitting {
		return nil, nil
	}

	// Filter and return only the selected options
	var selected []CheckboxMenuOption
	for _, opt := range finalModel.options {
		if opt.Checked {
			selected = append(selected, opt)
		}
	}

	return selected, nil
}
