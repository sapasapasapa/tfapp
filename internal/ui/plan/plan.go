// Package plan provides an interactive Terraform plan viewer with collapsible sections.
package plan

import (
	"fmt"
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
}

// Model represents the state of the plan viewer.
type Model struct {
	nodes        []*TreeNode // All root-level nodes
	allNodes     []*TreeNode // All nodes (flattened)
	cursor       int         // Current cursor position
	windowTop    int         // The top line of the window being displayed
	windowHeight int         // Height of visible window
	quitting     bool        // Whether the user is quitting
	ready        bool        // Whether we've received the window size yet
	showHelp     bool        // Whether to show the help tooltip
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
		nodes:        nodes,
		allNodes:     allNodes,
		cursor:       0,
		windowTop:    0,
		windowHeight: 25, // Show approximately 25 lines at a time for better visibility
		quitting:     false,
		ready:        false,
		showHelp:     false,
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
				// Adjust window if needed
				if m.cursor < m.windowTop {
					m.windowTop = m.cursor
				}
			}

		case "down", "j":
			// Get visible nodes and check if we can move down
			visibleNodes := getVisibleNodes(m.nodes)
			if m.cursor < len(visibleNodes)-1 {
				m.cursor++
				// Adjust window if needed
				if m.cursor >= m.windowTop+m.windowHeight {
					m.windowTop = m.cursor - m.windowHeight + 1
				}
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

		case "n":
			// Collapse all nodes with children
			for _, node := range m.allNodes {
				if len(node.Children) > 0 && !node.IsRoot {
					node.Expanded = false
				}
			}

			// Refresh the list of all nodes
			m.allNodes = flattenNodes(m.nodes)

		case "home", "g":
			// Jump to the top of the plan
			m.cursor = 0
			m.windowTop = 0

		case "end", "G":
			// Jump to the bottom of the plan
			visibleNodes := getVisibleNodes(m.nodes)
			if len(visibleNodes) > 0 {
				m.cursor = len(visibleNodes) - 1
				// Adjust window if needed
				if m.cursor >= m.windowTop+m.windowHeight {
					m.windowTop = m.cursor - m.windowHeight + 1
					if m.windowTop < 0 {
						m.windowTop = 0
					}
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
				expandChar = ui.ColorInfo + "▼ " + ui.ColorReset
			} else {
				expandChar = ui.ColorHighlight + "▶ " + ui.ColorReset
			}
		}

		// Style the line based on node type
		line := indent + expandChar + node.Text

		// Apply custom colorization based on node type
		var colorized string
		if node.Type == "resource" {
			// Resources are already colorized by the ui.Colorize function
			colorized = ui.Colorize(line)
		} else if node.Type == "block" {
			// Add custom color for blocks (e.g., rule {})
			if strings.Contains(line, "{") {
				parts := strings.SplitN(line, "{", 2)
				colorized = ui.ColorWarning + parts[0] + ui.ColorReset + "{"
				if len(parts) > 1 {
					colorized += parts[1]
				}
			} else {
				colorized = ui.Colorize(line)
			}
		} else {
			// Handle attributes (+ and - changes)
			if strings.HasPrefix(strings.TrimSpace(line), "+") {
				colorized = strings.Replace(line, "+", ui.ColorSuccess+"+"+ui.ColorReset, 1)
			} else if strings.HasPrefix(strings.TrimSpace(line), "-") {
				colorized = strings.Replace(line, "-", ui.ColorError+"-"+ui.ColorReset, 1)
			} else if strings.HasPrefix(strings.TrimSpace(line), "~") {
				colorized = strings.Replace(line, "~", ui.ColorWarning+"~"+ui.ColorReset, 1)
			} else {
				colorized = ui.Colorize(line)
			}
		}

		// Highlight the current line
		if i == m.cursor {
			colorized = lipgloss.NewStyle().
				Background(lipgloss.Color("#333333")).
				Render(colorized)
		}

		// Write the line to output
		sb.WriteString(cursor + colorized + "\n")
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
		Foreground(lipgloss.Color("#CCCCCC")).
		Background(lipgloss.Color(ui.GetHexColorByName("highlight"))).
		Bold(true).
		Width(100).
		Padding(0, 1)

	// Create the status message with navigation info
	var statusMsg string
	if totalNodes <= contentHeight {
		// Everything fits on screen
		statusMsg = fmt.Sprintf("All %d items visible - Press ? for help", totalNodes)
	} else {
		// Show percentage and position
		statusMsg = fmt.Sprintf("Line %d of %d (%d%%) - Press ? for help",
			m.cursor+1, totalNodes, percentage)

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
func parsePlan(planOutput string) []*TreeNode {
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

// getVisibleNodes returns only the nodes that should be visible based on expansion state.
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

// isNodeVisible determines if a node should be visible based on its parents' expansion state.
func isNodeVisible(node *TreeNode) bool {
	if node.IsRoot || node.Parent == nil {
		return true
	}

	// Check if all ancestors are expanded
	parent := node.Parent
	for parent != nil && !parent.IsRoot {
		if !parent.Expanded {
			return false
		}
		parent = parent.Parent
	}

	return true
}

// getVisibleNodeAt returns the node at the given index in the visible nodes list.
func getVisibleNodeAt(allNodes []*TreeNode, index int) *TreeNode {
	visibleNodes := getVisibleNodes(allNodes)
	if index >= 0 && index < len(visibleNodes) {
		return visibleNodes[index]
	}
	return nil
}

// countVisibleNodes counts the number of visible nodes.
func countVisibleNodes(allNodes []*TreeNode) int {
	return len(getVisibleNodes(allNodes))
}

// findFirstNodeWithChildren returns the first node in the list that has children
func findFirstNodeWithChildren(nodes []*TreeNode) *TreeNode {
	for _, node := range nodes {
		if len(node.Children) > 0 {
			return node
		}
	}
	return nil
}

// ensureCursorVisible ensures the cursor is visible within the window
func ensureCursorVisible(m *Model) {
	visibleNodes := getVisibleNodes(m.nodes)

	// Make sure cursor is within visible nodes range
	if m.cursor >= len(visibleNodes) {
		m.cursor = len(visibleNodes) - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
	}

	// Check if cursor is outside visible window
	if m.cursor < m.windowTop {
		// Cursor is above the window, adjust windowTop to show cursor
		m.windowTop = m.cursor
	} else if m.cursor >= m.windowTop+m.windowHeight {
		// Cursor is below the window, adjust windowTop to show cursor
		m.windowTop = m.cursor - m.windowHeight + 1
	}

	// Ensure windowTop is not negative
	if m.windowTop < 0 {
		m.windowTop = 0
	}

	// Ensure windowTop isn't too large
	maxTop := len(visibleNodes) - m.windowHeight
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
		{"→/l/Enter/Space", "Expand current node"},
		{"←/h", "Collapse current node or jump to parent"},
		{"a", "Expand all nodes"},
		{"n", "Collapse all nodes except root level"},
		{"Home/g", "Jump to the top"},
		{"End/G", "Jump to the bottom"},
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
