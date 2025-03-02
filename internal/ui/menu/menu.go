// Package menu provides interactive terminal menu components.
package menu

import (
	"fmt"
	"strings"

	"tfapp/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Option represents a single menu option.
type Option struct {
	Name        string
	Description string
	Selected    string
}

// String implements the fmt.Stringer interface.
func (o Option) String() string {
	if o.Description != "" {
		return fmt.Sprintf("%s - %s", o.Name, o.Description)
	}
	return o.Name
}

// model represents the menu state.
type model struct {
	options  []Option
	cursor   int
	selected *Option
	quitting bool
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
		case "enter", " ":
			m.selected = &m.options[m.cursor]
			return m, tea.Quit
		}
	}
	return m, nil
}

// updateStyles sets the styles for the menu based on terminal dimensions.
func (m *model) updateStyles() {
	// Use configured highlight color - use the hex color format for lipgloss
	highlightColor := lipgloss.Color(ui.GetHexColorByName("highlight"))
	faintColor := lipgloss.Color(ui.GetHexColorByName("faint"))

	// Update the styles directly
	activeStyle = lipgloss.NewStyle().Foreground(highlightColor).Bold(true)
	faintStyle = lipgloss.NewStyle().Foreground(faintColor)
	cursorStyle = lipgloss.NewStyle().Foreground(highlightColor)
	nameStyle = lipgloss.NewStyle().Foreground(faintColor)
	descriptionStyle = lipgloss.NewStyle().Foreground(faintColor)
}

var (
	// These will be initialized properly in updateStyles
	activeStyle            = lipgloss.NewStyle()
	faintStyle             = lipgloss.NewStyle()
	cursorStyle            = lipgloss.NewStyle()
	nameStyle              = lipgloss.NewStyle()
	descriptionStyle       = lipgloss.NewStyle()
	activeDescriptionStyle = lipgloss.NewStyle()
)

// View implements tea.Model.
func (m model) View() string {
	var s strings.Builder

	s.WriteString("Select Action\n\n")

	for i, option := range m.options {
		var cursor string
		optNameStyle := nameStyle
		optDescStyle := descriptionStyle

		if m.cursor == i {
			cursor = cursorStyle.Render(ui.GetCursorChar())
			optNameStyle = activeStyle
			optDescStyle = activeDescriptionStyle
		} else {
			cursor = " "
		}

		// Display option name with its description
		s.WriteString(fmt.Sprintf("%s %s",
			cursor,
			optNameStyle.Render(option.Name)))

		// Add description if available
		if option.Description != "" {
			s.WriteString(fmt.Sprintf(" - %s", optDescStyle.Render(option.Description)))
		}

		s.WriteString("\n")
	}

	return s.String()
}

// Show displays a menu with the default options and returns the selected option.
func Show() (string, error) {
	options := []Option{
		{
			Name:        "Apply Plan",
			Description: "Apply the plan to the selected targets",
		},
		{
			Name:        "Show Full Plan",
			Description: "Show the full plan",
		},
		{
			Name:        "Do a target apply",
			Description: "Apply the plan to the selected targets",
		},
		{
			Name:        "Exit",
			Description: "Exit the application",
		},
	}

	m := model{
		options: options,
		cursor:  0,
	}

	// Initialize styles with latest config
	m.updateStyles()

	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	if finalModel.(model).quitting && finalModel.(model).selected == nil {
		return "Exit", nil
	}

	if finalModel.(model).selected == nil {
		return "", fmt.Errorf("no option selected")
	}

	return finalModel.(model).selected.Name, nil
}

// ClearMenuOutput clears the menu output area from the terminal
// without clearing other content.
func ClearMenuOutput() {
	// Calculate number of lines in menu (header + blank line + 4 options + blank line)
	menuHeight := 7

	// ANSI escape sequence to:
	// 1. Move cursor up menuHeight lines
	// 2. Clear from cursor to end of screen
	fmt.Printf("\033[%dA\033[J", menuHeight)
}

// initialModel creates the initial model for the menu.
func initialModel(options []Option) model {
	mod := model{
		options: options,
		cursor:  0,
	}

	// Initialize styles
	mod.updateStyles()

	return mod
}
