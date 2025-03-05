// Package menu provides interactive terminal menu components.
package menu

import (
	"fmt"
	"strings"

	"tfapp/internal/ui"

	"errors"

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
	choice   string
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
			m.choice = m.selected.Name
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

// Show displays the menu and returns the selected option.
func Show() (string, error) {
	p := tea.NewProgram(initialModel())
	m, err := p.Run()
	if err != nil {
		return "", err
	}

	if m, ok := m.(model); ok {
		return m.choice, nil
	}

	return "", errors.New("could not get selected choice")
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

// initialModel creates a new model for the menu.
func initialModel() model {
	choices := []string{
		"Apply Plan",
		"Show Full Plan",
		"Do a target apply",
		"Exit",
	}

	descriptions := []string{
		"Apply the plan to your infrastructure",
		"View the plan with collapsible resources",
		"Apply specific resources from the plan",
		"Exit without applying changes",
	}

	options := make([]Option, len(choices))
	for i, choice := range choices {
		options[i] = Option{
			Name:        choice,
			Description: descriptions[i],
			Selected:    "",
		}
	}

	mod := model{
		options: options,
		cursor:  0,
	}

	// Initialize styles with latest config
	mod.updateStyles()

	return mod
}
