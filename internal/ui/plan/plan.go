// Package plan provides an interactive Terraform plan viewer with collapsible sections.
package plan

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"tfapp/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TreeNode represents a node in the plan's resource tree.
type TreeNode struct {
	Text            string      // The text content of this node
	Children        []*TreeNode // Child nodes (nested blocks)
	Parent          *TreeNode   // Parent node (nil for root)
	Depth           int         // Depth in the tree
	Expanded        bool        // Whether this node is expanded
	Type            string      // Type of node (resource, block, attribute)
	IsRoot          bool        // Whether this is a root node
	Toggleable      bool        // Whether this node can be expanded/collapsed
	ChangeType      string      // Type of change (create, update, delete, replace)
	PreviousAddress string      // Previous address for moved resources
	IsDrifted       bool        // Whether this resource has drifted
	ActionReason    string      // Reason for the action (e.g., tainted)
}

// Model represents the state of the plan viewer.
type Model struct {
	nodes            []*TreeNode // All root-level nodes
	allNodes         []*TreeNode // All nodes (flattened)
	cursor           int         // Current cursor position
	windowTop        int         // The top line of the window being displayed
	windowHeight     int         // Height of visible window
	horizontalOffset int         // Horizontal scroll position
	width            int         // Width of the terminal window for text wrapping
	quitting         bool        // Whether the user is quitting
	ready            bool        // Whether we've received the window size yet
	showHelp         bool        // Whether to show the help tooltip
	inputSearchModel bool        // Waiting user to insert search string
	searchMode       bool        // Whether to show the search results
	searchString     string      // The search string
	searchResults    []int       // The search results
	searchIndex      int         // The index of the search result
}

// New creates a new plan viewer model.
func New(planOutput string) Model {
	nodes := parsePlan(planOutput)

	// Set only the root section nodes to expanded by default, collapse all others
	for _, node := range nodes {
		if node.Type == "section" || node.IsRoot {
			node.Expanded = true
		} else {
			node.Expanded = false
		}

		// Collapse all children
		for _, child := range node.Children {
			collapseAllNodes(child)
		}
	}

	// Get all nodes in flattened list, respecting expansion state
	allNodes := flattenNodes(nodes)

	return Model{
		nodes:            nodes,
		allNodes:         allNodes,
		cursor:           0,
		windowTop:        0,
		windowHeight:     25, // Show approximately 25 lines at a time for better visibility
		horizontalOffset: 0,  // Start at the leftmost position
		width:            80, // Default terminal width
		quitting:         false,
		ready:            false,
		showHelp:         false,
		inputSearchModel: false,
		searchMode:       false,
		searchString:     "",
		searchResults:    []int{},
		searchIndex:      0,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	// Just return nil since windowHeight will be updated when we receive a WindowSizeMsg
	return nil
}

// Update handles user input and updates the model accordingly.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

		// Store terminal width for text wrapping
		m.width = msg.Width

		// Mark as ready now that we've received window dimensions
		m.ready = true

		// If the window height changed significantly, ensure the cursor remains visible
		if oldHeight != m.windowHeight {
			visibleNodes := getVisibleNodes(m.nodes)

			// Make sure we don't exceed the maximum possible windowTop
			maxTop := len(visibleNodes) - m.windowHeight
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
		if !m.searchMode && !m.inputSearchModel {
			switch msg.String() {
			case "q", "b", "ctrl+c":
				m.quitting = true
				return m, tea.Quit

			case "?":
				// Toggle help tooltip
				m.showHelp = !m.showHelp

			case "up", "k":
				// Get visible nodes and check if we can move up
				if m.cursor > 0 {
					m.cursor--
					// Use ensureCursorVisible to properly adjust the window
					ensureCursorVisible(&m)
				}

			case "down", "j":
				// Get visible nodes and check if we can move down
				visibleNodes := getVisibleNodes(m.nodes)
				if m.cursor < len(visibleNodes)-1 {
					m.cursor++
					// Use ensureCursorVisible to properly adjust the window
					ensureCursorVisible(&m)
				}

			case "right", "l":
				// Horizontal scrolling to the right
				m.horizontalOffset += 10
				if m.horizontalOffset > 500 {
					m.horizontalOffset = 500 // Set a reasonable maximum
				}

			case "left", "h":
				// Horizontal scrolling to the left
				m.horizontalOffset -= 10
				if m.horizontalOffset < 0 {
					m.horizontalOffset = 0
				}

			case " ":
				// Toggle expansion of the current node
				visibleNodes := getVisibleNodes(m.nodes)
				if m.cursor >= 0 && m.cursor < len(visibleNodes) {
					currentNode := visibleNodes[m.cursor]
					if len(currentNode.Children) > 0 && currentNode.Toggleable {
						// Toggle the expansion state
						currentNode.Expanded = !currentNode.Expanded
						// Refresh the list of visible nodes
						m.allNodes = flattenNodes(m.nodes)

						// Adjust cursor if it's now beyond the visible nodes
						newVisibleNodes := getVisibleNodes(m.nodes)
						if m.cursor >= len(newVisibleNodes) {
							m.cursor = len(newVisibleNodes) - 1
							if m.cursor < 0 {
								m.cursor = 0
							}
						}

						// Ensure cursor is in view
						ensureCursorVisible(&m)
					}
				}

			// Reset horizontal position when moving to parent or collapsing
			case "backspace":
				// Reset horizontal position
				m.horizontalOffset = 0

				// And do the same for left key behavior - collapse current node or move to parent
				visibleNodes := getVisibleNodes(m.nodes)
				if m.cursor >= 0 && m.cursor < len(visibleNodes) {
					currentNode := visibleNodes[m.cursor]
					if currentNode.Expanded && len(currentNode.Children) > 0 {
						// Collapse this node
						currentNode.Expanded = false
						// Refresh the list of visible nodes
						m.allNodes = flattenNodes(m.nodes)
					} else if currentNode.Parent != nil {
						// Find parent in visible nodes
						for i, node := range visibleNodes {
							if node == currentNode.Parent {
								m.cursor = i
								break
							}
						}
					}
				}

			case "enter":
				// Toggle expansion of the current node
				visibleNodes := getVisibleNodes(m.nodes)
				if m.cursor >= 0 && m.cursor < len(visibleNodes) {
					currentNode := visibleNodes[m.cursor]
					if len(currentNode.Children) > 0 && currentNode.Toggleable {
						if currentNode.Expanded {
							// Collapse all children while keeping the current node expanded
							collapseAllNodes(currentNode)
							// No need to set currentNode.Expanded = true since we don't change it now
						} else {
							expandAllNodes(currentNode)
						}
					}

					// Refresh the list of all nodes
					m.allNodes = flattenNodes(m.nodes)

					// Ensure cursor is in view
					ensureCursorVisible(&m)
				}

			case "a":
				// Expand all nodes recursively
				for _, rootNode := range m.nodes {
					expandAllNodes(rootNode)
				}

				// Refresh the list of all nodes
				m.allNodes = flattenNodes(m.nodes)

				// Ensure cursor is visible after expansion
				ensureCursorVisible(&m)

			case "A":
				// Collapse all nodes with children
				for _, node := range m.allNodes {
					if len(node.Children) > 0 && (!node.IsRoot || !node.Parent.IsRoot) {
						node.Expanded = false
					}
				}

				// Refresh the list of all nodes
				m.allNodes = flattenNodes(m.nodes)

				// Set cursor to first line and ensure it's visible
				m.cursor = 0

				// Ensure cursor is visible after collapse
				ensureCursorVisible(&m)

			case "n":
				// Jump to the next root node of resource type at depth 0
				visibleNodes := getVisibleNodes(m.nodes)
				if len(visibleNodes) > 0 {
					// Start searching from the node after current cursor position
					startPos := m.cursor + 1
					if startPos >= len(visibleNodes) {
						startPos = 0 // Wrap around to the beginning
					}

					// First, search from cursor to end
					found := false
					for i := startPos; i < len(visibleNodes); i++ {
						if isRootResource(visibleNodes[i]) {
							m.cursor = i
							found = true
							break
						}
					}

					// If not found and we started after position 0, search from beginning to cursor
					if !found && startPos > 0 {
						for i := 0; i < startPos; i++ {
							if isRootResource(visibleNodes[i]) {
								m.cursor = i
								found = true
								break
							}
						}
					}

					// Ensure the cursor is visible in the window
					ensureCursorVisible(&m)
				}

			case "N":
				// Jump to the previous root node of resource type at depth 0
				visibleNodes := getVisibleNodes(m.nodes)
				if len(visibleNodes) > 0 {
					// Start searching from the node before current cursor position
					startPos := m.cursor - 1
					if startPos < 0 {
						startPos = len(visibleNodes) - 1 // Wrap around to the end
					}

					// First, search from cursor to beginning
					found := false
					for i := startPos; i >= 0; i-- {
						if isRootResource(visibleNodes[i]) {
							m.cursor = i
							found = true
							break
						}
					}

					// If not found and we started before the end, search from end to cursor
					if !found && startPos < len(visibleNodes)-1 {
						for i := len(visibleNodes) - 1; i > startPos; i-- {
							if isRootResource(visibleNodes[i]) {
								m.cursor = i
								found = true
								break
							}
						}
					}

					// Ensure the cursor is visible in the window
					ensureCursorVisible(&m)
				}

			case "home", "g":
				// Jump to the top of the plan
				m.cursor = 0
				m.horizontalOffset = 0 // Reset horizontal position
				ensureCursorVisible(&m)

			case "end", "G":
				// Jump to the bottom of the plan
				visibleNodes := getVisibleNodes(m.nodes)
				if len(visibleNodes) > 0 {
					// Set cursor directly to the last visible node
					m.cursor = len(visibleNodes) - 1
					m.horizontalOffset = 0 // Reset horizontal position
					// Ensure cursor is visible
					ensureCursorVisible(&m)
				}
			case "/":
				// Search for a resource by name
				m.inputSearchModel = true
			}
		} else if m.inputSearchModel {
			switch msg.String() {
			case "enter":
				if len(m.searchString) > 0 {
					m.searchMode = true
					m.inputSearchModel = false
					m.searchResults = m.getSearchResults()
					if len(m.searchResults) > 0 {
						m.searchIndex = 0
						m.cursor = m.searchResults[m.searchIndex]
						ensureCursorVisible(&m)
					}
				} else {
					// If search string is empty, exit search mode
					m.searchString = ""
					m.inputSearchModel = false
				}
			case "esc", "ctrl+c":
				m.searchMode = false
				m.inputSearchModel = false
				m.searchString = ""
			case "backspace":
				// Handle backspace for search string
				if len(m.searchString) > 0 {
					m.searchString = m.searchString[:len(m.searchString)-1]
				}
			default:
				// Only add printable characters to the search string
				if len(msg.String()) > 0 {
					m.searchString += msg.String()
				}
			}
		} else if m.searchMode {
			switch msg.String() {
			case "n":
				m.findNext(1)
			case "N":
				m.findNext(-1)
			default:
				m.searchMode = false
				m.searchString = ""
			}
		}

	case tea.MouseMsg:
		// Handle mouse wheel events for scrolling
		if msg.Action == tea.MouseActionPress {
			visibleNodes := getVisibleNodes(m.nodes)
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				// Scroll up (same as 'k' key)
				if m.cursor > 0 {
					m.cursor--
					ensureCursorVisible(&m)
				}
			case tea.MouseButtonWheelDown:
				// Scroll down (same as 'j' key)
				if m.cursor < len(visibleNodes)-1 {
					m.cursor++
					ensureCursorVisible(&m)
				}
			case tea.MouseButtonWheelRight:
				// Horizontal scroll left (same as 'h' key)
				m.horizontalOffset -= 20
				if m.horizontalOffset < 0 {
					m.horizontalOffset = 0
				}
			case tea.MouseButtonWheelLeft:
				// Horizontal scroll right (same as 'l' key)
				m.horizontalOffset += 20
				if m.horizontalOffset > 500 {
					m.horizontalOffset = 500 // Set a reasonable maximum
				}
			}
		}
	}

	return m, nil
}

