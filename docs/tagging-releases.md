# Creating and Pushing Tags for Releases

This guide explains how to create and push Git tags to trigger the automated release process for TFApp.

## Understanding Semantic Versioning

TFApp follows [Semantic Versioning](https://semver.org/) for release numbering:

- **MAJOR.MINOR.PATCH** (e.g., `1.2.3`)
- Increment MAJOR version for incompatible API changes
- Increment MINOR version for backward-compatible new features
- Increment PATCH version for backward-compatible bug fixes

All release tags should be prefixed with `v` (e.g., `v1.2.3`).

## Preparing for a Release

Before creating a release tag:

1. **Ensure your code is ready**:
   - All features and fixes are complete and tested
   - The documentation is up-to-date
   - Any version references in the code are updated

2. **Update the version information**:
   - If needed, update the version constants in `internal/version/version.go`
   - The actual version will be taken from the Git tag, but it's good practice to keep them in sync

3. **Commit all changes**:
   ```bash
   git add .
   git commit -m "Prepare for v0.1.0 release"
   git push origin main
   ```

## Creating a Release Tag

1. **Make sure you're on the correct branch**:
   ```bash
   # Usually main or master
   git checkout main
   
   # Pull the latest changes
   git pull origin main
   ```

2. **Create an annotated tag**:
   ```bash
   # For a regular release:
   git tag -a v0.1.0 -m "Release v0.1.0"
   
   # For a pre-release (if needed):
   git tag -a v0.1.0-beta.1 -m "Beta release v0.1.0-beta.1"
   ```

3. **Push the tag to GitHub**:
   ```bash
   # Push a specific tag
   git push origin v0.1.0
   
   # Or push all tags (use with caution)
   git push origin --tags
   ```

## Monitoring the Release Process

After pushing the tag:

1. **Check GitHub Actions**:
   - Go to the "Actions" tab in your repository
   - Look for the workflow run triggered by the tag push
   - Monitor the build progress

2. **Verify Release Assets**:
   - Once the workflow completes, go to the "Releases" section of your repository
   - Check that all expected assets are attached to the release
   - Verify the changelog is populated correctly

3. **Check Homebrew Tap**:
   - Verify the formula was created/updated in your homebrew-tap repository
   - Navigate to `sapasapasapa/homebrew-tap` and check the `Formula` directory

## Testing the Released Version

After a successful release:

1. **Install via Homebrew**:
   ```bash
   # Update Homebrew
   brew update
   
   # Install the formula
   brew install sapasapasapa/homebrew-tap/tfapp
   ```

2. **Verify installation**:
   ```bash
   # Check the version
   tfapp --version
   
   # Test functionality
   tfapp -h
   ```

## Handling Failed Releases

If the release process fails:

1. **Check the workflow logs** for specific errors
2. **Fix any issues** in your configuration or code
3. **Delete the tag** locally and remotely:
   ```bash
   # Delete local tag
   git tag -d v0.1.0
   
   # Delete remote tag
   git push --delete origin v0.1.0
   ```
4. **Create a new tag** once issues are resolved

## Release Cadence Best Practices

- **Document your changes** in a CHANGELOG.md file
- **Consider using pre-releases** (alpha/beta) for significant changes
- **Be consistent with your versioning** to build user trust
- **Test release process** with -test suffix tags before official releases 