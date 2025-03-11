package plan

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// parsePlanJSON parses Terraform plan in JSON format
func parsePlanJSON(jsonPlan string) []*TreeNode {
	// Root node for the entire plan
	root := &TreeNode{
		Text:       "Terraform Plan",
		Expanded:   true,
		IsRoot:     true,
		Toggleable: true,
	}

	// Parse the JSON
	var plan TerraformPlan
	err := json.Unmarshal([]byte(jsonPlan), &plan)
	if err != nil {
		// Return error as a node
		errorNode := &TreeNode{
			Text:       "Error parsing JSON: " + err.Error(),
			Expanded:   true,
			Type:       "error",
			Depth:      0,
			Parent:     root,
			Toggleable: false,
		}
		root.Children = append(root.Children, errorNode)
		return root.Children
	}

	// Process each resource change
	for _, change := range plan.ResourceChanges {
		if len(change.Change.Actions) == 0 || change.Change.Actions[0] == "no-op" {
			continue
		}

		// Create resource header node
		actionText := strings.Join(change.Change.Actions, ", ")
		var headerPrefix string
		var resourcePrefix string
		var changeType string

		// Determine the prefix and change type based on the actions
		isReplacement := false
		for _, action := range change.Change.Actions {
			if action == "delete" && (contains(change.Change.Actions, "create") || contains(change.Change.Actions, "read")) {
				isReplacement = true
				break
			}
		}

		if isReplacement {
			headerPrefix = "# "
			resourcePrefix = "-/+ "
			actionText = "will be replaced"
			changeType = "replace"
		} else {
			// Handle non-replacement actions
			switch change.Change.Actions[0] {
			case "create":
				headerPrefix = "# "
				resourcePrefix = "+ "
				actionText = "will be created"
				changeType = "create"
			case "update":
				headerPrefix = "# "
				resourcePrefix = "~ "
				actionText = "will be updated"
				changeType = "update"
			case "delete":
				headerPrefix = "# "
				resourcePrefix = "- "
				actionText = "will be destroyed"
				changeType = "delete"
			default:
				headerPrefix = "# "
				resourcePrefix = "? "
				changeType = "unknown"
			}
		}

		resourceHeader := headerPrefix + change.Address + " " + actionText

		resourceNode := &TreeNode{
			Text:       resourceHeader,
			Expanded:   true,
			Type:       "resource",
			Depth:      0,
			Parent:     root,
			Toggleable: false,
			ChangeType: changeType,
		}

		root.Children = append(root.Children, resourceNode)

		// Create the resource definition line
		resourceDefText := resourcePrefix + "resource \"" + change.Type + "\" \"" + change.Name + "\" {"
		resourceDefNode := &TreeNode{
			Text:       resourceDefText,
			Expanded:   false,
			Type:       "block",
			Depth:      0,
			Parent:     root,
			Toggleable: true,
			ChangeType: changeType,
		}

		root.Children = append(root.Children, resourceDefNode)

		// Convert Before to map if it exists, otherwise use empty map
		var beforeMap map[string]interface{}
		if before, ok := change.Change.Before.(map[string]interface{}); ok {
			beforeMap = before
		} else {
			beforeMap = make(map[string]interface{})
		}

		// Add attributes and nested blocks, passing both before and after
		processAttributes(resourceDefNode, beforeMap, change.Change.After, change.Change.AfterUnknown, 2, resourcePrefix)

		// Add closing brace for resource block
		closingNode := createClosingBrace(0, resourceDefNode)
		resourceDefNode.Children = append(resourceDefNode.Children, closingNode)
	}

	return root.Children
}

