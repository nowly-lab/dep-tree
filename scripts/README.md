# Scripts

This directory contains utility scripts for the nowly-tree project.

## release.sh

Automated release script that handles version tagging and Homebrew release via GoReleaser.

### Prerequisites

1. **GoReleaser**: Install via Homebrew
   ```bash
   brew install goreleaser
   ```

2. **GitHub Token**: Set the `GITHUB_TOKEN` environment variable
   ```bash
   export GITHUB_TOKEN="your_github_token_here"
   ```

   The token needs the following permissions:
   - `repo` (full repository access)
   - `write:packages` (for publishing releases)

### Usage

```bash
# Auto-increment patch version (default)
./scripts/release.sh

# Auto-increment minor version
./scripts/release.sh --minor

# Auto-increment major version
./scripts/release.sh --major

# Release specific version
./scripts/release.sh 0.23.7

# Dry run (test without making changes)
./scripts/release.sh --dry-run

# Force release (overwrite existing tag)
./scripts/release.sh --force

# Show help
./scripts/release.sh --help
```

### What the script does

1. **Validation**:
   - Checks if you're in a git repository
   - Verifies working directory is clean
   - Validates version format (semantic versioning)
   - Checks if required tools are installed

2. **Version Update**:
   - Updates the version in `cmd/root.go`
   - Commits the version change

3. **Tagging**:
   - Creates a git tag with the specified version
   - Pushes the tag to origin

4. **Release**:
   - Runs GoReleaser to build binaries and create GitHub release
   - Automatically updates the Homebrew formula in `Formula/nowly-tree.rb`

### Examples

```bash
# Auto-increment patch version (e.g., 0.23.6 → 0.23.7) - default
./scripts/release.sh

# Auto-increment minor version (e.g., 0.23.6 → 0.24.0)
./scripts/release.sh --minor

# Auto-increment major version (e.g., 0.23.6 → 1.0.0)
./scripts/release.sh --major

# Release specific version
./scripts/release.sh 0.24.0

# Test the release process without making changes
./scripts/release.sh --dry-run

# Force release even if tag already exists
./scripts/release.sh --force 0.24.0

# Combine options (dry run with minor increment)
./scripts/release.sh --dry-run --minor
```

### Environment Variables

- `GITHUB_TOKEN`: Required for creating GitHub releases and updating Homebrew formula
- `GORELEASER_CURRENT_TAG`: Automatically set by the script

### Troubleshooting

1. **"goreleaser is not installed"**:
   ```bash
   brew install goreleaser
   ```

2. **"GITHUB_TOKEN environment variable is not set"**:
   ```bash
   export GITHUB_TOKEN="your_token_here"
   ```

3. **"Working directory is not clean"**:
   Commit or stash your changes before running the release script.

4. **"Tag already exists"**:
   Use `--force` flag to overwrite existing tag, or choose a different version.

### Installation Instructions for Users

```bash
# Install nowly-tree
brew tap nowly-lab/dep-tree https://github.com/nowly-lab/dep-tree
brew install nowly-tree

# Update to latest version
brew update
brew upgrade nowly-tree
```

### Manual Release (if script fails)

If the automated script fails, you can run the steps manually:

```bash
# 1. Update version in cmd/root.go
# 2. Commit the change
git add cmd/root.go
git commit -m "chore: bump version to v0.24.0"

# 3. Create and push tag
git tag -a v0.24.0 -m "Release v0.24.0"
git push origin v0.24.0

# 4. Run GoReleaser
export GITHUB_TOKEN="your_token_here"
goreleaser release --clean