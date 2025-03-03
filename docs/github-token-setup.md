# Setting Up GitHub Token for Releases

This guide explains how to set up the GitHub token required for releasing TFApp to Homebrew.

## Creating a Personal Access Token (PAT)

1. **Generate a new token**:
   - Go to your GitHub account settings: https://github.com/settings/tokens
   - Click "Generate new token" > "Generate new token (classic)"
   - Give it a descriptive name like "TFApp Homebrew Publishing"
   - Set an appropriate expiration date (or "No expiration" if you'll manage it manually)

2. **Select the required permissions**:
   - At minimum, check the `repo` scope (Full control of private repositories)
   - If you need to access workflows across repositories, also select `workflow`

3. **Generate and copy the token**:
   - Click "Generate token" at the bottom of the page
   - **IMPORTANT**: Copy the token immediately! GitHub will only show it once.

## Adding the Token to GitHub Repository Secrets

1. **Go to your TFApp repository**:
   - Navigate to the repository on GitHub

2. **Access repository secrets**:
   - Click "Settings" > "Secrets and variables" > "Actions"

3. **Add a new secret**:
   - Click "New repository secret"
   - Name: `HOMEBREW_TAP_GH_TOKEN` (exactly as referenced in .goreleaser.yml)
   - Value: Paste the personal access token you copied
   - Click "Add secret"

## Verifying Token Setup

1. **Create a tag to test the release process**:
   ```bash
   git tag -a v0.1.0-test -m "Test release"
   git push origin v0.1.0-test
   ```

2. **Check the GitHub Actions workflow**:
   - Go to the "Actions" tab in your repository
   - Verify that the workflow runs without token-related errors

3. **Check Homebrew tap**:
   - After a successful run, check your homebrew-tap repository
   - Verify that a new formula has been created

## Troubleshooting Token Issues

If you encounter errors related to authentication or permissions:

1. **Check the token scope**:
   - Ensure the token has the `repo` scope
   - If releasing to an organization's repository, make sure you have the necessary organization permissions

2. **Verify the token name**:
   - Confirm that the secret name in GitHub (`HOMEBREW_TAP_GH_TOKEN`) matches what's referenced in your configuration files

3. **Check token expiration**:
   - Tokens can expire; verify your token is still valid

4. **Repository access**:
   - Ensure the token has access to both your main repository and the homebrew-tap repository

5. **Workflow permission issues**:
   - Check that the workflow has the necessary permissions in the YAML file (contents: write, etc.)

## Security Considerations

- **Never commit tokens to your repository**
- **Use repository secrets for all tokens**
- **Rotate tokens periodically** for better security
- **Use the minimum necessary permissions** for the token
- **Set reasonable expiration dates** on tokens 