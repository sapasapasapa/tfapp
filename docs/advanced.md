# Advanced Features

This guide covers the advanced features and techniques for getting the most out of TFApp.

## Targeted Resource Application

TFApp's targeted apply feature allows you to selectively apply changes to specific resources rather than applying the entire plan.

### When to Use Targeted Applies

- When deploying changes incrementally to reduce risk
- When fixing a specific resource without affecting others
- When working with large infrastructure where full applies take too long
- When testing changes to a specific component

### How to Use Targeted Applies

1. Run TFApp normally:
   ```bash
   tfapp
   ```

2. From the interactive menu, select "Do a target apply"

3. In the checkbox menu:
   - Use arrow keys (↑/↓) to navigate
   - Press Space to select/deselect resources
   - Press Enter to confirm your selection

4. Review the targeted plan that's generated
   
5. Choose from the interactive menu again:
   - Apply Plan: Apply only the selected resources
   - Show Full Plan: See the targeted plan details
   - Exit: Cancel the operation

### Behind the Scenes

When you select resources for targeted apply, TFApp:

1. Adds the `-target` flag for each selected resource
2. Generates a new plan with only those targets
3. Presents the menu again for the targeted plan

### Example: Targeted Deployment

```bash
# Start TFApp
tfapp

# Select "Do a target apply" in the menu
# Select only the database resources
# Apply the targeted plan
```

## Managing Terraform Modules

TFApp supports Terraform modules and can help manage them efficiently.

### Initializing with Modules

```bash
# Initialize and download modules
tfapp -init

# Update modules to latest versions
tfapp -init-upgrade
```

## Troubleshooting Advanced Scenarios

### Debugging TFApp

If you encounter issues with TFApp, you can inspect what Terraform commands are being executed:

```bash
# Run TFApp with verbose output
TF_LOG=DEBUG tfapp
```

### Resolving State Lock Issues

If a previous Terraform run was interrupted and left the state locked:

```bash
# Force unlock the state
terraform force-unlock LOCK_ID

# Then run TFApp
tfapp
```
