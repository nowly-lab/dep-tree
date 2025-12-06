#!/bin/bash

# Release script for nowly-tree
# This script handles version tagging and Homebrew release via GoReleaser

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS] [version]"
    echo ""
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  -d, --dry-run  Perform a dry run without actually creating tags or releases"
    echo "  -f, --force    Force release even if tag already exists"
    echo "  -p, --patch    Increment patch version (default behavior)"
    echo "  -m, --minor    Increment minor version"
    echo "  -M, --major    Increment major version"
    echo ""
    echo "Arguments:"
    echo "  version        Specific version to release (e.g., 0.23.7, 1.0.0)"
    echo "                 If not provided, automatically increments patch version"
    echo ""
    echo "Examples:"
    echo "  $0                           # Auto-increment patch version"
    echo "  $0 --minor                   # Auto-increment minor version"
    echo "  $0 --major                   # Auto-increment major version"
    echo "  $0 0.23.7                    # Release specific version 0.23.7"
    echo "  $0 --dry-run                 # Dry run with auto-increment"
    echo "  $0 --force 0.23.7            # Force release specific version"
}

# Function to get current version from cmd/root.go
get_current_version() {
    grep 'Version:' cmd/root.go | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+' | sed 's/v//'
}

# Function to increment version
increment_version() {
    local version=$1
    local increment_type=$2

    IFS='.' read -ra VERSION_PARTS <<< "$version"
    local major=${VERSION_PARTS[0]}
    local minor=${VERSION_PARTS[1]}
    local patch=${VERSION_PARTS[2]}

    case $increment_type in
        "major")
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        "minor")
            minor=$((minor + 1))
            patch=0
            ;;
        "patch")
            patch=$((patch + 1))
            ;;
        *)
            print_error "Invalid increment type: $increment_type"
            exit 1
            ;;
    esac

    echo "$major.$minor.$patch"
}

# Parse command line arguments
DRY_RUN=false
FORCE=false
VERSION=""
INCREMENT_TYPE="patch"  # default to patch increment
AUTO_INCREMENT=true

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        -d|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -f|--force)
            FORCE=true
            shift
            ;;
        -m|--minor)
            INCREMENT_TYPE="minor"
            shift
            ;;
        -p|--patch)
            INCREMENT_TYPE="patch"
            shift
            ;;
        -M|--major)
            INCREMENT_TYPE="major"
            shift
            ;;
        -*)
            print_error "Unknown option $1"
            show_usage
            exit 1
            ;;
        *)
            if [[ -z "$VERSION" ]]; then
                VERSION="$1"
                AUTO_INCREMENT=false
            else
                print_error "Multiple versions specified"
                show_usage
                exit 1
            fi
            shift
            ;;
    esac
done

# Auto-increment version if not provided
if [[ "$AUTO_INCREMENT" == "true" ]]; then
    CURRENT_VERSION=$(get_current_version)
    if [[ -z "$CURRENT_VERSION" ]]; then
        print_error "Could not determine current version from cmd/root.go"
        exit 1
    fi

    VERSION=$(increment_version "$CURRENT_VERSION" "$INCREMENT_TYPE")
    print_info "Auto-incrementing $INCREMENT_TYPE version: $CURRENT_VERSION → $VERSION"
fi

# Validate version argument
if [[ -z "$VERSION" ]]; then
    print_error "Version could not be determined"
    show_usage
    exit 1
fi