// View renders the model as a string.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	// If we haven't received the window size yet, show a loading message
	if !m.ready {
		return "Loading plan viewer..."
	}

	var sb strings.Builder

	// Get visible nodes accounting for expansion state
	visibleNodes := getVisibleNodes(m.nodes)
	totalNodes := len(visibleNodes)

	// Calculate visible range
	start := m.windowTop
	if start < 0 {
		start = 0
	}

	end := m.windowTop + m.windowHeight
	if end > totalNodes {
		end = totalNodes
	}

	// Reserve last line for status bar, adjust rendering height
	contentHeight := m.windowHeight - 1
	if contentHeight < 1 {
		contentHeight = 1 // Ensure at least one line for content
	}

	// Adjust end for content area
	contentEnd := start + contentHeight
	if contentEnd > totalNodes {
		contentEnd = totalNodes
	}

	// Render visible nodes
	linesRendered := 0
	for i := start; i < contentEnd && linesRendered < contentHeight; i++ {
		node := visibleNodes[i]

		// Indent based on depth
		indent := strings.Repeat("  ", node.Depth)

		// Show cursor if this is the selected node - make it more prominent
		cursor := "  "
		if i == m.cursor {
			// Use a more prominent cursor character and styling
			cursorChar := ui.GetCursorChar()
			cursor = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ui.GetHexColorByName("highlight"))).
				Bold(true).
				Render(cursorChar) + " "
		}

		// Show expansion indicator if this node has children
		expandChar := "  "
		if len(node.Children) > 0 && node.Toggleable {
			if node.Expanded {
				expandChar = ui.ColorInfo + "▼ " + ui.ColorForegroundReset
			} else {
				expandChar = ui.ColorHighlight + "▶ " + ui.ColorForegroundReset
			}
		}

		// For section headers, don't show expansion indicators
		if node.Type == "section_header" || node.Type == "header" {
			expandChar = "  "
		}

		// Style the line based on node type
		var line string
		if m.searchMode && m.searchString != "" {
			// Highlight search matches
			nodeText := node.Text
			if strings.Contains(nodeText, m.searchString) {
				// Split the text by the search string to highlight matches
				parts := strings.Split(nodeText, m.searchString)
				highlightedText := parts[0]
				for j := 1; j < len(parts); j++ {
					if m.cursor == i {
						// Replace the simple color highlight with lipgloss styling for both foreground and background
						searchMatchStyle := lipgloss.NewStyle().
							Foreground(lipgloss.Color(ui.GetHexColorByName("success"))).
							Background(lipgloss.Color("#333333")).
							Bold(true)
						highlightedText += searchMatchStyle.Render(m.searchString) + parts[j]
					} else {
						highlightedText += ui.ColorHighlight + m.searchString + ui.ColorForegroundReset + parts[j]
					}
				}
				line = indent + expandChar + highlightedText
			} else {
				line = indent + expandChar + nodeText
			}
		} else {
			line = indent + expandChar + node.Text
		}

		// Apply custom colorization based on node type
		var colorized string

		// Special handling for different node types
		if node.Type == "header" {
			// Apply bold formatting and background color to main header
			colorized = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#4a2a8a")). // Purple background for main header
				Render(line)
		} else if node.Type == "section_header" {
			// Use highlight color for all section headers
			colorized = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color(ui.GetHexColorByName("highlight"))).
				Render(line)
		} else if node.IsDrifted {
			// Apply drift color only to the "has drifted" phrase
			if strings.Contains(line, "has drifted") {
				// Split the line at "has drifted" to color only that part
				parts := strings.SplitN(line, "has drifted", 2)
				colorized = parts[0] + lipgloss.NewStyle().
					Foreground(lipgloss.Color("#FF9900")). // Orange color for drift phrase only
					Render("has drifted") + parts[1]
			} else {
				colorized = line
			}
		} else if node.Type == "resource" {
			// Resources are already colorized by the ui.Colorize function
			colorized = ui.Colorize(line)
		} else {
			// Apply color based on the node's change type
			switch node.ChangeType {
			case "create":
				if strings.Contains(line, "+") {
					colorized = strings.Replace(line, "+", ui.ColorSuccess+"+"+ui.ColorForegroundReset, 1)
				} else if strings.HasPrefix(strings.TrimSpace(line), "}") {
					// Don't color closing braces
					colorized = line
				} else {
					colorized = ui.ColorSuccess + line + ui.ColorForegroundReset
				}
			case "delete", "destroy":
				if strings.Contains(line, "-") {
					colorized = strings.Replace(line, "-", ui.ColorError+"-"+ui.ColorForegroundReset, 1)
				} else if strings.HasPrefix(strings.TrimSpace(line), "}") {
					// Don't color closing braces
					colorized = line
				} else {
					colorized = ui.ColorError + line + ui.ColorForegroundReset
				}
			case "update", "replace":
				if strings.Contains(line, "~") {
					colorized = strings.Replace(line, "~", ui.ColorWarning+"~"+ui.ColorForegroundReset, 1)
				} else if strings.Contains(line, "-/+") {
					colorized = strings.Replace(line, "-/+", ui.ColorError+"-"+ui.ColorForegroundReset+"/"+ui.ColorSuccess+"+"+ui.ColorForegroundReset, 1)
				} else if strings.HasPrefix(strings.TrimSpace(line), "}") {
					colorized = line
				} else {
					colorized = ui.ColorWarning + line + ui.ColorForegroundReset
				}
			case "drift":
				// Apply a distinctive color only to the "has drifted" phrase
				if strings.Contains(line, "has drifted") {
					// Split the line at "has drifted" to color only that part
					parts := strings.SplitN(line, "has drifted", 2)
					colorized = parts[0] + lipgloss.NewStyle().
						Foreground(lipgloss.Color("#FF9900")). // Orange color for drift phrase only
						Render("has drifted") + parts[1]
				} else {
					colorized = line
				}
			case "move":
				// Special color for moved resources
				colorized = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#00CCFF")). // Light blue for moved resources
					Render(line)
			default:
				// For comments (like "# (5 unchanged attributes hidden)")
				if strings.HasPrefix(strings.TrimSpace(line), "#") {
					// Check for status text with "will be"
					if strings.Contains(line, "will be") {
						// Color the status text appropriately
						if strings.Contains(line, "will be created") ||
							strings.Contains(line, "will be create") {
							colorized = ui.ColorSuccess + line + ui.ColorForegroundReset
						} else if strings.Contains(line, "will be destroyed") ||
							strings.Contains(line, "will be destroy") {
							colorized = ui.ColorError + line + ui.ColorForegroundReset
						} else if strings.Contains(line, "will be updated") ||
							strings.Contains(line, "will be update") ||
							strings.Contains(line, "will be replaced") ||
							strings.Contains(line, "will be replace") {
							colorized = ui.ColorWarning + line + ui.ColorForegroundReset
						} else {
							colorized = ui.ColorInfo + line + ui.ColorForegroundReset
						}
					} else if strings.Contains(line, "unchanged") && strings.Contains(line, "hidden") {
						// Use cyan color specifically for "unchanged ... hidden" comments
						colorized = lipgloss.NewStyle().
							Foreground(lipgloss.Color("#00FFFF")). // Bright cyan color
							Render(line)
					} else {
						colorized = ui.ColorInfo + line + ui.ColorForegroundReset
					}
				} else if node.Type == "closing_brace" {
					// Never color closing braces
					colorized = line
				} else {
					colorized = ui.Colorize(line)
				}
			}
		}

		// Don't highlight the cursor line yet - we'll do that after text processing

		// Apply horizontal scrolling by truncating the left portion of colorized text
		// We need to be careful with ANSI color codes, which should not be counted in string length
		visibleText := colorized

		// Figure out available width for text after cursor and indentation
		cursorWidth := len(cursor) // Width of cursor area

		// Extract all ANSI codes for parsing text length and preserving them
		ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
		ansiMatches := ansiRegex.FindAllStringIndex(colorized, -1)

		// Create a map of ANSI code positions
		ansiCodes := make(map[int]string)
		ansiLengths := make(map[int]int)

		for _, match := range ansiMatches {
			ansiCode := colorized[match[0]:match[1]]
			ansiCodes[match[0]] = ansiCode
			ansiLengths[match[0]] = match[1] - match[0]
		}

		// Calculate the visual length (text without ANSI codes)
		visualLength := 0
		for pos := 0; pos < len(colorized); pos++ {
			if _, found := ansiCodes[pos]; found {
				pos += ansiLengths[pos] - 1
				continue
			}
			visualLength++
		}

		// Only apply horizontal scrolling if this line exceeds the terminal width and scrolling is active
		if m.horizontalOffset > 0 && visualLength > m.width {
			// Find the starting position
			visualPos := 0
			actualStart := 0

			for pos := 0; pos < len(colorized) && visualPos < m.horizontalOffset; pos++ {
				if _, found := ansiCodes[pos]; found {
					pos += ansiLengths[pos] - 1
					continue
				}
				visualPos++
				actualStart = pos + 1
			}

			// Extract all ANSI codes that were active before the starting position
			var activeStyleCodes []string
			for pos := 0; pos < actualStart; pos++ {
				if code, found := ansiCodes[pos]; found {
					if !strings.Contains(code, "[0m") {
						// Add to active style codes
						activeStyleCodes = append(activeStyleCodes, code)
					} else {
						// Reset clears all active codes
						activeStyleCodes = nil
					}
				}
			}

			// Build the visible part of the text
			var sb strings.Builder

			// Add leading ellipsis if we're not at the start
			if m.horizontalOffset > 0 {
				sb.WriteString("... ")
			}

			// Add active styles at the beginning
			for _, code := range activeStyleCodes {
				sb.WriteString(code)
			}

			// Extract all text from the starting position without any length limit
			for pos := actualStart; pos < len(colorized); pos++ {
				if code, found := ansiCodes[pos]; found {
					sb.WriteString(code)
					pos += len(code) - 1
					continue
				}

				sb.WriteByte(colorized[pos])
			}

			visibleText = sb.String()
		} else {
			// No horizontal scrolling - show the entire text without any trimming
			visibleText = colorized
		}

		// Now apply highlighting to the processed visible text if this is the cursor line
		if i == m.cursor {
			// First determine the visual width of the text (without ANSI codes)
			cleanText := ansiRegex.ReplaceAllString(visibleText, "")
			visualWidth := len(cleanText)

			// Calculate remaining width to fill the terminal width
			remainingWidth := m.width - cursorWidth - visualWidth - 2

			// Only add padding if we need to fill extra space
			if remainingWidth > 0 {
				// Add spaces to pad to the terminal width
				padding := strings.Repeat(" ", remainingWidth)
				visibleText += padding
			}

			// Apply highlighting with lipgloss style
			visibleText = lipgloss.NewStyle().
				Background(lipgloss.Color("#555555")).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true).
				Render(visibleText)
		}

		// Write the line to output with cursor
		sb.WriteString(cursor + visibleText + "\n")
		linesRendered++
	}

	// Calculate the percentage
	var percentage int
	if totalNodes <= 1 {
		percentage = 100
	} else if m.cursor <= 0 {
		percentage = 0
	} else if m.cursor >= totalNodes-1 {
		percentage = 100
	} else {
		percentage = (m.cursor * 100) / (totalNodes - 1)
	}

	// Add status line at the bottom
	statusStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#5300D1")).
		Bold(true).
		Width(100).
		Padding(0, 1)

	// Create the status message with navigation info
	var statusMsg string = ""
	if totalNodes <= contentHeight {
		statusMsg += "All items visible"
	} else {
		statusMsg += fmt.Sprintf("Line %d of %d (%d%%)",
			m.cursor+1, totalNodes, percentage)
	}

	if m.searchMode || m.inputSearchModel {
		if m.searchMode && len(m.searchResults) > 0 {
			statusMsg += fmt.Sprintf(" - Search: %s%s (%d/%d matches)%s",
				ui.ColorSuccess, m.searchString, m.searchIndex+1, len(m.searchResults), ui.ColorForegroundReset)
		} else if m.searchMode && len(m.searchResults) == 0 {
			statusMsg += fmt.Sprintf(" - Search: %s%s (No matches)%s",
				ui.ColorError, m.searchString, ui.ColorForegroundReset)
		} else if m.inputSearchModel {
			// Show a cursor indicator in the search input
			statusMsg += fmt.Sprintf(" - Search: %s|", m.searchString)
		} else {
			statusMsg += fmt.Sprintf(" - Search: %s", m.searchString)
		}
	} else {
		statusMsg += " - Press ? for help"
	}

	if totalNodes > contentHeight {
		// Add hint about content above/below if applicable
		if start > 0 && contentEnd < totalNodes {
			statusMsg += " - More content above and below"
		} else if start > 0 {
			statusMsg += " - More content above"
		} else if contentEnd < totalNodes {
			statusMsg += " - More content below"
		}
	}

	// Add the status bar
	sb.WriteString(statusStyle.Render(statusMsg))

	// If help is toggled, show the help tooltip as a floating overlay
	if m.showHelp {
		// Generate the help content
		helpText := renderHelpTooltip()

		// Return the content with the help dialog appended
		// The help dialog will appear to float over the content
		return sb.String() + "\n\n" + helpText
	}

	return sb.String()
}

