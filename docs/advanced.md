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

## Managing Multiple Terraform Workspaces

TFApp works seamlessly with Terraform workspaces for managing multiple environments.

### Setting Up Workspaces

```bash
# Create a new workspace
terraform workspace new staging

# Use TFApp with the current workspace
tfapp
```

### Workflow with Workspaces

```bash
# Development environment
terraform workspace select dev
tfapp -- -var-file=dev.tfvars

# Staging environment
terraform workspace select staging
tfapp -- -var-file=staging.tfvars

# Production environment
terraform workspace select prod
tfapp -- -var-file=prod.tfvars
```

## CI/CD Pipeline Integration

TFApp can be integrated into CI/CD pipelines for automated infrastructure deployment.

### Example GitHub Actions Workflow

```yaml
name: Deploy Infrastructure

on:
  push:
    branches: [ main ]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v1
        
      - name: Setup Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.24
          
      - name: Install TFApp
        run: |
          git clone https://github.com/yourusername/tfapp.git
          cd tfapp
          go build -o build/tfapp ./cmd/tfapp
          sudo cp build/tfapp /usr/local/bin/
          
      - name: Deploy Infrastructure
        run: tfapp -- -auto-approve -var-file=ci.tfvars
```

### Jenkins Pipeline Example

```groovy
pipeline {
    agent any
    
    tools {
        go 'go1.24'
        terraform 'terraform'
    }
    
    stages {
        stage('Prepare') {
            steps {
                git 'https://github.com/yourusername/tfapp.git'
                sh 'go build -o build/tfapp ./cmd/tfapp'
                sh 'cp build/tfapp /usr/local/bin/'
            }
        }
        
        stage('Deploy') {
            steps {
                dir('terraform-configs') {
                    sh 'tfapp -- -auto-approve -var-file=jenkins.tfvars'
                }
            }
        }
    }
}
```

## Working with Remote State

TFApp works with Terraform's remote state backends for team collaboration and state management.

### Setting Up Remote State

```bash
# Initialize with backend configuration
tfapp -init -- -backend-config=backend.hcl
```

### Example backend.hcl

```hcl
bucket = "my-terraform-state"
key    = "terraform.tfstate"
region = "us-west-2"
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

### Working with Local Modules

```bash
# Navigate to module directory
cd modules/networking

# Run TFApp in the module directory
tfapp
```

## Handling Large Infrastructure

When working with large Terraform projects, consider these techniques:

### State Filtering

```bash
# Apply only compute resources
tfapp -- -target="module.compute.*"

# Apply only network resources
tfapp -- -target="module.network.*"
```

### Using TFApp's Target Selection

For large infrastructures, the target selection menu helps avoid applying unwanted changes:

1. Run `tfapp`
2. Select "Do a target apply"
3. Select resources by category or group
4. Apply your changes

## Advanced Configuration Management

Managing configurations across environments:

### Using Multiple Variable Files

```bash
# Base configuration with environment-specific overrides
tfapp -- -var-file=base.tfvars -var-file=prod.tfvars
```

### Environment-Specific Workflows

```bash
# Production deployment script
#!/bin/bash
set -e

# Ensure we're in the right workspace
terraform workspace select prod

# Initialize if needed
tfapp -init -- -backend-config=prod-backend.hcl

# Apply with production variables
tfapp -- -var-file=prod.tfvars
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

### Managing Provider Versions

```bash
# Create a provider constraints file
cat > .terraform.lock.hcl << EOF
provider "aws" {
  version = "~> 4.0"
}
EOF

# Initialize with the lock file
tfapp -init
``` 