# Validate version format (semantic versioning)
if ! [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
    print_error "Invalid version format. Use semantic versioning (e.g., 1.0.0, 1.0.0-beta.1)"
    exit 1
fi

TAG="v$VERSION"

print_info "Starting release process for version $VERSION"

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    print_error "Not in a git repository"
    exit 1
fi

# Check if working directory is clean
if [[ -n $(git status --porcelain) ]]; then
    print_error "Working directory is not clean. Please commit or stash your changes."
    git status --short
    exit 1
fi

# Check if we're on main branch
CURRENT_BRANCH=$(git branch --show-current)
if [[ "$CURRENT_BRANCH" != "main" ]]; then
    print_warning "You are not on the main branch (current: $CURRENT_BRANCH)"
    read -p "Do you want to continue? [y/N]: " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Release cancelled"
        exit 0
    fi
fi

# Check if tag already exists
if git tag -l | grep -q "^$TAG$"; then
    if [[ "$FORCE" == "true" ]]; then
        print_warning "Tag $TAG already exists, but --force was specified"
        if [[ "$DRY_RUN" == "false" ]]; then
            print_info "Deleting existing tag $TAG"
            git tag -d "$TAG"
            git push origin ":refs/tags/$TAG" || true
        fi
    else
        print_error "Tag $TAG already exists. Use --force to overwrite or choose a different version."
        exit 1
    fi
fi

# Check if required tools are installed
print_info "Checking required tools..."

if ! command -v goreleaser &> /dev/null; then
    print_error "goreleaser is not installed. Please install it first:"
    print_error "  brew install goreleaser"
    print_error "  or visit: https://goreleaser.com/install/"
    exit 1
fi

if ! command -v git &> /dev/null; then
    print_error "git is not installed"
    exit 1
fi

print_success "All required tools are available"

# Update version in cmd/root.go
print_info "Updating version in cmd/root.go..."
if [[ "$DRY_RUN" == "false" ]]; then
    # Use a more precise sed pattern to match the exact format
    sed -i.bak 's/Version:[[:space:]]*"v[0-9]\+\.[0-9]\+\.[0-9]\+"/Version:           "v'$VERSION'"/' cmd/root.go
    rm cmd/root.go.bak
    print_success "Version updated in cmd/root.go"

    # Verify the update was successful
    if grep -q "Version:.*\"v$VERSION\"" cmd/root.go; then
        print_success "Version verification successful: v$VERSION"
    else
        print_error "Version update failed - version not found in cmd/root.go"
        exit 1
    fi
else
    print_info "[DRY RUN] Would update version in cmd/root.go to v$VERSION"
fi

# Commit and push version update before creating tag
if [[ "$DRY_RUN" == "false" ]]; then
    if [[ -n $(git status --porcelain cmd/root.go) ]]; then
        print_info "Committing version update..."
        git add cmd/root.go
        git commit -m "chore: bump version to v$VERSION"
        print_success "Version update committed"

        print_info "Pushing version update to origin..."
        git push origin "$CURRENT_BRANCH"
        print_success "Version update pushed"
    fi
else
    print_info "[DRY RUN] Would commit and push version update"
fi

# Create and push tag
if [[ "$DRY_RUN" == "false" ]]; then
    print_info "Creating tag $TAG..."
    git tag -a "$TAG" -m "Release $TAG"
    print_success "Tag $TAG created"

    print_info "Pushing tag to origin..."
    git push origin "$TAG"
    print_success "Tag pushed to origin"
else
    print_info "[DRY RUN] Would create and push tag $TAG"
fi

# Run GoReleaser
print_info "Running GoReleaser..."
if [[ "$DRY_RUN" == "false" ]]; then
    # Check if GITHUB_TOKEN is set
    if [[ -z "$GITHUB_TOKEN" ]]; then
        print_warning "GITHUB_TOKEN environment variable is not set"
        print_info "GoReleaser needs a GitHub token to create releases and update Homebrew formula"
        print_info "Please set GITHUB_TOKEN and run: goreleaser release --clean"
        exit 1
    fi

    goreleaser release --clean
    print_success "GoReleaser completed successfully"
else
    print_info "[DRY RUN] Would run: goreleaser release --clean"
    goreleaser check
    print_success "GoReleaser configuration is valid"
fi

# Pull GoReleaser changes (Formula updates) and sync local repository
if [[ "$DRY_RUN" == "false" ]]; then
    print_info "Syncing local repository with GoReleaser changes..."

    # Fetch all changes from remote
    git fetch origin

    # Check if there are new commits on the remote branch
    LOCAL_COMMIT=$(git rev-parse HEAD)
    REMOTE_COMMIT=$(git rev-parse origin/"$CURRENT_BRANCH")

    if [[ "$LOCAL_COMMIT" != "$REMOTE_COMMIT" ]]; then
        print_info "New commits detected on remote, pulling changes..."
        if git pull origin "$CURRENT_BRANCH" --rebase; then
            print_success "Successfully synced with remote repository"
        else
            print_warning "Failed to sync with remote, but release was successful"
            print_info "Manual sync may be required:"
            print_info "  git pull origin $CURRENT_BRANCH"
        fi
    else
        print_info "Local repository is already up to date"
    fi
else
    print_info "[DRY RUN] Would sync local repository with remote changes"
fi

print_success "Release process completed successfully!"
print_info "Release details:"
print_info "  Version: $VERSION"
print_info "  Tag: $TAG"
print_info "  Branch: $CURRENT_BRANCH"

if [[ "$DRY_RUN" == "false" ]]; then
    print_info ""
    print_info "The release should be available at:"
    print_info "  https://github.com/nowly-lab/dep-tree/releases/tag/$TAG"
    print_info ""
    print_info "Homebrew formula has been automatically updated in the Formula/ directory"
    print_info ""
    print_info "Users can update to the new version with:"
    print_info "  brew update"
    print_info "  brew upgrade nowly-tree"
    print_info ""
    print_info "Or install fresh with:"
    print_info "  brew tap nowly-lab/dep-tree"
    print_info "  brew install nowly-tree"
    print_info ""
    print_info "Note: It may take a few minutes for the Homebrew tap to reflect the changes."
    print_info "If users don't see the update immediately, they can run:"
    print_info "  brew untap nowly-lab/dep-tree && brew tap nowly-lab/dep-tree"
fi