// Show displays the plan viewer and returns when the user quits.
func Show(planOutput string) error {
	model := New(planOutput)

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Capture mouse events
	)
	_, err := p.Run()

	return err
}

// parsePlan parses the terraform plan output and builds a tree of nodes.
// It now accepts both plain text and JSON format
func parsePlan(planOutput string) []*TreeNode {
	// Check if the input is JSON
	if strings.TrimSpace(planOutput)[0] == '{' {
		return parseTerraformPlanJSON(planOutput)
	}

	// Continue with the existing text parsing logic
	lines := strings.Split(planOutput, "\n")

	// Root node for the entire plan
	root := &TreeNode{
		Text:       "Terraform Plan",
		Expanded:   true, // Root should always be expanded
		IsRoot:     true,
		Toggleable: true,
	}

	// Create a stack for tracking the current path in the tree
	stack := []*TreeNode{root}

	// Parse the plan output line by line
	var currentResourceNode *TreeNode
	var blockLevel int
	var inResourceBlock bool

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Calculate the indentation level
		indent := len(line) - len(strings.TrimLeft(line, " "))
		trimmedLine := strings.TrimSpace(line)

		// Check if this is a resource header line (with # prefix)
		if strings.Contains(line, "#") && (strings.Contains(line, "will be") || strings.Contains(line, "must be")) {

			// Start a new resource node
			resourceNode := &TreeNode{
				Text:       strings.TrimSpace(line),
				Expanded:   false, // Resources are collapsed by default
				Type:       "resource",
				Depth:      indent / 2,
				Parent:     root,
				Toggleable: true,
			}

			// Check if the next line is a continuation (reason for destruction)
			if i+1 < len(lines) &&
				strings.Contains(line, " will be destroyed") &&
				strings.TrimSpace(lines[i+1]) != "" &&
				strings.Contains(lines[i+1], "# (because") {
				// Add the reason to the current node text
				resourceNode.Text += "\n" + strings.TrimSpace(lines[i+1])
				i++ // Skip the next line since we've incorporated it
			}

			root.Children = append(root.Children, resourceNode)
			currentResourceNode = resourceNode
			stack = []*TreeNode{root, currentResourceNode}
			blockLevel = 0
			inResourceBlock = false
			continue
		}

		// Check if this is the start of resource definition block (format: "- resource "type" "name" {")
		if currentResourceNode != nil && !inResourceBlock &&
			(strings.HasPrefix(trimmedLine, "- resource ") ||
				strings.HasPrefix(trimmedLine, "+ resource ") ||
				strings.HasPrefix(trimmedLine, "~ resource ")) &&
			strings.Contains(trimmedLine, "{") {
			inResourceBlock = true
		}

		// Skip processing if we haven't found a resource yet
		if currentResourceNode == nil {
			continue
		}

		// Track block level based on braces
		if strings.Contains(line, "{") {
			blockLevel++
		}
		if strings.Contains(line, "}") {
			blockLevel--
		}

		// Create a node for this line
		depth := indent / 2
		nodeType := "attribute"

		// If the line contains an opening brace, it's a block
		if strings.Contains(line, "{") {
			nodeType = "block"
		}

		// Create the node
		node := &TreeNode{
			Text:       strings.TrimSpace(line),
			Expanded:   false, // Blocks are collapsed by default
			Type:       nodeType,
			Depth:      depth,
			Toggleable: true, // Regular nodes are toggleable by default
		}

		// Find the appropriate parent based on indentation
		for len(stack) > 1 && depth <= stack[len(stack)-1].Depth {
			stack = stack[:len(stack)-1] // Pop from stack
		}

		// Set parent and add to children
		parent := stack[len(stack)-1]
		node.Parent = parent
		parent.Children = append(parent.Children, node)

		// Push this node to the stack if it's a block
		if nodeType == "block" {
			stack = append(stack, node)
		}
	}

	// Return only the root's children (the resource nodes)
	return root.Children
}

