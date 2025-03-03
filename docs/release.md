# Release Process

This document outlines the process for releasing new versions of TFApp.

## Requirements

- [GoReleaser](https://goreleaser.com/) installed locally for testing
- GitHub account with access to the repository
- GitHub Personal Access Token with `repo` scope (for publishing to Homebrew tap)

For detailed instructions on setting up the GitHub token, see [GitHub Token Setup](./github-token-setup.md).

## Configuration

TFApp uses GoReleaser for building and releasing binaries and publishing to Homebrew. The configuration is in `.goreleaser.yml` in the root of the repository.

Key configuration aspects:
- Builds for macOS and Linux on amd64 and arm64 architectures
- Homebrew formula generation for sapasapasapa/homebrew-tap
- Version information injected via ldflags

## Release Steps

1. **Update Version:**
   Update the version numbers in `internal/version/version.go` if needed (these are fallbacks and will be overridden by the tagged version).

2. **Create and Push Tag:**
   ```bash
   # Ensure you're on the main branch with the latest changes
   git checkout main
   git pull origin main
   
   # Create an annotated tag
   git tag -a v0.1.0 -m "Release v0.1.0"
   
   # Push the tag to GitHub
   git push origin v0.1.0
   ```

   For more detailed instructions on tagging releases, see [Tagging Releases](./tagging-releases.md).

3. **Automated Release Process:**
   Once the tag is pushed, GitHub Actions will automatically:
   - Build the application for all supported platforms (macOS and Linux)
   - Create a GitHub Release with binaries
   - Update the Homebrew formula in `sapasapasapa/homebrew-tap`

4. **Verify Release:**
   - Check the GitHub Actions build logs
   - Verify GitHub Release is created with all binaries
   - Verify Homebrew formula is updated in your tap repository
   - Test installation via Homebrew:
     ```bash
     brew update
     brew install sapasapasapa/homebrew-tap/tfapp
     ```

## Testing Locally

To test the release process locally without publishing:

```bash
# Test the build process
goreleaser release --snapshot --clean --skip=publish
```

This will create a local build in the `dist/` directory without publishing to GitHub or Homebrew. You can inspect the generated files to verify everything looks correct.

## Troubleshooting

### GitHub Token Issues

If you encounter GitHub token issues:

1. Ensure the `GITHUB_TOKEN` secret is properly set up in GitHub repository settings under "Settings > Secrets and variables > Actions".
2. For the Homebrew tap, you need a token with `repo` scope permissions.
3. For local testing, set the token in your environment:
   ```bash
   export GITHUB_TOKEN=your_github_personal_access_token
   ```

### Build Failures

If builds fail:

1. Check if all dependencies are properly specified in go.mod
2. Verify the goreleaser configuration in `.goreleaser.yml`
3. Ensure the Go version specified in the GitHub workflow matches the version you're using locally
4. Check the version format - it should follow semantic versioning (e.g., v0.1.0)

### Homebrew Formula Issues

If the Homebrew formula isn't published correctly:

1. Check that the repository path is correct (sapasapasapa/homebrew-tap)
2. Verify your GitHub token has write permissions to the tap repository 
3. Try running a local test with the `--debug` flag to see more details:
   ```bash
   goreleaser release --snapshot --clean --skip=publish --debug
   ``` 