package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Available spinners
	spinners = []spinner.Spinner{
		spinner.MiniDot,
	}
	textStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8239F3"))
)

// quitMsg is sent when the spinner should stop
type quitMsg struct{}

// spinnerModel represents the spinner state
type spinnerModel struct {
	spinner  spinner.Model
	message  string
	quitting bool
	err      error
	program  *tea.Program
	done     chan struct{}
	wg       sync.WaitGroup
}

// NewSpinner creates a new bubbletea-based spinner
func NewSpinner(message string) *spinnerModel {
	s := spinner.New()
	s.Spinner = spinners[0] // MiniDot spinner
	s.Style = spinnerStyle

	return &spinnerModel{
		spinner: s,
		message: message,
		done:    make(chan struct{}),
	}
}

// Init implements tea.Model
func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update implements tea.Model
func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

// View implements tea.Model
func (m spinnerModel) View() string {
	if m.quitting {
		return ""
	}
	return fmt.Sprintf("%s %s", m.spinner.View(), textStyle.Render(m.message))
}

// Start begins the spinner animation
func (m *spinnerModel) Start() {
	m.wg.Add(1)
	p := tea.NewProgram(m,
		tea.WithoutCatchPanics(),
		tea.WithMouseCellMotion(),
	)
	m.program = p

	go func() {
		defer m.wg.Done()
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running spinner: %v\n", err)
			os.Exit(1)
		}
		close(m.done)
	}()
}

// Stop ends the spinner animation
func (m *spinnerModel) Stop() {
	if m.program != nil {
		m.program.Send(quitMsg{})

		// Wait for cleanup with timeout
		cleanup := make(chan struct{})
		go func() {
			m.wg.Wait()
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