func getVisibleNodes(nodes []*TreeNode) []*TreeNode {
	var result []*TreeNode

	for _, node := range nodes {
		result = append(result, node)
		if node.Expanded {
			result = append(result, getVisibleNodes(node.Children)...)
		}
	}

	return result
}

// ensureCursorVisible ensures the cursor is visible within the window.
func ensureCursorVisible(m *Model) {
	visibleNodes := getVisibleNodes(m.nodes)

	// Make sure cursor is within visible nodes range
	if m.cursor >= len(visibleNodes) {
		m.cursor = len(visibleNodes) - 1
	}

	// Ensure cursor is never negative
	if m.cursor < 0 {
		m.cursor = 0
	}

	// Content height (accounting for status bar)
	effectiveWindowHeight := m.windowHeight - 1
	if effectiveWindowHeight < 1 {
		effectiveWindowHeight = 1
	}

	// Always check if cursor is outside visible window, regardless of order
	cursorBelowWindow := m.cursor >= m.windowTop+effectiveWindowHeight
	cursorAboveWindow := m.cursor < m.windowTop

	// Handle case where cursor is below window
	if cursorBelowWindow {
		// Cursor is below the window, adjust windowTop to show cursor at bottom of window
		m.windowTop = m.cursor - effectiveWindowHeight + 1
	}

	// Handle case where cursor is above window
	if cursorAboveWindow {
		// Cursor is above the window, adjust windowTop to show cursor at top
		m.windowTop = m.cursor
	}

	// Add a buffer space at the top when possible to provide context
	// (don't do this when cursor is at the very top)
	if m.cursor > 2 && m.windowTop == m.cursor {
		// Add some context lines above the cursor (show 2 lines above when possible)
		m.windowTop = m.cursor - 2
	}

	// Final safety checks
	// Ensure windowTop is not negative
	if m.windowTop < 0 {
		m.windowTop = 0
	}

	// Double-check cursor is visible after all adjustments
	if m.cursor < m.windowTop || m.cursor >= m.windowTop+effectiveWindowHeight {
		if m.cursor < effectiveWindowHeight {
			// If cursor is near the top, show from the beginning
			m.windowTop = 0
		} else {
			// Otherwise center cursor in the window
			m.windowTop = m.cursor - (effectiveWindowHeight / 2)
			if m.windowTop < 0 {
				m.windowTop = 0
			}
		}
	}
}

// expandAllNodes recursively expands a node and all its children
func expandAllNodes(node *TreeNode) {
	if len(node.Children) > 0 {
		node.Expanded = true
		for _, child := range node.Children {
			expandAllNodes(child)
		}
	}
}

// collapseAllNodes recursively collapses a node's children and their descendants
func collapseAllNodes(node *TreeNode) {
	// We don't collapse the node itself, only its children and descendants
	node.Expanded = false
	for _, child := range node.Children {
		// Collapse each child
		child.Expanded = false
		// And recursively collapse its children
		collapseAllNodes(child)
	}
}