// Helper function to check if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// processAttributes recursively processes attributes and blocks
func processAttributes(parentNode *TreeNode, beforeAttrs map[string]interface{}, afterAttrs map[string]interface{}, unknownAttrs map[string]interface{}, depth int, prefix string) {
	// Sort keys for consistent output
	var keys []string

	// Collect all keys from both before and after
	keyMap := make(map[string]bool)

	for k := range beforeAttrs {
		keyMap[k] = true
	}

	for k := range afterAttrs {
		keyMap[k] = true
	}

	for k := range keyMap {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	// Track hidden attributes and blocks for summary
	var hiddenAttrCount int
	var hiddenBlockCount int
	var childNodes []*TreeNode // Temporary storage for child nodes

	for _, key := range keys {
		beforeValue, beforeExists := beforeAttrs[key]
		afterValue, afterExists := afterAttrs[key]
		unknown := false

		// Check if the value is known or will be known after apply
		if unknownAttrs != nil {
			if unknownVal, exists := unknownAttrs[key]; exists {
				if boolVal, ok := unknownVal.(bool); ok && boolVal {
					unknown = true
				}
			}
		}

		// Determine the change type for this attribute
		var attrPrefix string
		var changeType string

		if unknown {
			// Handle unknown values
			if !beforeExists {
				// New attribute with unknown value
				attrPrefix = "+ "
				changeType = "create"
			} else {
				// Existing attribute with unknown new value
				attrPrefix = "~ "
				changeType = "update"
			}
		} else if !beforeExists && afterExists {
			// New attribute
			attrPrefix = "+ "
			changeType = "create"
		} else if beforeExists && !afterExists {
			// Deleted attribute
			attrPrefix = "- "
			changeType = "delete"
		} else if beforeExists && afterExists && afterValue == nil && !isEffectivelyEqual(beforeValue, afterValue) {
			// Changed to null - mark as deleted, but only if not effectively equal (e.g., "" to null)
			attrPrefix = "- "
			changeType = "delete"
		} else if beforeExists && afterExists && !isEffectivelyEqual(beforeValue, afterValue) {
			// Changed attribute
			attrPrefix = "~ "
			changeType = "update"
		} else {
			// Unchanged
			attrPrefix = "  "
			changeType = "no-op"

			// Skip processing unchanged attributes unless parent is being created/deleted
			if parentNode.ChangeType != "create" && parentNode.ChangeType != "delete" && parentNode.ChangeType != "replace" {
				hiddenAttrCount++
				continue
			}
		}

		// Override prefix if parent node is being created or deleted
		if parentNode.ChangeType == "create" {
			attrPrefix = "+ "
			changeType = "create"
		} else if parentNode.ChangeType == "delete" {
			attrPrefix = "- "
			changeType = "delete"
		} else if parentNode.ChangeType == "replace" {
			if attrPrefix == "  " {
				// If attribute is unchanged in a replacement, show it normally
				attrPrefix = "  "

				// Skip processing unchanged attributes in replacements too
				hiddenAttrCount++
			}
		}

		// Use the value from the side that exists, preferring after
		var value interface{}
		if afterExists {
			value = afterValue
		} else {
			value = beforeValue
		}

		// Skip processing if both values are nil and not unknown
		if value == nil && !unknown && changeType == "no-op" {
			hiddenAttrCount++
			continue
		}

		switch v := value.(type) {
		case map[string]interface{}:
			// Get before/after maps for this nested object
			var beforeMap map[string]interface{}
			var afterMap map[string]interface{}

			if beforeVal, ok := beforeValue.(map[string]interface{}); ok {
				beforeMap = beforeVal
			} else {
				beforeMap = make(map[string]interface{})
			}

			if afterVal, ok := afterValue.(map[string]interface{}); ok {
				afterMap = afterVal
			} else {
				afterMap = make(map[string]interface{})
			}

			// For labels and specific map types, format as inline
			if key == "terraform_labels" || key == "effective_labels" || key == "labels" || key == "tags" {
				blockNode := &TreeNode{
					Text:       attrPrefix + key + " = {",
					Expanded:   false,
					Type:       "block",
					Depth:      depth,
					Parent:     parentNode,
					Toggleable: true,
					ChangeType: changeType,
				}

				childNodes = append(childNodes, blockNode)

				// Format map entries properly
				formatMapEntries(blockNode, v, depth+1, attrPrefix)

				// Add closing brace
				closingNode := createClosingBrace(depth, blockNode)
				childNodes = append(childNodes, closingNode)
			} else {
				// This is a nested block
				blockNode := &TreeNode{
					Text:       attrPrefix + key + " {",
					Expanded:   false,
					Type:       "block",
					Depth:      depth,
					Parent:     parentNode,
					Toggleable: true,
					ChangeType: changeType,
				}

				childNodes = append(childNodes, blockNode)

				// Process nested attributes
				var nestedUnknown map[string]interface{}
				if unknownAttrs != nil {
					if unknownBlock, exists := unknownAttrs[key].(map[string]interface{}); exists {
						nestedUnknown = unknownBlock
					}
				}

				// Track if this block has any changes
				tempParent := &TreeNode{
					ChangeType: changeType,
				}

				// Recursive call with before and after maps for this block
				processAttributes(tempParent, beforeMap, afterMap, nestedUnknown, depth+1, attrPrefix)

				// Check if the block has any real changes or just hidden attributes
				hasChanges := false
				hasHiddenMessage := false
				hasRealChanges := false

				for _, child := range tempParent.Children {
					if strings.HasPrefix(child.Text, "#") && strings.Contains(child.Text, "unchanged") {
						hasHiddenMessage = true
					} else if child.ChangeType != "no-op" || strings.HasPrefix(child.Text, "+") || strings.HasPrefix(child.Text, "-") || strings.HasPrefix(child.Text, "~") {
						hasChanges = true

						// Check if this change is a real change or just a formatting difference
						// We need to check if the text indicates an empty string to null conversion
						isEmptyStringToNull := false
						if strings.Contains(child.Text, " -> ") {
							if strings.Contains(child.Text, "\"\"") && strings.Contains(child.Text, "null") {
								// This is likely an empty string to null change
								parts := strings.Split(child.Text, " -> ")
								if len(parts) == 2 {
									beforePart := strings.TrimSpace(parts[0])
									afterPart := strings.TrimSpace(parts[1])

									// Check for "" -> null or null -> ""
									if (beforePart == "\"\"" && afterPart == "null") ||
										(beforePart == "null" && afterPart == "\"\"") {
										isEmptyStringToNull = true
									}
								}
							}
						}

						// Mark as real change if it's not an empty string to null conversion
						if !isEmptyStringToNull {
							hasRealChanges = true
						}
					}
				}

				// If the block has real changes or we're showing everything, add all children
				if hasRealChanges || parentNode.ChangeType == "create" || parentNode.ChangeType == "delete" || parentNode.ChangeType == "replace" {
					// Before adding, do one final check - if all children have been modified to be no-op,
					// then we should hide this block too
					allChildrenEffectivelyUnchanged := true
					for _, child := range tempParent.Children {
						if child.ChangeType != "no-op" && !strings.HasPrefix(strings.TrimSpace(child.Text), "#") {
							allChildrenEffectivelyUnchanged = false
							break
						}
					}

					if allChildrenEffectivelyUnchanged && len(tempParent.Children) > 0 {
						// All children effectively unchanged, so just hide this block
						hiddenBlockCount++
						childNodes = childNodes[:len(childNodes)-1] // Remove the block node
					} else {
						blockNode.Children = tempParent.Children

						// Add closing brace
						closingNode := createClosingBrace(depth, blockNode)
						childNodes = append(childNodes, closingNode)
					}
				} else if hasChanges {
					// Block has changes but they're all just formatting (empty string to null)
					// Add this to hidden blocks count instead of displaying it
					hiddenBlockCount++
					childNodes = childNodes[:len(childNodes)-1] // Remove the block node
				} else if hasHiddenMessage {
					// If there are only hidden attributes, don't show the block
					hiddenBlockCount++
					childNodes = childNodes[:len(childNodes)-1] // Remove the block node
				} else {
					// Empty block with no changes, skip it
					hiddenBlockCount++
					childNodes = childNodes[:len(childNodes)-1] // Remove the block node
				}
			}

		case []interface{}:
			// Handle arrays/lists
			if len(v) > 0 {
				// Get before/after arrays for this list
				var beforeList []interface{}
				var afterList []interface{}

				if beforeVal, ok := beforeValue.([]interface{}); ok {
					beforeList = beforeVal
				} else {
					beforeList = []interface{}{}
				}

				if afterVal, ok := afterValue.([]interface{}); ok {
					afterList = afterVal
				} else {
					afterList = []interface{}{}
				}

				// Process each item in the list
				maxLen := len(afterList)
				if len(beforeList) > maxLen {
					maxLen = len(beforeList)
				}

				var hiddenItems int
				var listItems []*TreeNode

				for i := 0; i < maxLen; i++ {
					var beforeItem, afterItem interface{}
					var itemChangeType string
					var itemPrefix string

					if i < len(beforeList) {
						beforeItem = beforeList[i]
					}

					if i < len(afterList) {
						afterItem = afterList[i]
					}

					// Determine change type for this list item
					if beforeItem == nil && afterItem != nil {
						itemChangeType = "create"
						itemPrefix = "+ "
					} else if beforeItem != nil && afterItem == nil {
						itemChangeType = "delete"
						itemPrefix = "- "
					} else if beforeItem != nil && afterItem != nil && !isEffectivelyEqual(beforeItem, afterItem) {
						itemChangeType = "update"
						itemPrefix = "~ "
					} else {
						itemChangeType = "no-op"
						itemPrefix = "  "
					}

					// Override prefix if parent node is being created or deleted
					if parentNode.ChangeType == "create" {
						itemPrefix = "+ "
						itemChangeType = "create"
					} else if parentNode.ChangeType == "delete" {
						itemPrefix = "- "
						itemChangeType = "delete"
					}

					// Skip unchanged items unless we're showing everything
					if itemChangeType == "no-op" && parentNode.ChangeType != "create" && parentNode.ChangeType != "delete" && parentNode.ChangeType != "replace" {
						hiddenItems++
						continue
					}

					// Use the item that exists, preferring after
					var item interface{}
					if i < len(afterList) {
						item = afterList[i]
					} else {
						item = beforeList[i]
					}

					if _, ok := item.(map[string]interface{}); ok {
						// For maps in a list, process as nested blocks
						// Create node and add directly to parent instead of to blockNode
						itemNode := &TreeNode{
							Text:       itemPrefix + key + "[" + strconv.Itoa(i) + "] {",
							Expanded:   false,
							Type:       "block",
							Depth:      depth,
							Parent:     parentNode,
							Toggleable: true,
							ChangeType: itemChangeType,
						}

						listItems = append(listItems, itemNode)

						// Handle nested unknown values
						var nestedUnknown map[string]interface{}
						if unknownAttrs != nil && i < len(afterList) {
							if unknownList, exists := unknownAttrs[key].([]interface{}); exists && i < len(unknownList) {
								if unknownMap, ok := unknownList[i].(map[string]interface{}); ok {
									nestedUnknown = unknownMap
								}
							}
						}

						// Get before/after maps for this list item
						var beforeItemMap map[string]interface{}
						var afterItemMap map[string]interface{}

						if i < len(beforeList) {
							if beforeMapItem, ok := beforeList[i].(map[string]interface{}); ok {
								beforeItemMap = beforeMapItem
							} else {
								beforeItemMap = make(map[string]interface{})
							}
						} else {
							beforeItemMap = make(map[string]interface{})
						}

						if i < len(afterList) {
							if afterMapItem, ok := afterList[i].(map[string]interface{}); ok {
								afterItemMap = afterMapItem
							} else {
								afterItemMap = make(map[string]interface{})
							}
						} else {
							afterItemMap = make(map[string]interface{})
						}

						// Recursive call with before/after for this list item
						processAttributes(itemNode, beforeItemMap, afterItemMap, nestedUnknown, depth+1, itemPrefix)

						// Check if this list item's changes are just empty string to null conversions
						hasRealChanges := false
						hasOnlyEmptyToNull := false

						// First check if there are any children that indicate changes
						hasChanges := itemChangeType != "no-op"

						// Then check each child to see if they're just empty string to null changes
						if hasChanges {
							hasOnlyEmptyToNull = true // Assume all changes are empty string to null until proven otherwise

							for _, child := range itemNode.Children {
								// Skip unchanged or comment elements
								if child.ChangeType == "no-op" || strings.HasPrefix(strings.TrimSpace(child.Text), "#") {
									continue
								}

								// Check if this is an empty string to null change
								isEmptyStringToNull := false
								if strings.Contains(child.Text, " -> ") {
									if strings.Contains(child.Text, "\"\"") && strings.Contains(child.Text, "null") {
										// This is likely an empty string to null change
										parts := strings.Split(child.Text, " -> ")
										if len(parts) == 2 {
											beforePart := strings.TrimSpace(parts[0])
											afterPart := strings.TrimSpace(parts[1])

											// Check for "" -> null or null -> ""
											if (beforePart == "\"\"" && afterPart == "null") ||
												(beforePart == "null" && afterPart == "\"\"") {
												isEmptyStringToNull = true
											}
										}
									}
								}

								// If this change is not an empty string to null conversion,
								// then the list item has real changes
								if !isEmptyStringToNull {
									hasOnlyEmptyToNull = false
									hasRealChanges = true
									break
								}
							}
						}

						// If this list item has only empty string to null changes, modify it to appear unchanged
						if hasOnlyEmptyToNull && !hasRealChanges {
							// Change the appearance of this list item to look unchanged
							itemNode.Text = "  " + strings.TrimLeft(itemNode.Text, "~+-") // Remove change indicators
							itemNode.ChangeType = "no-op"

							// Also add a hidden message inside the block
							hiddenNode := &TreeNode{
								Text:       "# (All attributes unchanged - only null representation differs)",
								Expanded:   true,
								Type:       "comment",
								Depth:      depth + 1,
								Parent:     itemNode,
								Toggleable: false,
								ChangeType: "no-op",
							}

							// Insert the hidden message as the first child
							if len(itemNode.Children) > 0 {
								itemNode.Children = append([]*TreeNode{hiddenNode}, itemNode.Children...)
							} else {
								itemNode.Children = []*TreeNode{hiddenNode}
							}
						}

						// Add closing brace
						itemClosingNode := createClosingBrace(depth, itemNode)
						listItems = append(listItems, itemClosingNode)
					} else {
						// Simple value in list - we'll keep the traditional structure for simple lists
						// Create a container for the list
						listNode := &TreeNode{
							Text:       attrPrefix + key + " {",
							Expanded:   false,
							Type:       "block",
							Depth:      depth,
							Parent:     parentNode,
							Toggleable: true,
							ChangeType: changeType,
						}

						childNodes = append(childNodes, listNode)

						// Add the list item
						valueText := fmt.Sprintf("%v", item)
						listItemNode := &TreeNode{
							Text:       itemPrefix + valueText,
							Expanded:   true,
							Type:       "attribute",
							Depth:      depth + 1,
							Parent:     listNode,
							Toggleable: false,
							ChangeType: itemChangeType,
						}

						listNode.Children = append(listNode.Children, listItemNode)

						// Add closing brace for simple list
						listClosingNode := createClosingBrace(depth, listNode)
						childNodes = append(childNodes, listClosingNode)

						// Skip further processing for this list
						continue
					}
				}

				// Add hidden items message if any
				if hiddenItems > 0 {
					// Check if all list items were modified to become "no-op" due to having only empty string to null changes
					allItemsNoOp := true
					for _, item := range listItems {
						if item.Type == "block" && item.ChangeType != "no-op" && !strings.HasPrefix(item.Text, "  ") {
							allItemsNoOp = false
							break
						}
					}

					// If all items are effectively no-op, we'll count the whole list as a hidden block
					// and not add it to the output at all
					if allItemsNoOp {
						hiddenBlockCount++
						// Skip adding any list items to the parent node
						continue
					}

					hiddenNode := &TreeNode{
						Text:       "# (" + strconv.Itoa(hiddenItems) + " unchanged items hidden)",
						Expanded:   true,
						Type:       "comment",
						Depth:      depth,
						Parent:     parentNode,
						Toggleable: false,
						ChangeType: "no-op",
					}

					listItems = append(listItems, hiddenNode)
				} else {
					// Even with no hidden items, check if all visible items are effectively no-op
					// (they might have been modified to appear unchanged)
					allItemsNoOp := true
					for _, item := range listItems {
						if item.Type == "block" && item.ChangeType != "no-op" && !strings.HasPrefix(item.Text, "  ") {
							allItemsNoOp = false
							break
						}
					}

					// If all items are effectively no-op, we'll count the whole list as a hidden block
					if allItemsNoOp && len(listItems) > 0 {
						hiddenBlockCount++
						// Skip adding any list items to the parent node
						continue
					}
				}

				// Add all list items directly to parent
				childNodes = append(childNodes, listItems...)
			} else if unknown {
				// Empty list that will be known after apply
				attributeNode := &TreeNode{
					Text:       attrPrefix + key + " = (known after apply)",
					Expanded:   true,
					Type:       "attribute",
					Depth:      depth,
					Parent:     parentNode,
					Toggleable: false,
					ChangeType: changeType,
				}

				childNodes = append(childNodes, attributeNode)
			} else {
				// Empty list with known values (empty list)
				attributeNode := &TreeNode{
					Text:       attrPrefix + key + " = []",
					Expanded:   true,
					Type:       "attribute",
					Depth:      depth,
					Parent:     parentNode,
					Toggleable: false,
					ChangeType: changeType,
				}

				childNodes = append(childNodes, attributeNode)
			}

		default:
			// This is a simple attribute
			var valueText string

			if unknown {
				valueText = "(known after apply)"
			} else if v == nil {
				valueText = "null"
			} else {
				valueText = fmt.Sprintf("%v", v)
				// Add quotes for string values
				if _, ok := v.(string); ok {
					valueText = "\"" + valueText + "\""
				}
			}

			// For updated values, show both before and after
			if changeType == "update" && beforeExists {
				// Get the before value text
				beforeText := "null"
				if beforeValue != nil {
					beforeText = fmt.Sprintf("%v", beforeValue)
					if _, ok := beforeValue.(string); ok {
						beforeText = "\"" + beforeText + "\""
					}
				}

				// Check if this is just an empty string to null change (or vice versa)
				beforeIsEmptyStr := beforeValue != nil && fmt.Sprintf("%v", beforeValue) == "" && reflect.TypeOf(beforeValue).Kind() == reflect.String
				afterIsNull := afterValue == nil
				afterIsEmptyStr := afterValue != nil && fmt.Sprintf("%v", afterValue) == "" && reflect.TypeOf(afterValue).Kind() == reflect.String
				beforeIsNull := beforeValue == nil

				// If it's an empty string to null change or vice versa, don't show it as a change
				if (beforeIsEmptyStr && afterIsNull) || (beforeIsNull && afterIsEmptyStr) {
					// Override the change type to no-op for the node
					changeType = "no-op"
					attrPrefix = "  "
					valueText = beforeText // Just show the original value
				} else {
					// Regular change formatting
					if unknown {
						valueText = beforeText + " -> (known after apply)"
					} else {
						valueText = beforeText + " -> " + valueText
					}
				}
			} else if changeType == "delete" && beforeExists {
				// For deleted values, show "value -> null"
				beforeText := "null"
				if beforeValue != nil {
					beforeText = fmt.Sprintf("%v", beforeValue)
					if _, ok := beforeValue.(string); ok {
						beforeText = "\"" + beforeText + "\""
					}
				}

				// Check if this is just an empty string being deleted
				beforeIsEmptyStr := beforeValue != nil && fmt.Sprintf("%v", beforeValue) == "" && reflect.TypeOf(beforeValue).Kind() == reflect.String

				// If it's an empty string being "deleted", don't show it as a change
				if beforeIsEmptyStr {
					// Override the change type to no-op for the node
					changeType = "no-op"
					attrPrefix = "  "
					valueText = beforeText // Just show the original value
				} else {
					valueText = beforeText + " -> null"
				}
			}

			attributeNode := &TreeNode{
				Text:       attrPrefix + key + " = " + valueText,
				Expanded:   true,
				Type:       "attribute",
				Depth:      depth,
				Parent:     parentNode,
				Toggleable: false,
				ChangeType: changeType,
			}

			childNodes = append(childNodes, attributeNode)
		}
	}

	// Add hidden attributes message if any
	if hiddenAttrCount > 0 {
		attributesMsg := "attribute"
		if hiddenAttrCount > 1 {
			attributesMsg = "attributes"
		}

		hiddenNode := &TreeNode{
			Text:       "# (" + strconv.Itoa(hiddenAttrCount) + " unchanged " + attributesMsg + " hidden)",
			Expanded:   true,
			Type:       "comment",
			Depth:      depth,
			Parent:     parentNode,
			Toggleable: false,
			ChangeType: "no-op",
		}

		childNodes = append(childNodes, hiddenNode)
	}

	// Add hidden blocks message if any
	if hiddenBlockCount > 0 {
		blocksMsg := "block"
		if hiddenBlockCount > 1 {
			blocksMsg = "blocks"
		}

		hiddenNode := &TreeNode{
			Text:       "# (" + strconv.Itoa(hiddenBlockCount) + " unchanged " + blocksMsg + " hidden)",
			Expanded:   true,
			Type:       "comment",
			Depth:      depth,
			Parent:     parentNode,
			Toggleable: false,
			ChangeType: "no-op",
		}

		childNodes = append(childNodes, hiddenNode)
	}

	// Add all child nodes to parent
	parentNode.Children = append(parentNode.Children, childNodes...)

	// Add any unknown attributes that aren't in the original map
	for k, v := range unknownAttrs {
		if _, existsInBefore := beforeAttrs[k]; !existsInBefore {
			if _, existsInAfter := afterAttrs[k]; !existsInAfter {
				// Only process true boolean unknowns that don't exist in attrs
				if boolVal, ok := v.(bool); ok && boolVal {
					attributeNode := &TreeNode{
						Text:       "+ " + k + " = (known after apply)",
						Expanded:   true,
						Type:       "attribute",
						Depth:      depth,
						Parent:     parentNode,
						Toggleable: false,
						ChangeType: "create",
					}

					parentNode.Children = append(parentNode.Children, attributeNode)
				} else if mapVal, ok := v.(map[string]interface{}); ok && len(mapVal) > 0 {
					// This is a block that exists only in unknown
					blockNode := &TreeNode{
						Text:       "+ " + k + " (known after apply)",
						Expanded:   false,
						Type:       "block",
						Depth:      depth,
						Parent:     parentNode,
						Toggleable: true,
						ChangeType: "create",
					}

					parentNode.Children = append(parentNode.Children, blockNode)
				}
			}
		}
	}
}

// formatMapEntries formats a map as indented key-value pairs
func formatMapEntries(parentNode *TreeNode, mapData map[string]interface{}, depth int, prefix string) {
	// Sort keys for consistent output
	var keys []string
	for k := range mapData {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := mapData[key]

		var valueText string
		if value == nil {
			valueText = "null"
		} else {
			valueText = fmt.Sprintf("%v", value)
			// Add quotes for string values
			if _, ok := value.(string); ok {
				valueText = "\"" + valueText + "\""
			}
		}

		// Determine the appropriate prefix for this entry based on parent's changeType
		entryPrefix := prefix
		entryChangeType := parentNode.ChangeType

		// Ensure we use correct change type and prefix for map entries
		switch parentNode.ChangeType {
		case "create":
			entryPrefix = "+ "
			entryChangeType = "create"
		case "delete":
			entryPrefix = "- "
			entryChangeType = "delete"
		case "update":
			entryPrefix = "~ "
			entryChangeType = "update"
		case "replace":
			entryPrefix = "~ "
			entryChangeType = "update"
		}

		attributeNode := &TreeNode{
			Text:       entryPrefix + key + " = " + valueText,
			Expanded:   true,
			Type:       "attribute",
			Depth:      depth,
			Parent:     parentNode,
			Toggleable: false,
			ChangeType: entryChangeType, // Use the determined change type, not just parent's
		}

		parentNode.Children = append(parentNode.Children, attributeNode)
	}
}
