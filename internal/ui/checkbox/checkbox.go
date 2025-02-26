// Package checkbox provides a multi-selection checkbox menu for terminal applications.
package checkbox

import (
	"fmt"
	"strings"

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
}

// Init implements tea.Model.
func (m model) Init() tea.Cmd {
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
	activeStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#8239F3")).Bold(true)
	faintStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("242"))
	cursorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#8239F3"))
	checkedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	uncheckedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	keyBindingStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	helpTextStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	instructionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
)

// View implements tea.Model.
func (m model) View() string {
	var s strings.Builder

	s.WriteString("Select resources to apply\n\n")

	for i, option := range m.options {
		var cursor string
		nameStyle := faintStyle
		checkedSymbol := "[ ] "
		if option.Checked {
			checkedSymbol = "[x] "
		}

		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
			nameStyle = activeStyle
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

		// Render name, checkbox and description
		s.WriteString(fmt.Sprintf("%s%s%s\n",
			cursor,
			checkedSymbol,
			nameStyle.Render(option.Name)))
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

// Show displays an interactive checkbox menu and returns the selected options.
func Show(options []Option) ([]Option, error) {
	m := model{
		options: options,
	}

	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("error running checkbox menu: %w", err)
	}

	checkboxModel := result.(model)
	if checkboxModel.quitting {
		return nil, nil
	}

	// Filter out only the checked options
	var selected []Option
	for _, option := range checkboxModel.options {
		if option.Checked {
			selected = append(selected, option)
		}
	}

	return selected, nil
}