// renderHelpTooltip generates a help tooltip with all navigation commands
func renderHelpTooltip() string {
	helpStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ui.GetHexColorByName("highlight"))).
		Padding(1, 2).
		Width(60)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ui.GetHexColorByName("info"))).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#DDDDDD"))

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#2a2a6a")). // Match the section header color
		Bold(true).
		Padding(0, 1)

	// Create help content with key bindings and descriptions
	keys := []struct {
		key  string
		desc string
	}{
		{"↑/k", "Move cursor up"},
		{"↓/j", "Move cursor down"},
		{"→/l", "Scroll right (view more text)"},
		{"←/h", "Scroll left (view beginning of text)"},
		{"Space", "Expand current node"},
		{"Backspace", "Reset horizontal position and collapse node"},
		{"Enter", "Expand current node and all its children"},
		{"a", "Expand all nodes"},
		{"A", "Collapse all nodes except root level"},
		{"n", "Jump to next root resource (in normal mode) or next search match (in search mode)"},
		{"N", "Jump to previous root resource (in normal mode) or previous search match (in search mode)"},
		{"Home/g", "Jump to the top"},
		{"End/G", "Jump to the bottom"},
		{"/", "Start search mode"},
		{"Esc", "Exit search mode"},
		{"?", "Toggle this help dialog"},
		{"q/Ctrl+c", "Quit"},
	}

	var helpContent strings.Builder
	helpContent.WriteString(headerStyle.Render("Navigation Commands") + "\n\n")

	// Format each key binding with description
	for _, item := range keys {
		line := fmt.Sprintf("%s  %s\n",
			keyStyle.Render(item.key),
			descStyle.Render(item.desc))
		helpContent.WriteString(line)
	}

	// Add color coding information
	helpContent.WriteString("\n" + headerStyle.Render("Color Coding") + "\n\n")

	colorInfo := []struct {
		sample string
		desc   string
	}{
		{ui.ColorSuccess + "■■■" + ui.ColorForegroundReset, "Resources to be created"},
		{ui.ColorError + "■■■" + ui.ColorForegroundReset, "Resources to be destroyed"},
		{ui.ColorWarning + "■■■" + ui.ColorForegroundReset, "Resources to be updated/replaced"},
		{"", ""}, // Spacer
	}

	// Custom colors that might not be in the UI package
	driftColor := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF9900")).
		Render("■■■")
	moveColor := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00CCFF")).
		Render("■■■")

	// Format color coding information
	for _, item := range colorInfo {
		if item.sample == "" {
			helpContent.WriteString("\n")
			continue
		}
		line := fmt.Sprintf("%s  %s\n",
			item.sample,
			descStyle.Render(item.desc))
		helpContent.WriteString(line)
	}

	// Add special colors
	helpContent.WriteString(fmt.Sprintf("%s  %s\n",
		driftColor,
		descStyle.Render("Resources that have drifted outside of Terraform")))

	helpContent.WriteString(fmt.Sprintf("%s  %s\n",
		moveColor,
		descStyle.Render("Resources to be moved")))

	return helpStyle.Render(helpContent.String())
}

func (m *Model) getSearchResults() []int {
	// Expand all nodes recursively
	for _, rootNode := range m.nodes {
		expandAllNodes(rootNode)
	}

	// Refresh the list of all nodes
	m.allNodes = flattenNodes(m.nodes)

	results := []int{}
	for i, node := range m.allNodes {
		if strings.Contains(node.Text, m.searchString) {
			results = append(results, i)
		}
	}
	return results
}

func (m *Model) findNext(direction int) {
	if len(m.searchResults) == 0 {
		return
	}

	m.searchIndex += direction
	if m.searchIndex < 0 {
		m.searchIndex = len(m.searchResults) - 1
	}

	if m.searchIndex >= len(m.searchResults) {
		m.searchIndex = 0
	}

	m.cursor = m.searchResults[m.searchIndex]
	ensureCursorVisible(m)
}

// flattenNodes flattens the node tree into a single list, respecting expansion state.
func flattenNodes(nodes []*TreeNode) []*TreeNode {
	var result []*TreeNode

	for _, node := range nodes {
		result = append(result, node)
		if node.Expanded {
			result = append(result, flattenNodes(node.Children)...)
		}
	}

	return result
}

// createClosingBrace creates a closing brace node with consistent type and formatting
func createClosingBrace(depth int, parent *TreeNode) *TreeNode {
	return &TreeNode{
		Text:       "}",
		Expanded:   true,
		Type:       "closing_brace",
		Depth:      depth,
		Parent:     parent,
		Toggleable: false,
		ChangeType: "no-op",
	}
}

// isEffectivelyEqual compares two values and determines if they're effectively equal
// This handles special cases like empty strings, empty maps, and nulls
func isEffectivelyEqual(a, b interface{}) bool {
	// If both are nil or exactly equal, they're equal
	if a == nil && b == nil {
		return true
	}
	if reflect.DeepEqual(a, b) {
		return true
	}

	// Check if one is nil and the other is an empty string
	aStr, aIsStr := a.(string)
	if aIsStr && b == nil && aStr == "" {
		return true
	}

	bStr, bIsStr := b.(string)
	if bIsStr && a == nil && bStr == "" {
		return true
	}

	// Check for empty strings
	if aIsStr && bIsStr && (aStr == "" && bStr == "") {
		return true
	}

	// Check for empty maps
	aMap, aIsMap := a.(map[string]interface{})
	bMap, bIsMap := b.(map[string]interface{})
	if aIsMap && bIsMap && (len(aMap) == 0 && len(bMap) == 0) {
		return true
	}

	// Map to nil comparison
	if aIsMap && b == nil && len(aMap) == 0 {
		return true
	}
	if bIsMap && a == nil && len(bMap) == 0 {
		return true
	}

	// Check for empty arrays
	aSlice, aIsSlice := a.([]interface{})
	bSlice, bIsSlice := b.([]interface{})
	if aIsSlice && bIsSlice && (len(aSlice) == 0 && len(bSlice) == 0) {
		return true
	}

	// Array to nil comparison
	if aIsSlice && b == nil && len(aSlice) == 0 {
		return true
	}
	if bIsSlice && a == nil && len(bSlice) == 0 {
		return true
	}

	return false
}

// wrapText wraps text at the specified width while preserving ANSI color codes
// and maintaining proper indentation for wrapped lines.
func wrapText(text string, width int, indentStr string) string {
	if width <= 0 {
		return text // No wrapping needed
	}

	// Find all ANSI color codes in the text
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	ansiMatches := ansiRegex.FindAllStringIndex(text, -1)

	// If there are no ANSI codes or the text is shorter than width, return as is
	if len(ansiMatches) == 0 && len(text) <= width {
		return text
	}

	var result strings.Builder
	var currentLine strings.Builder
	var activeColorCode string
	var visualLength int

	// Process the text character by character
	for i := 0; i < len(text); i++ {
		// Check if current position is the start of an ANSI code
		isAnsiStart := false
		for _, match := range ansiMatches {
			if i == match[0] {
				// Extract the ANSI code
				ansiCode := text[match[0]:match[1]]
				currentLine.WriteString(ansiCode)

				// Store the active color code if it's a color code
				if !strings.Contains(ansiCode, "[0m") { // If not a reset code
					activeColorCode = ansiCode
				} else {
					activeColorCode = ""
				}

				// Skip to the end of the ANSI code
				i = match[1] - 1
				isAnsiStart = true
				break
			}
		}

		if isAnsiStart {
			continue
		}

		// Add the current character
		currentLine.WriteByte(text[i])
		visualLength++

		// Check if we need to wrap
		if visualLength >= width && i < len(text)-1 {
			// Write the current line to the result
			result.WriteString(currentLine.String())
			result.WriteString("\n")

			// Start a new line with proper indentation and active color code
			currentLine.Reset()
			currentLine.WriteString(indentStr + "  ") // Additional indentation for wrapped lines
			if activeColorCode != "" {
				currentLine.WriteString(activeColorCode)
			}

			visualLength = 0
		}
	}

	// Add the last line if there's anything left
	if currentLine.Len() > 0 {
		result.WriteString(currentLine.String())
	}

	return result.String()
}

