package main

import (
	"fmt"
	"strings"
	"time"
)

// Spinner represents a loading animation
type Spinner struct {
	message string
	done    chan bool
	frames  []string
}

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		done:    make(chan bool),
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	go func() {
		for {
			for _, frame := range s.frames {
				select {
				case <-s.done:
					return
				default:
					fmt.Printf("\r%s %s", frame, s.message)
					time.Sleep(100 * time.Millisecond)
				}
			}
		}
	}()
}

// Stop ends the spinner animation
func (s *Spinner) Stop() {
	s.done <- true
	fmt.Printf("\r%s", strings.Repeat(" ", len(s.message)+2)) // Clear the line
}
