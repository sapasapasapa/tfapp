// Package checkbox provides a multi-selection checkbox menu for terminal applications.
package checkbox

import (
	"fmt"
	"strings"

	"tfapp/internal/ui"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Option represents a single checkbox menu option.
type Option struct {
	Name        string
	Description string
	Checked     bool
}

// String implements the fmt.Stringer interface.
func (o Option) String() string {
	return fmt.Sprintf("%s - %s", o.Name, o.Description)
}

// model represents the checkbox menu state.
type model struct {
	options  []Option
	cursor   int
	quitting bool
	list     list.Model
}

// Init implements tea.Model.
func (m model) Init() tea.Cmd {
	// Ensure styles are initialized
	m.updateStyles()
	return nil
}

// Update implements tea.Model.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "a":
			// Select all
			for i := range m.options {
				m.options[i].Checked = true
			}
		case "n":
			// Select none
			for i := range m.options {
				m.options[i].Checked = false
			}
		case "enter":
			// Return with current selection
			return m, tea.Quit
		}
	}
	return m, nil
}

var (
	// These will be initialized properly in updateStyles
	activeStyle      = lipgloss.NewStyle()
	faintStyle       = lipgloss.NewStyle()
	cursorStyle      = lipgloss.NewStyle()
	checkedStyle     = lipgloss.NewStyle()
	uncheckedStyle   = lipgloss.NewStyle()
	keyBindingStyle  = lipgloss.NewStyle()
	helpTextStyle    = lipgloss.NewStyle()
	instructionStyle = lipgloss.NewStyle()
	// Action type styles
	createStyle  = lipgloss.NewStyle()
	updateStyle  = lipgloss.NewStyle()
	destroyStyle = lipgloss.NewStyle()
	nameStyle    = lipgloss.NewStyle()
)

// View implements tea.Model.
func (m model) View() string {
	var s strings.Builder

	s.WriteString("Select resources to apply\n\n")

	for i, option := range m.options {
		var cursor string
		optNameStyle := nameStyle
		checkedSymbol := "[ ] "
		if option.Checked {
			checkedSymbol = "[x] "
		}

		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
			optNameStyle = activeStyle
			if option.Checked {
				checkedSymbol = checkedStyle.Render("[x] ")
			} else {
				checkedSymbol = uncheckedStyle.Render("[ ] ")
			}
		} else {
			cursor = "  "
			if option.Checked {
				checkedSymbol = checkedStyle.Render("[x] ")
			} else {
				checkedSymbol = uncheckedStyle.Render("[ ] ")
			}
		}

		// Render name with checkbox
		s.WriteString(fmt.Sprintf("%s%s%s",
			cursor,
			checkedSymbol,
			optNameStyle.Render(option.Name)))

		// Add description with appropriate color based on action type
		if option.Description != "" {
			var descStyle lipgloss.Style

			switch option.Description {
			case "create":
				descStyle = createStyle
			case "update":
				descStyle = updateStyle
			case "destroy":
				descStyle = destroyStyle
			default:
				descStyle = faintStyle
			}

			s.WriteString(fmt.Sprintf(" - %s", descStyle.Render(option.Description)))
		}

		s.WriteString("\n")
	}

	// Add instructions at the bottom
	s.WriteString("\n")
	s.WriteString(instructionStyle.Render("Press space to toggle selection, enter to confirm\n"))
	s.WriteString(instructionStyle.Render("Press "))
	s.WriteString(keyBindingStyle.Render("a"))
	s.WriteString(instructionStyle.Render(" to select all, "))
	s.WriteString(keyBindingStyle.Render("n"))
	s.WriteString(instructionStyle.Render(" to select none\n"))

	return s.String()
}

// Show displays a checkbox menu with the provided options.
func Show(options []Option) ([]Option, error) {
	if len(options) == 0 {
		return nil, nil
	}

	m := model{
		options: options,
	}

	// Initialize styles with latest config
	m.updateStyles()

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	if finalModel.(model).quitting {
		return nil, nil
	}

	var selected []Option
	for _, opt := range finalModel.(model).options {
		if opt.Checked {
			selected = append(selected, opt)
		}
	}

	return selected, nil
}

// updateStyles sets the styles for the checkbox menu based on terminal dimensions.
func (m *model) updateStyles() {
	// Use configured highlight color
	highlightColor := lipgloss.Color(ui.GetHexColorByName("highlight"))
	faintColor := lipgloss.Color(ui.GetHexColorByName("faint"))
	successColor := lipgloss.Color(ui.GetHexColorByName("success"))
	infoColor := lipgloss.Color(ui.GetHexColorByName("info"))
	warningColor := lipgloss.Color(ui.GetHexColorByName("warning"))
	errorColor := lipgloss.Color(ui.GetHexColorByName("error"))

	// Update the styles to use the configured colors
	activeStyle = lipgloss.NewStyle().Foreground(highlightColor).Bold(true)
	faintStyle = lipgloss.NewStyle().Foreground(faintColor)
	cursorStyle = lipgloss.NewStyle().Foreground(highlightColor)
	checkedStyle = lipgloss.NewStyle().Foreground(successColor)
	uncheckedStyle = lipgloss.NewStyle().Foreground(faintColor)
	keyBindingStyle = lipgloss.NewStyle().Foreground(infoColor)
	helpTextStyle = lipgloss.NewStyle().Foreground(faintColor)
	instructionStyle = lipgloss.NewStyle().Foreground(faintColor)

	// Update action styles
	nameStyle = lipgloss.NewStyle().Foreground(faintColor)
	createStyle = lipgloss.NewStyle().Foreground(successColor) // Green
	updateStyle = lipgloss.NewStyle().Foreground(warningColor) // Yellow
	destroyStyle = lipgloss.NewStyle().Foreground(errorColor)  // Red
}