// parseTerraformPlanJSON parses the terraform show json output and builds a tree of nodes.
func parseTerraformPlanJSON(planJSON string) []*TreeNode {
	// Implementation - this function parses JSON format plans
	var plan map[string]interface{}
	err := json.Unmarshal([]byte(planJSON), &plan)
	if err != nil {
		// If we can't parse JSON, return an error node
		errorNode := &TreeNode{
			Text:       "Error parsing plan JSON: " + err.Error(),
			Expanded:   true,
			Type:       "error",
			Depth:      0,
			Toggleable: false,
		}
		return []*TreeNode{errorNode}
	}

	// Root collection of nodes
	var rootNodes []*TreeNode

	// Resource type counters
	createCount := 0
	updateCount := 0
	replaceCount := 0
	destroyCount := 0
	moveCount := 0
	driftCount := 0

	// First, check for resource drift and add them directly as root nodes
	if resourceDrift, ok := plan["resource_drift"].([]interface{}); ok && len(resourceDrift) > 0 {
		// Create a map to store drifted resources by their path
		driftedResources := make(map[string]*TreeNode)

		// Process each drifted resource
		for _, driftItem := range resourceDrift {
			driftMap, ok := driftItem.(map[string]interface{})
			if !ok {
				continue
			}

			driftCount++
			address, _ := driftMap["address"].(string)
			typeStr, _ := driftMap["type"].(string)
			change, ok := driftMap["change"].(map[string]interface{})
			if !ok {
				continue
			}

			actions, _ := change["actions"].([]interface{})

			// Convert actions to strings
			actionStrs := make([]string, 0, len(actions))
			for _, a := range actions {
				if aStr, ok := a.(string); ok {
					actionStrs = append(actionStrs, aStr)
				}
			}

			// Create resource node as a root node
			changeType := mapActionsToChangeType(actionStrs)
			resourceNode := &TreeNode{
				Text:       fmt.Sprintf("# %s has drifted (%s)", address, changeType),
				Expanded:   false, // Start collapsed
				Type:       "resource",
				Depth:      0, // As a root node
				Toggleable: true,
				ChangeType: changeType,
				IsDrifted:  true,
			}

			// Create a node for the resource block itself
			resourceBlockNode := &TreeNode{
				Text:       formatResourceDeclaration(address, typeStr, changeType),
				Expanded:   false, // Start collapsed
				Type:       "block",
				Depth:      1, // One level deeper
				Parent:     resourceNode,
				Toggleable: true,
				ChangeType: changeType,
				IsDrifted:  true,
			}
			resourceNode.Children = append(resourceNode.Children, resourceBlockNode)

			// Add before/after details if available as children of the resource block
			addResourceDiffNodes(resourceBlockNode, change)

			// Add closing brace
			closingBraceNode := &TreeNode{
				Text:       "}",
				Expanded:   false, // Consistent with block node
				Type:       "closing_brace",
				Depth:      1,
				Parent:     resourceNode,
				Toggleable: false,
			}
			resourceNode.Children = append(resourceNode.Children, closingBraceNode)

			// Store in map
			driftedResources[address] = resourceNode
		}

		// Sort and add drifted resources
		var driftedPaths []string
		for path := range driftedResources {
			driftedPaths = append(driftedPaths, path)
		}
		sort.Strings(driftedPaths)
		for _, path := range driftedPaths {
			rootNodes = append(rootNodes, driftedResources[path])
		}

		// Add a separator after drifted resources
		rootNodes = append(rootNodes, &TreeNode{
			Text:       "",
			Expanded:   true,
			Type:       "separator",
			Depth:      0,
			Toggleable: false,
		})
	}

	// Process resource changes
	resourceChanges, ok := plan["resource_changes"].([]interface{})
	if !ok {
		// No changes found
		noChangesNode := &TreeNode{
			Text:       "No changes. Infrastructure is up-to-date.",
			Expanded:   true,
			Type:       "info",
			Depth:      0,
			Toggleable: false,
		}
		rootNodes = append(rootNodes, noChangesNode)
		return rootNodes
	}

	// Create a map to store resources by their path
	type resourceInfo struct {
		node       *TreeNode
		path       string
		changeType string
		wasMoved   bool
	}

	// Store all resources in a map
	resources := make(map[string]resourceInfo)

	// Process each resource change
	for _, change := range resourceChanges {
		changeMap, ok := change.(map[string]interface{})
		if !ok {
			continue
		}

		address, _ := changeMap["address"].(string)
		previousAddress, hasPrevious := changeMap["previous_address"].(string)
		actionReason, _ := changeMap["action_reason"].(string)
		typeStr, _ := changeMap["type"].(string)
		mode, _ := changeMap["mode"].(string)

		changeDetails, ok := changeMap["change"].(map[string]interface{})
		if !ok {
			continue
		}

		actions, _ := changeDetails["actions"].([]interface{})

		// Skip no-ops
		if len(actions) == 0 {
			continue
		}

		// Convert actions to strings
		actionStrs := make([]string, 0, len(actions))
		for _, a := range actions {
			if aStr, ok := a.(string); ok {
				actionStrs = append(actionStrs, aStr)
			}
		}

		// Skip no-ops
		if len(actionStrs) == 1 && actionStrs[0] == "no-op" {
			continue
		}

		// Determine the change type
		changeType := mapActionsToChangeType(actionStrs)

		// Create resource node with appropriate text
		var resourceText string
		wasMoved := hasPrevious && previousAddress != address && previousAddress != ""

		if wasMoved {
			resourceText = fmt.Sprintf("# %s will be %s (moved from %s)", address, getGrammaticalAction(changeType), previousAddress)
			moveCount++
		} else if actionReason != "" {
			reasonText := getActionReasonDisplay(actionReason)
			resourceText = fmt.Sprintf("# %s will be %s (%s)", address, getGrammaticalAction(changeType), reasonText)
		} else {
			resourceText = fmt.Sprintf("# %s will be %s", address, getGrammaticalAction(changeType))
		}

		// Create the resource node (as a root node)
		resourceNode := &TreeNode{
			Text:            resourceText,
			Expanded:        false, // Start collapsed
			Type:            "resource",
			Depth:           0, // As a root node
			Toggleable:      true,
			ChangeType:      changeType,
			PreviousAddress: previousAddress,
			ActionReason:    actionReason,
		}

		// Create a node for the resource block itself with the appropriate formatting based on the action
		var resourceBlockPrefix string
		switch changeType {
		case "create":
			resourceBlockPrefix = "+"
			createCount++
		case "destroy":
			resourceBlockPrefix = "-"
			destroyCount++
		case "update":
			resourceBlockPrefix = "~"
			updateCount++
		case "replace":
			resourceBlockPrefix = "-/+"
			replaceCount++
			// A replace counts as both a create and a destroy
			createCount++
			destroyCount++
		default:
			resourceBlockPrefix = " "
		}

		// Format resource declaration (e.g., "+ resource "aws_instance" "example" {")
		resourceBlockDeclaration := fmt.Sprintf("%s resource \"%s\" \"%s\" {",
			resourceBlockPrefix,
			typeStr,
			getResourceNameFromAddress(address, mode, typeStr))

		resourceBlockNode := &TreeNode{
			Text:            resourceBlockDeclaration,
			Expanded:        false, // Start collapsed
			Type:            "block",
			Depth:           1, // One level deeper
			Parent:          resourceNode,
			Toggleable:      true,
			ChangeType:      changeType,
			PreviousAddress: previousAddress,
			ActionReason:    actionReason,
		}
		resourceNode.Children = append(resourceNode.Children, resourceBlockNode)

		// Add details as children of the resource block node
		addResourceDiffNodes(resourceBlockNode, changeDetails, changeType)

		// Add a closing brace node to the resource block
		closingBraceNode := &TreeNode{
			Text:       "}",
			Expanded:   false, // Consistent with block node
			Type:       "closing_brace",
			Depth:      1,
			Parent:     resourceNode,
			Toggleable: false,
		}
		resourceNode.Children = append(resourceNode.Children, closingBraceNode)

		// Store the resource in our map
		resources[address] = resourceInfo{
			node:       resourceNode,
			path:       address,
			changeType: changeType,
			wasMoved:   wasMoved,
		}
	}

	// Sort resources by path
	var sortedPaths []string
	for path := range resources {
		sortedPaths = append(sortedPaths, path)
	}
	sort.Strings(sortedPaths)

	// Add sorted resources to root nodes
	for _, path := range sortedPaths {
		info := resources[path]
		if !info.wasMoved {
			rootNodes = append(rootNodes, info.node)
		}
	}

	// Add moved resources at the end
	for _, path := range sortedPaths {
		info := resources[path]
		if info.wasMoved {
			rootNodes = append(rootNodes, info.node)
		}
	}

	// Add summary
	summaryText := fmt.Sprintf("Plan: %d to add, %d to change, %d to destroy", createCount, updateCount, destroyCount)
	if moveCount > 0 {
		summaryText += fmt.Sprintf(" (%d moved)", moveCount)
	}
	if driftCount > 0 {
		summaryText += fmt.Sprintf(" (%d drifted)", driftCount)
	}

	rootNodes = append(rootNodes, &TreeNode{
		Text:       summaryText,
		Expanded:   true,
		Type:       "summary",
		Depth:      0,
		Toggleable: false,
	})

	return rootNodes
}

