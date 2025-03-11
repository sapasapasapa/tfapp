// Package plan provides an interactive Terraform plan viewer with collapsible sections.
package plan

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"tfapp/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TreeNode represents a node in the plan's resource tree.
type TreeNode struct {
	Text       string      // The text content of this node
	Children   []*TreeNode // Child nodes (nested blocks)
	Parent     *TreeNode   // Parent node (nil for root)
	Depth      int         // Depth in the tree
	Expanded   bool        // Whether this node is expanded
	Type       string      // Type of node (resource, block, attribute)
	IsRoot     bool        // Whether this is a root node
	Toggleable bool        // Whether this node can be expanded/collapsed
	ChangeType string      // Type of change (create, update, delete, replace)
}

// Model represents the state of the plan viewer.
type Model struct {
	nodes            []*TreeNode // All root-level nodes
	allNodes         []*TreeNode // All nodes (flattened)
	cursor           int         // Current cursor position
	windowTop        int         // The top line of the window being displayed
	windowHeight     int         // Height of visible window
	terminalWidth    int         // Width of the terminal window for text wrapping
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

	// Set all resource nodes to expanded by default
	for _, node := range nodes {
		if node.Type == "resource" {
			node.Expanded = true
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
		terminalWidth:    80, // Default terminal width
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
		m.terminalWidth = msg.Width

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

			case "right", "l", " ":
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

			case "left", "h":
				// Collapse current node or move to parent
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
						if visibleNodes[i].Type == "resource" && visibleNodes[i].Depth == 0 {
							m.cursor = i
							found = true
							break
						}
					}

					// If not found and we started after position 0, search from beginning to cursor
					if !found && startPos > 0 {
						for i := 0; i < startPos; i++ {
							if visibleNodes[i].Type == "resource" && visibleNodes[i].Depth == 0 {
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
						if visibleNodes[i].Type == "resource" && visibleNodes[i].Depth == 0 {
							m.cursor = i
							found = true
							break
						}
					}

					// If not found and we started before the end, search from end to cursor
					if !found && startPos < len(visibleNodes)-1 {
						for i := len(visibleNodes) - 1; i > startPos; i-- {
							if visibleNodes[i].Type == "resource" && visibleNodes[i].Depth == 0 {
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
				ensureCursorVisible(&m)

			case "end", "G":
				// Jump to the bottom of the plan
				visibleNodes := getVisibleNodes(m.nodes)
				if len(visibleNodes) > 0 {
					// Set cursor directly to the last visible node
					m.cursor = len(visibleNodes) - 1
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

	// Calculate effective width for text wrapping (accounting for cursor and some buffer)
	effectiveWidth := m.terminalWidth - 2 // Account for cursor space
	if effectiveWidth < 20 {
		effectiveWidth = 20 // Minimum reasonable width
	}

	// Render visible nodes
	for i := start; i < contentEnd; i++ {
		node := visibleNodes[i]

		// Indent based on depth
		indent := strings.Repeat("  ", node.Depth)

		// Show cursor if this is the selected node
		cursor := "  "
		if i == m.cursor {
			cursor = ui.GetCursorChar() + " "
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
		if node.Type == "resource" {
			// Resources are already colorized by the ui.Colorize function
			colorized = ui.Colorize(line)
		} else {
			// Apply color based on the node's change type
			switch node.ChangeType {
			case "create":
				if strings.Contains(strings.TrimSpace(line), "+") {
					colorized = strings.Replace(line, "+", ui.ColorSuccess+"+"+ui.ColorForegroundReset, 1)
				} else if strings.HasPrefix(strings.TrimSpace(line), "}") {
					// Don't color closing braces
					colorized = line
				} else {
					colorized = ui.ColorSuccess + line + ui.ColorForegroundReset
				}
			case "delete":
				if strings.Contains(strings.TrimSpace(line), "-") {
					colorized = strings.Replace(line, "-", ui.ColorError+"-"+ui.ColorForegroundReset, 1)
				} else if strings.HasPrefix(strings.TrimSpace(line), "}") {
					// Don't color closing braces
					colorized = line
				} else {
					colorized = ui.ColorError + line + ui.ColorForegroundReset
				}
			case "update", "replace":
				if strings.Contains(strings.TrimSpace(line), "~") {
					colorized = strings.Replace(line, "~", ui.ColorWarning+"~"+ui.ColorForegroundReset, 1)
				} else if strings.Contains(strings.TrimSpace(line), "-/+") {
					colorized = strings.Replace(line, "-/+", ui.ColorError+"-"+ui.ColorForegroundReset+"/"+ui.ColorSuccess+"+"+ui.ColorForegroundReset, 1)
				} else if strings.HasPrefix(strings.TrimSpace(line), "}") {
					colorized = line
				} else {
					colorized = ui.ColorWarning + line + ui.ColorForegroundReset
				}
			default:
				// For comments (like "# (5 unchanged attributes hidden)")
				if strings.HasPrefix(strings.TrimSpace(line), "#") {
					colorized = ui.ColorInfo + line + ui.ColorForegroundReset
				} else if node.Type == "closing_brace" {
					// Never color closing braces
					colorized = line
				} else {
					colorized = ui.Colorize(line)
				}
			}
		}

		// Highlight the current line
		if i == m.cursor {
			colorized = lipgloss.NewStyle().
				Background(lipgloss.Color("#333333")).
				Render(colorized)
		}

		// Apply text wrapping to handle long lines
		wrappedText := wrapText(colorized, effectiveWidth, indent)

		// Write the line to output
		sb.WriteString(cursor + wrappedText + "\n")
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
		return parsePlanJSON(planOutput)
	}

	// Continue with the existing text parsing logic
	lines := strings.Split(planOutput, "\n")

	// Root node for the entire plan
	root := &TreeNode{
		Text:       "Terraform Plan",
		Expanded:   true,
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
				Expanded:   true, // Resources are expanded by default
				Type:       "resource",
				Depth:      indent / 2,
				Parent:     root,
				Toggleable: false, // Resource headers should not be toggleable
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

	// Check if cursor is outside visible window
	if m.cursor < m.windowTop {
		// Cursor is above the window, adjust windowTop to show cursor at top
		m.windowTop = m.cursor
	} else if m.cursor >= m.windowTop+effectiveWindowHeight {
		// Cursor is below the window, adjust windowTop to show cursor at bottom of window
		m.windowTop = m.cursor - effectiveWindowHeight + 1
	}

	// Ensure windowTop is not negative
	if m.windowTop < 0 {
		m.windowTop = 0
	}

	// Ensure windowTop isn't too large (which would leave empty space at bottom)
	maxTop := len(visibleNodes) - effectiveWindowHeight
	if maxTop < 0 {
		maxTop = 0
	}
	if m.windowTop > maxTop {
		m.windowTop = maxTop
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

	// Create help content with key bindings and descriptions
	keys := []struct {
		key  string
		desc string
	}{
		{"↑/k", "Move cursor up"},
		{"↓/j", "Move cursor down"},
		{"→/l/Space", "Expand current node"},
		{"Enter", "Expand current node and all its children"},
		{"←/h", "Collapse current node or jump to parent"},
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
	helpContent.WriteString("Navigation Commands\n\n")

	// Format each key binding with description
	for _, item := range keys {
		line := fmt.Sprintf("%s  %s\n",
			keyStyle.Render(item.key),
			descStyle.Render(item.desc))
		helpContent.WriteString(line)
	}

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
