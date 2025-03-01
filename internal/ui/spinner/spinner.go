// Package spinner provides an animated terminal spinner for long-running operations.
package spinner

import (
	"fmt"
	"os"
	"sync"
	"time"

	"tfapp/internal/ui"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Available spinners
	spinners = []spinner.Spinner{
		spinner.MiniDot,
	}
	textStyle    = lipgloss.NewStyle() // Default color for text
	spinnerStyle = lipgloss.NewStyle()
)

// Initialize styles with proper colors
func init() {
	// Use the highlight color for the spinner
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ui.GetHexColorByName("highlight")))
	// Keep textStyle as default (no foreground color set)
}

// quitMsg is sent when the spinner should stop
type quitMsg struct{}

// model represents the spinner state
type model struct {
	spinner  spinner.Model
	message  string
	quitting bool
	err      error
	program  *tea.Program
	done     chan struct{}
	wg       sync.WaitGroup
}

// Spinner provides a terminal spinner with a message.
type Spinner struct {
	model *model
}

// New creates a new bubbletea-based spinner.
func New(message string) *Spinner {
	s := spinner.New()
	s.Spinner = spinners[0] // MiniDot spinner
	s.Style = spinnerStyle

	return &Spinner{
		model: &model{
			spinner: s,
			message: message,
			done:    make(chan struct{}),
		},
	}
}

// Init implements tea.Model.
func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update implements tea.Model.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	case quitMsg:
		m.quitting = true
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View implements tea.Model.
func (m model) View() string {
	if m.quitting {
		return ""
	}
	return fmt.Sprintf("%s %s", m.spinner.View(), textStyle.Render(m.message))
}

// Start begins the spinner animation.
func (s *Spinner) Start() {
	s.model.wg.Add(1)
	p := tea.NewProgram(s.model,
		tea.WithoutCatchPanics(),
		tea.WithMouseCellMotion(),
	)
	s.model.program = p

	go func() {
		defer s.model.wg.Done()
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running spinner: %v\n", err)
			os.Exit(1)
		}
		close(s.model.done)
	}()
}

// Stop ends the spinner animation.
func (s *Spinner) Stop() {
	if s.model.program != nil {
		s.model.program.Send(quitMsg{})

		// Wait for cleanup with timeout
		cleanup := make(chan struct{})
		go func() {
			s.model.wg.Wait()
			close(cleanup)
		}()

		select {
		case <-cleanup:
			// Normal cleanup completed
		case <-time.After(500 * time.Millisecond):
			// Timeout - force quit
			fmt.Print("\r") // Clear the spinner line
		}
	}
}