// Extract the resource name from an address (e.g., "module.foo.aws_instance.bar[0]" -> "bar")
func getResourceNameFromAddress(address, mode, resourceType string) string {
	// Handle module resources
	parts := strings.Split(address, ".")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		// Handle indexed resources like "aws_instance.bar[0]"
		indexBracket := strings.IndexByte(lastPart, '[')
		if indexBracket > 0 {
			lastPart = lastPart[:indexBracket]
		}
		return lastPart
	}
	return address
}

// Helper function to format a resource declaration
func formatResourceDeclaration(address, resourceType, changeType string) string {
	switch changeType {
	case "create":
		return fmt.Sprintf("+ resource \"%s\" {", resourceType)
	case "destroy":
		return fmt.Sprintf("- resource \"%s\" {", resourceType)
	case "update":
		return fmt.Sprintf("~ resource \"%s\" {", resourceType)
	case "replace":
		return fmt.Sprintf("-/+ resource \"%s\" {", resourceType)
	default:
		return fmt.Sprintf("  resource \"%s\" {", resourceType)
	}
}

// Helper function to add resource attribute and diff nodes based on the change type
func addResourceDiffNodes(parent *TreeNode, change map[string]interface{}, changeType ...string) {
	before, hasBefore := change["before"]
	after, hasAfter := change["after"]

	// Determine the change type - either from parameter or parent node
	var effectiveChangeType string
	if len(changeType) > 0 {
		effectiveChangeType = changeType[0]
	} else if parent.ChangeType != "" {
		effectiveChangeType = parent.ChangeType
	} else {
		// Default handling if no change type is provided
		if hasBefore && hasAfter {
			effectiveChangeType = "update"
		} else if hasAfter {
			effectiveChangeType = "create"
		} else if hasBefore {
			effectiveChangeType = "destroy"
		}
	}

	// Process attributes based on change type
	if effectiveChangeType == "create" && hasAfter {
		// For creates, only show after values
		addResourceAttributes(parent, after, "+", parent.Depth+1)
	} else if effectiveChangeType == "destroy" && hasBefore {
		// For destroys, only show before values
		addResourceAttributes(parent, before, "-", parent.Depth+1)
	} else if (effectiveChangeType == "update" || effectiveChangeType == "replace") && hasBefore && hasAfter {
		// For updates/replaces, compare before and after
		processAttributeDiffs(parent, before, after, parent.Depth+1)
	}
}

// Helper to map action strings to change type
func mapActionsToChangeType(actions []string) string {
	if sliceContains(actions, "create") && sliceContains(actions, "delete") {
		return "replace"
	} else if sliceContains(actions, "create") {
		return "create"
	} else if sliceContains(actions, "delete") {
		return "destroy"
	} else if sliceContains(actions, "update") {
		return "update"
	} else if len(actions) == 0 {
		return "no-op"
	}
	return strings.Join(actions, "/")
}

// Helper to translate action reason to display string
func getActionReasonDisplay(reason string) string {
	switch reason {
	case "replace_because_tainted":
		return "tainted, so must be replaced"
	case "replace_because_cannot_update":
		return "cannot be updated in-place"
	case "replace_by_request":
		return "replacement requested"
	case "delete_because_no_resource_config":
		return "no resource configuration found"
	case "delete_because_no_module":
		return "containing module is gone"
	case "delete_because_wrong_repetition":
		return "wrong repetition mode"
	case "delete_because_count_index":
		return "count index out of range"
	case "delete_because_each_key":
		return "for_each key not found"
	case "read_because_config_unknown":
		return "configuration contains unknown values"
	case "read_because_dependency_pending":
		return "has pending dependent resources"
	default:
		return reason
	}
}

// Helper to check if a slice contains a string
func sliceContains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// Helper to get grammatically correct action text
func getGrammaticalAction(action string) string {
	switch action {
	case "create":
		return "created"
	case "update":
		return "updated"
	case "replace":
		return "replaced"
	case "destroy":
		return "destroyed"
	case "move":
		return "moved"
	default:
		return action + "d" // Add 'd' as a general case
	}
}

