# Usage Guide

## Basic Usage

TFApp is designed to be intuitive and straightforward to use. To get started with a basic workflow:

```bash
cd /path/to/terraform/project
tfapp
```

This will:
1. Generate a Terraform plan
2. Display an interactive menu for further actions
3. Allow you to apply the plan or view details

## Command-line Flags

TFApp supports several command-line flags to customize its behavior:

| Flag | Description |
|------|-------------|
| `-init` | Run `terraform init` before creating a plan |
| `-init-upgrade` | Run `terraform init -upgrade` to update modules and providers |

## Arguments and Pass-through Options

TFApp passes any additional arguments after your flags to Terraform. To differentiate between TFApp flags and Terraform arguments, use the `--` separator:

```bash
tfapp [tfapp-flags] -- [terraform-arguments]
```

### Examples of Pass-through Arguments

```bash
# Pass variables to terraform
tfapp -- -var="environment=production" -var="region=us-west-2"

# Use a variable file
tfapp -- -var-file=production.tfvars

# Specify a different state file
tfapp -- -state=custom.tfstate

# Use auto-approval (non-interactive mode)
tfapp -- -auto-approve
```

## The Interactive Menu

After generating a plan, TFApp displays an interactive menu with the following options:

### Apply Plan

Executes `terraform apply` with the generated plan file. This option:
- Applies all changes in the plan
- Shows real-time output
- Maintains Terraform's interactive approvals

### Show Full Plan

Displays the detailed Terraform plan with:
- Resource additions, changes, and deletions
- Attribute changes
- Color-coded output for better readability

### Do a Target Apply

Allows selective application of the plan:
1. Presents a checkbox menu of all resources in the plan
2. Use space to select/deselect resources
3. Press Enter to confirm selections
4. Shows a new plan with only the selected resources
5. Presents the main menu again for the targeted plan

### Exit

Exits the application without making any changes.

## Navigation Controls

While using TFApp's interactive components:

### Menu Navigation
- Use arrow keys (↑/↓) to navigate menu items
- Press Enter to select an option

### Checkbox Selection
- Use arrow keys (↑/↓) to navigate between items
- Press Space to toggle selection
- Press Enter to confirm selections

### Plan Viewer Navigation
TFApp includes an interactive plan viewer with collapsible sections:

- **Basic Navigation**
  - Use arrow keys (↑/↓) or (j/k) to move up and down
  - Press Home/g to jump to the top of the plan
  - Press End/G to jump to the bottom of the plan

- **Expanding/Collapsing Sections**
  - Press Right arrow, l, Enter, or Space to expand a node
  - Press Left arrow or h to collapse a node or jump to parent
  - Press a to expand all nodes recursively
  - Press n to collapse all nodes except root level

- **Visual Indicators**
  - Purple triangles (▶/▼) indicate expandable/collapsible sections
  - A status bar at the bottom shows your current position and percentage
  - Messages indicate if there's more content above or below

- **Help System**
  - Press ? at any time to view a complete list of navigation commands
  - The help tooltip shows all available keyboard shortcuts
  - Press ? again to hide the help overlay

### During Terraform Operations
- Ctrl+C to interrupt operations
- Standard Terraform prompts will appear for confirmation

## Advanced Features

### Responsive Plan Viewer

The TFApp plan viewer automatically adapts to your terminal window size:

- The viewer adjusts its height when you resize your terminal
- Content is scrollable for plans of any size
- The status bar always displays at the bottom of the window
- All UI elements scale appropriately with the window size

### Smart Node Collapsing

When viewing complex plans, TFApp provides smart node management:

- Expand a single node to see its immediate children
- Expand all nodes recursively using the 'a' key
- Collapse all nodes except root level with the 'n' key
- When a node is already expanded, pressing Enter collapses all its children while keeping the node itself expanded
  - This is useful for quickly clearing out nested content while maintaining your place in the hierarchy

### Visual Feedback

TFApp provides clear visual feedback during navigation:

- Resources are automatically colorized based on action type:
  - Green for creations
  - Yellow for updates
  - Red for deletions
- The selected line is highlighted for easy tracking
- The status bar shows your exact position and percentage through the document
- Indicators tell you when there's more content above or below your current view

## Workflow Examples

### Development Workflow

```bash
# Initialize with latest modules and apply with dev variables
tfapp -init-upgrade -- -var-file=dev.tfvars
```

### Production Deployment

```bash
# Initialize and apply with production variables
tfapp -init -- -var-file=prod.tfvars
```

### CI/CD Pipeline Usage

```bash
# Non-interactive mode with automatic approval (for CI/CD)
tfapp -- -auto-approve -var-file=ci.tfvars
```

### Working with Workspaces

```bash
# Create and select a new workspace before planning
terraform workspace new staging
tfapp -- -var-file=staging.tfvars
```

### Using with Remote State

```bash
# Initialize with backend configuration
tfapp -init -- -backend-config=backend.hcl
```

## Error Handling

TFApp provides informative error messages with color-coding to help diagnose issues:
- Red text indicates errors
- Yellow text indicates warnings
- Blue text indicates informational messages

When an error occurs, TFApp will:
1. Display the error message
2. Provide context about what operation failed
3. Exit with a non-zero status code for script integration 