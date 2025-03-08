// Package checkbox provides a multi-selection checkbox menu for terminal applications.
package checkbox

import (
	"fmt"
	"strings"

	"tfapp/internal/ui"

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
	options      []Option
	cursor       int
	quitting     bool
	windowTop    int  // The top line of the window being displayed
	windowHeight int  // Height of visible window
	ready        bool // Whether we've received the window size yet
	showHelp     bool // Whether to show the help tooltip
}

// Init implements tea.Model.
func (m model) Init() tea.Cmd {
	// Initialize styles
	m.updateStyles()
	return nil
}

// Update implements tea.Model.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Store the old window height to check if it changed significantly
		oldHeight := m.windowHeight

		// Update window height based on terminal size
		// Reserve some space for headers and footers (approximately 3 lines)
		m.windowHeight = msg.Height - 3
		if m.windowHeight < 5 {
			m.windowHeight = 5 // Minimum reasonable height
		}

		// Mark as ready now that we've received window dimensions
		m.ready = true

		// If the window height changed significantly, ensure the cursor remains visible
		if oldHeight != m.windowHeight {
			// Make sure we don't exceed the maximum possible windowTop
			maxTop := len(m.options) - m.windowHeight
			if maxTop < 0 {
				maxTop = 0
			}
			if m.windowTop > maxTop {
				m.windowTop = maxTop
			}

			// Make sure cursor is visible in the new window size
			ensureCursorVisible(&m)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "?":
			// Toggle help tooltip
			m.showHelp = !m.showHelp

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				// Adjust window if needed
				if m.cursor < m.windowTop {
					m.windowTop = m.cursor
				}
			} else {
				// Wrap around to the bottom
				m.cursor = len(m.options) - 1
				// Adjust window if needed
				ensureCursorVisible(&m)
			}

		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
				// Adjust window if needed
				if m.cursor >= m.windowTop+m.windowHeight {
					m.windowTop = m.cursor - m.windowHeight + 1
				}
			} else {
				// Wrap around to the top
				m.cursor = 0
				m.windowTop = 0
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

		case "home", "g":
			// Jump to the top of the list
			m.cursor = 0
			m.windowTop = 0

		case "end", "G":
			// Jump to the bottom of the list
			if len(m.options) > 0 {
				m.cursor = len(m.options) - 1
				// Adjust window if needed
				if m.cursor >= m.windowTop+m.windowHeight {
					m.windowTop = m.cursor - m.windowHeight + 1
					if m.windowTop < 0 {
						m.windowTop = 0
					}
				}
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
	if m.quitting {
		return ""
	}

	// If we haven't received the window size yet, show a loading message
	if !m.ready {
		return "Loading checkbox menu..."
	}

	var sb strings.Builder

	sb.WriteString("Select resources to apply\n\n")

	// Calculate visible range
	start := m.windowTop
	if start < 0 {
		start = 0
	}

	end := m.windowTop + m.windowHeight - 1 // Reserve space for status bar
	if end > len(m.options) {
		end = len(m.options)
	}

	// Render visible options
	for i := start; i < end; i++ {
		option := m.options[i]

		// Show cursor if this is the selected option
		cursor := "  "
		if i == m.cursor {
			cursor = ui.GetCursorChar() + " "
		}

		// Determine the checkbox state
		checkedSymbol := "[ ] "
		if option.Checked {
			checkedSymbol = "[x] "
		}

		// Style based on selection state
		optNameStyle := nameStyle
		if i == m.cursor {
			// Highlight the cursor position
			if option.Checked {
				checkedSymbol = checkedStyle.Render("[x] ")
			} else {
				checkedSymbol = uncheckedStyle.Render("[ ] ")
			}
			cursor = cursorStyle.Render(cursor)
			optNameStyle = activeStyle
		} else {
			if option.Checked {
				checkedSymbol = checkedStyle.Render("[x] ")
			} else {
				checkedSymbol = uncheckedStyle.Render("[ ] ")
			}
		}

		// Render name with checkbox
		line := fmt.Sprintf("%s%s%s",
			cursor,
			checkedSymbol,
			optNameStyle.Render(option.Name))

		// Add description with appropriate color based on action type
		if option.Description != "" {
			var descStyle lipgloss.Style

			switch option.Description {
			case "create":
				descStyle = createStyle
			case "update":
				descStyle = updateStyle
			case "destroy", "replace":
				descStyle = destroyStyle
			default:
				descStyle = faintStyle
			}

			line += fmt.Sprintf(" - %s", descStyle.Render(option.Description))
		}

		// Highlight the current line with background
		if i == m.cursor {
			line = lipgloss.NewStyle().
				Background(lipgloss.Color("#333333")).
				Render(line)
		}

		sb.WriteString(line + "\n")
	}

	// Calculate the percentage
	var percentage int
	if len(m.options) <= 1 {
		percentage = 100
	} else if m.cursor <= 0 {
		percentage = 0
	} else if m.cursor >= len(m.options)-1 {
		percentage = 100
	} else {
		percentage = (m.cursor * 100) / (len(m.options) - 1)
	}

	// Add status line at the bottom
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CCCCCC")).
		Background(lipgloss.Color(ui.GetHexColorByName("highlight"))).
		Bold(true).
		Width(100).
		Padding(0, 1)

	// Create the status message with navigation info
	var statusMsg string
	if len(m.options) <= m.windowHeight-1 {
		// Everything fits on screen
		statusMsg = fmt.Sprintf("All %d items visible - Press ? for help", len(m.options))
	} else {
		// Show percentage and position
		statusMsg = fmt.Sprintf("Item %d of %d (%d%%) - Press ? for help",
			m.cursor+1, len(m.options), percentage)

		// Add hint about content above/below if applicable
		if start > 0 && end < len(m.options) {
			statusMsg += " - More items above and below"
		} else if start > 0 {
			statusMsg += " - More items above"
		} else if end < len(m.options) {
			statusMsg += " - More items below"
		}
	}

	// Add the status bar
	sb.WriteString(statusStyle.Render(statusMsg))

	// If help is toggled, show the help tooltip as a floating overlay
	if m.showHelp {
		// Generate the help content
		helpText := renderHelpTooltip()
		sb.WriteString("\n\n" + helpText)
	}

	return sb.String()
}

// Show displays a checkbox menu with the provided options.
func Show(options []Option) ([]Option, error) {
	if len(options) == 0 {
		return nil, nil
	}

	m := model{
		options:      options,
		cursor:       0,
		windowTop:    0,
		windowHeight: 25, // Default height, will be adjusted when we receive WindowSizeMsg
		ready:        false,
		showHelp:     false,
	}

	// Initialize styles
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

// renderHelpTooltip generates a help tooltip with keyboard shortcuts.
func renderHelpTooltip() string {
	width := 50
	padding := 1

	helpStyle := lipgloss.NewStyle().
		Width(width).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ui.GetHexColorByName("highlight"))).
		Padding(padding, padding)

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.GetHexColorByName("highlight"))).
		Bold(true).
		MarginBottom(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.GetHexColorByName("info")))

	var helpContent strings.Builder

	helpContent.WriteString(titleStyle.Render("Checkbox Selection Controls") + "\n\n")

	helpContent.WriteString(keyStyle.Render("j/down") + ": Move cursor down\n")
	helpContent.WriteString(keyStyle.Render("k/up") + ": Move cursor up\n")
	helpContent.WriteString(keyStyle.Render("space") + ": Toggle selection\n")
	helpContent.WriteString(keyStyle.Render("a") + ": Select all items\n")
	helpContent.WriteString(keyStyle.Render("n") + ": Deselect all items\n")
	helpContent.WriteString(keyStyle.Render("g/home") + ": Jump to first item\n")
	helpContent.WriteString(keyStyle.Render("G/end") + ": Jump to last item\n")
	helpContent.WriteString(keyStyle.Render("enter") + ": Confirm selection\n")
	helpContent.WriteString(keyStyle.Render("q") + ": Quit without selecting\n")
	helpContent.WriteString(keyStyle.Render("?") + ": Toggle this help\n")

	return helpStyle.Render(helpContent.String())
}

// ensureCursorVisible adjusts the view window to keep the cursor visible.
func ensureCursorVisible(m *model) {
	// Check if cursor is below the visible window
	if m.cursor >= m.windowTop+m.windowHeight {
		m.windowTop = m.cursor - m.windowHeight + 1
	}

	// Check if cursor is above the visible window
	if m.cursor < m.windowTop {
		m.windowTop = m.cursor
	}

	// Ensure windowTop doesn't go below 0
	if m.windowTop < 0 {
		m.windowTop = 0
	}

	// Ensure windowTop doesn't exceed max possible (total - visible)
	maxWindowTop := len(m.options) - m.windowHeight
	if maxWindowTop < 0 {
		maxWindowTop = 0
	}

	if m.windowTop > maxWindowTop {
		m.windowTop = maxWindowTop
	}
}