// Helper function to process attribute differences
func processAttributeDiffs(parent *TreeNode, before, after interface{}, depth int) {
	if before == nil && after == nil {
		return
	}

	// Check if both before and after are maps
	beforeMap, beforeIsMap := before.(map[string]interface{})
	afterMap, afterIsMap := after.(map[string]interface{})

	if !beforeIsMap || !afterIsMap {
		// Handle non-map types with a simple comparison
		if !reflect.DeepEqual(before, after) {
			beforeStr := formatAttributeValue(before)
			afterStr := formatAttributeValue(after)

			node := &TreeNode{
				Text:       fmt.Sprintf("~ value = %s -> %s", beforeStr, afterStr),
				Expanded:   false,
				Type:       "attribute",
				Depth:      depth,
				Parent:     parent,
				Toggleable: false,
				ChangeType: "update",
			}
			parent.Children = append(parent.Children, node)
		}
		return
	}

	// Collect all keys from both before and after
	allKeys := make(map[string]bool)
	for k := range beforeMap {
		allKeys[k] = true
	}
	for k := range afterMap {
		allKeys[k] = true
	}

	// Sort the keys for consistent display
	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Process each key
	for _, key := range keys {
		beforeVal, beforeExists := beforeMap[key]
		afterVal, afterExists := afterMap[key]

		// Handle added attributes
		if !beforeExists && afterExists {
			addResourceAttributes(parent, map[string]interface{}{key: afterVal}, "+", depth)
			continue
		}

		// Handle removed attributes
		if beforeExists && !afterExists {
			addResourceAttributes(parent, map[string]interface{}{key: beforeVal}, "-", depth)
			continue
		}

		// Handle changed attributes
		if !reflect.DeepEqual(beforeVal, afterVal) {
			beforeMapValue, beforeIsMap := beforeVal.(map[string]interface{})
			afterMapValue, afterIsMap := afterVal.(map[string]interface{})

			// Handle nested blocks
			if beforeIsMap && afterIsMap {
				// Create block node
				blockNode := &TreeNode{
					Text:       fmt.Sprintf("~ %s {", key),
					Expanded:   true, // Expand by default to show changes
					Type:       "block",
					Depth:      depth,
					Parent:     parent,
					Toggleable: true,
					ChangeType: "update",
				}
				parent.Children = append(parent.Children, blockNode)

				// Recursively compare nested blocks
				processAttributeDiffs(blockNode, beforeMapValue, afterMapValue, depth+1)

				// Add closing brace
				closingBrace := &TreeNode{
					Text:       "}",
					Expanded:   false,
					Type:       "closing_brace",
					Depth:      depth,
					Parent:     parent,
					Toggleable: false,
				}
				parent.Children = append(parent.Children, closingBrace)
			} else if beforeSlice, beforeIsSlice := beforeVal.([]interface{}); beforeIsSlice {
				if afterSlice, afterIsSlice := afterVal.([]interface{}); afterIsSlice {
					// Handle array changes
					blockNode := &TreeNode{
						Text:       fmt.Sprintf("~ %s {", key),
						Expanded:   true, // Expand by default to show changes
						Type:       "block",
						Depth:      depth,
						Parent:     parent,
						Toggleable: true,
						ChangeType: "update",
					}
					parent.Children = append(parent.Children, blockNode)

					// Process each item in the array
					maxLen := len(beforeSlice)
					if len(afterSlice) > maxLen {
						maxLen = len(afterSlice)
					}

					for i := 0; i < maxLen; i++ {
						var beforeItem, afterItem interface{}
						if i < len(beforeSlice) {
							beforeItem = beforeSlice[i]
						}
						if i < len(afterSlice) {
							afterItem = afterSlice[i]
						}

						if !reflect.DeepEqual(beforeItem, afterItem) {
							if beforeItemMap, ok := beforeItem.(map[string]interface{}); ok {
								if afterItemMap, ok := afterItem.(map[string]interface{}); ok {
									// Create a node for this array item
									itemNode := &TreeNode{
										Text:       fmt.Sprintf("~ [%d] {", i),
										Expanded:   true,
										Type:       "block",
										Depth:      depth + 1,
										Parent:     blockNode,
										Toggleable: true,
										ChangeType: "update",
									}
									blockNode.Children = append(blockNode.Children, itemNode)

									// Process the item's attributes
									processAttributeDiffs(itemNode, beforeItemMap, afterItemMap, depth+2)

									// Add closing brace for the item
									itemClosingBrace := &TreeNode{
										Text:       "}",
										Expanded:   false,
										Type:       "closing_brace",
										Depth:      depth + 1,
										Parent:     blockNode,
										Toggleable: false,
									}
									blockNode.Children = append(blockNode.Children, itemClosingBrace)
								}
							} else {
								// Simple value in array
								beforeStr := formatAttributeValue(beforeItem)
								afterStr := formatAttributeValue(afterItem)
								node := &TreeNode{
									Text:       fmt.Sprintf("~ [%d] = %s -> %s", i, beforeStr, afterStr),
									Expanded:   false,
									Type:       "attribute",
									Depth:      depth + 1,
									Parent:     blockNode,
									Toggleable: false,
									ChangeType: "update",
								}
								blockNode.Children = append(blockNode.Children, node)
							}
						}
					}

					// Add closing brace for the array block
					closingBrace := &TreeNode{
						Text:       "}",
						Expanded:   false,
						Type:       "closing_brace",
						Depth:      depth,
						Parent:     parent,
						Toggleable: false,
					}
					parent.Children = append(parent.Children, closingBrace)
				}
			} else {
				// Handle simple value changes
				beforeStr := formatAttributeValue(beforeVal)
				afterStr := formatAttributeValue(afterVal)

				if afterVal == "(known after apply)" || afterStr == "(known after apply)" {
					node := &TreeNode{
						Text:       fmt.Sprintf("~ %s = %s -> (known after apply)", key, beforeStr),
						Expanded:   false,
						Type:       "attribute",
						Depth:      depth,
						Parent:     parent,
						Toggleable: false,
						ChangeType: "update",
					}
					parent.Children = append(parent.Children, node)
				} else {
					node := &TreeNode{
						Text:       fmt.Sprintf("~ %s = %s -> %s", key, beforeStr, afterStr),
						Expanded:   false,
						Type:       "attribute",
						Depth:      depth,
						Parent:     parent,
						Toggleable: false,
						ChangeType: "update",
					}
					parent.Children = append(parent.Children, node)
				}
			}
		} else {
			// Unchanged attribute - could add with a comment about being unchanged
			// For now, we'll skip to reduce clutter

			// Handle complex unchanged values like blocks
			if _, isMap := beforeVal.(map[string]interface{}); isMap {
				// Add a collapsed node for the unchanged block
				node := &TreeNode{
					Text:       fmt.Sprintf("  %s {", key),
					Expanded:   false, // Keep collapsed by default since it's unchanged
					Type:       "block",
					Depth:      depth,
					Parent:     parent,
					Toggleable: true,
				}
				parent.Children = append(parent.Children, node)

				// Add a hint about unchanged content
				hint := &TreeNode{
					Text:       "# (unchanged block hidden)",
					Expanded:   false,
					Type:       "comment",
					Depth:      depth + 1,
					Parent:     node,
					Toggleable: false,
				}
				node.Children = append(node.Children, hint)

				// Add closing brace
				closingBrace := &TreeNode{
					Text:       "}",
					Expanded:   false, // Consistent with block node
					Type:       "closing_brace",
					Depth:      depth,
					Parent:     parent,
					Toggleable: false,
				}
				parent.Children = append(parent.Children, closingBrace)
			}
		}
	}

	// Add a hint about hidden unchanged attributes if there are many
	var unusedCount int = 0
	for _, key := range keys {
		beforeVal, beforeExists := beforeMap[key]
		afterVal, afterExists := afterMap[key]

		if beforeExists && afterExists && reflect.DeepEqual(beforeVal, afterVal) {
			unusedCount++
		}
	}

	if unusedCount > 3 {
		// Add a comment about hidden attributes
		comment := &TreeNode{
			Text:       fmt.Sprintf("# (%d unchanged attributes hidden)", unusedCount),
			Expanded:   false,
			Type:       "comment",
			Depth:      depth,
			Parent:     parent,
			Toggleable: false,
		}
		parent.Children = append(parent.Children, comment)
	}
}

// Format an attribute value for display
func formatAttributeValue(value interface{}) string {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case map[string]interface{}:
		return "{...}" // Simplified representation for maps
	case []interface{}:
		return "[...]" // Simplified representation for arrays
	default:
		return fmt.Sprintf("%v", value)
	}
}

// Helper function to add resource attributes as child nodes
func addResourceAttributes(parent *TreeNode, attributes interface{}, prefix string, depth int) {
	if attributes == nil {
		return
	}

	attrMap, ok := attributes.(map[string]interface{})
	if !ok {
		return
	}

	// Sort the attribute keys for consistent display
	keys := make([]string, 0, len(attrMap))
	for k := range attrMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Add each attribute as a node
	for _, key := range keys {
		value := attrMap[key]

		// Handle different value types
		if value == nil {
			// Nil value
			node := &TreeNode{
				Text:       fmt.Sprintf("%s %s = null", prefix, key),
				Expanded:   false,
				Type:       "attribute",
				Depth:      depth,
				Parent:     parent,
				Toggleable: false,
			}
			parent.Children = append(parent.Children, node)
		} else if mapValue, isMap := value.(map[string]interface{}); isMap {
			// Nested block
			blockNode := &TreeNode{
				Text:       fmt.Sprintf("%s %s {", prefix, key),
				Expanded:   true,
				Type:       "block",
				Depth:      depth,
				Parent:     parent,
				Toggleable: true,
			}
			parent.Children = append(parent.Children, blockNode)

			// Add nested attributes
			addResourceAttributes(blockNode, mapValue, prefix, depth+1)

			// Add closing brace
			closingBrace := &TreeNode{
				Text:       "}",
				Expanded:   false, // Consistent with block node
				Type:       "closing_brace",
				Depth:      depth,
				Parent:     parent,
				Toggleable: false,
			}
			parent.Children = append(parent.Children, closingBrace)
		} else if arrValue, isArr := value.([]interface{}); isArr {
			// Array value
			for i, item := range arrValue {
				if mapItem, isMapItem := item.(map[string]interface{}); isMapItem {
					// Nested block in array
					blockNode := &TreeNode{
						Text:       fmt.Sprintf("%s %s {", prefix, key),
						Expanded:   true,
						Type:       "block",
						Depth:      depth,
						Parent:     parent,
						Toggleable: true,
					}
					parent.Children = append(parent.Children, blockNode)

					// Add nested attributes
					addResourceAttributes(blockNode, mapItem, prefix, depth+1)

					// Add closing brace
					closingBrace := &TreeNode{
						Text:       "}",
						Expanded:   false, // Consistent with block node
						Type:       "closing_brace",
						Depth:      depth,
						Parent:     parent,
						Toggleable: false,
					}
					parent.Children = append(parent.Children, closingBrace)
				} else {
					// Simple array item
					node := &TreeNode{
						Text:       fmt.Sprintf("%s %s[%d] = %v", prefix, key, i, item),
						Expanded:   false,
						Type:       "attribute",
						Depth:      depth,
						Parent:     parent,
						Toggleable: false,
					}
					parent.Children = append(parent.Children, node)
				}
			}
		} else {
			// Simple value
			valueStr := fmt.Sprintf("%v", value)
			if strValue, isStr := value.(string); isStr {
				valueStr = fmt.Sprintf("\"%s\"", strValue)
			}

			node := &TreeNode{
				Text:       fmt.Sprintf("%s %s = %s", prefix, key, valueStr),
				Expanded:   false,
				Type:       "attribute",
				Depth:      depth,
				Parent:     parent,
				Toggleable: false,
			}
			parent.Children = append(parent.Children, node)
		}
	}
}

// Helper function to check if a node is a resource node that should be navigated to
func isRootResource(node *TreeNode) bool {
	// With the new structure, resource nodes are directly at the root level with depth 0
	return node.Type == "resource"
}
