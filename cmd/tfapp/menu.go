package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// Model represents the menu state
type menuModel struct {
	options  []MenuOption
	cursor   int
	selected *MenuOption
	quitting bool
}

// Init implements tea.Model
func (m menuModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m menuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

var (
	activeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8239F3")).Bold(true) // purple and bold
	faintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))                // faint
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8239F3"))            // purple cursor
)

// View implements tea.Model
func (m menuModel) View() string {
	var s strings.Builder

	s.WriteString("Select Action\n\n")

	for i, option := range m.options {
		var cursor string
		nameStyle := faintStyle
		descStyle := faintStyle
		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
			nameStyle = activeStyle
			descStyle = lipgloss.NewStyle() // default style for active description
		} else {
			cursor = "  "
		}

		// Render name and description separately
		s.WriteString(fmt.Sprintf("%s%s - %s\n",
			cursor,
			nameStyle.Render(option.Name),
			descStyle.Render(option.Description)))
	}

	return s.String()
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
			Name:        "Do a target apply",
			Description: "Execute the plan for some specific resources",
			Selected:    "Listing resources...",
		},
		{
			Name:        "Exit",
			Description: "Discard the plan and exit",
			Selected:    "The plan has been discarded.",
		},
	}

	m := menuModel{
		options: options,
	}

	p := tea.NewProgram(m)
	model, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("error running menu: %v", err)
	}

	finalModel := model.(menuModel)
	if finalModel.quitting || finalModel.selected == nil {
		return "Exit", nil
	}

	return finalModel.selected.Name, nil
}
