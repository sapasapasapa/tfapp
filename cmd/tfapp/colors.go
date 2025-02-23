package main

import "strings"

const (
	colorRed    = "\033[1;31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
)

// colorize adds ANSI color codes to terraform plan output
func colorize(line string) string {
	line = strings.ReplaceAll(line, "destroyed", colorRed+"destroyed"+colorReset)
	line = strings.ReplaceAll(line, "replaced", colorRed+"replaced"+colorReset)
	line = strings.ReplaceAll(line, "created", colorGreen+"created"+colorReset)
	line = strings.ReplaceAll(line, "updated in-place", colorYellow+"updated in-place"+colorReset)
	return line
}
