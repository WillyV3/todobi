#!/bin/bash

# Full Release Pipeline for todobi
# Usage: ./scripts/release.sh [patch|minor|major] ["optional commit message"]

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

# Configuration
REPO_ORG="WillyV3"
REPO_NAME="todobi"
HOMEBREW_TAP_PATH="$HOME/homebrew-tap"
FORMULA_PATH="$HOMEBREW_TAP_PATH/Formula/todobi.rb"

# Function to print colored output
print_step() {
    echo -e "${BLUE}→${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
    exit 1
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Get current version from tags
get_current_version() {
    git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"
}

# Calculate next version
get_next_version() {
    local current=$1
    local bump_type=$2

    # Remove 'v' prefix
    version=${current#v}

    # Split into parts
    IFS='.' read -r major minor patch <<< "$version"

    case $bump_type in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch)
            patch=$((patch + 1))
            ;;
        *)
            echo "$current"
            return
            ;;
    esac

    echo "v${major}.${minor}.${patch}"
}

# Generate changelog from commits
generate_changelog() {
    local from_tag=$1
    local to_tag=$2

    echo "## What's Changed"
    echo ""

    # Group commits by type
    local features=""
    local fixes=""
    local other=""

    while IFS= read -r commit; do
        if [[ $commit == *"feat:"* ]] || [[ $commit == *"add:"* ]] || [[ $commit == *"Add"* ]]; then
            features="${features}- ${commit}\n"
        elif [[ $commit == *"fix:"* ]] || [[ $commit == *"Fix"* ]]; then
            fixes="${fixes}- ${commit}\n"
        else
            other="${other}- ${commit}\n"
        fi
    done < <(git log ${from_tag}..HEAD --pretty=format:"%s" --no-merges)

    if [[ -n $features ]]; then
        echo "### Features"
        echo -e "$features"
    fi

    if [[ -n $fixes ]]; then
        echo "### Bug Fixes"
        echo -e "$fixes"
    fi

    if [[ -n $other ]]; then
        echo "### Other Changes"
        echo -e "$other"
    fi

    echo ""
    echo "**Full Changelog**: https://github.com/${REPO_ORG}/${REPO_NAME}/compare/${from_tag}...${to_tag}"
}

# Main release process
main() {
    echo ""
    echo "========================================="
    echo "   todobi Release Pipeline"
    echo "========================================="
    echo ""

    # Parse arguments
    BUMP_TYPE=${1:-patch}
    CUSTOM_MESSAGE=${2:-""}

    if [[ $BUMP_TYPE != "major" && $BUMP_TYPE != "minor" && $BUMP_TYPE != "patch" ]]; then
        print_error "Invalid bump type. Use: major, minor, or patch"
    fi

    # Step 1: Commit any pending changes
    print_step "Checking for changes..."
    if [[ -n $(git status -s) ]]; then
        print_warning "Uncommitted changes found. Adding and committing..."

        git add -A

        if [[ -n $CUSTOM_MESSAGE ]]; then
            git commit -m "$CUSTOM_MESSAGE"
        else
            # Generate commit message from changes
            COMMIT_MSG="Release prep: $(git diff --cached --name-only | head -3 | xargs basename | paste -sd ', ' -)"
            git commit -m "$COMMIT_MSG"
        fi
        print_success "Changes committed"
    else
        print_success "No uncommitted changes"
    fi

    # Step 2: Get version info
    CURRENT_VERSION=$(get_current_version)
    NEW_VERSION=$(get_next_version "$CURRENT_VERSION" "$BUMP_TYPE")

    echo ""
    print_step "Current version: ${CURRENT_VERSION}"
    print_step "New version: ${NEW_VERSION}"
    echo ""

    # Step 3: Update version in main.go (if version constant exists)
    if grep -q "Version.*=" main.go 2>/dev/null; then
        print_step "Updating version in main.go..."
        sed -i '' "s/Version.*=.*/Version = \"${NEW_VERSION#v}\"/" main.go
        git add main.go
        git commit -m "Bump version to ${NEW_VERSION}" || true
        print_success "Version updated in code"
    fi

    # Step 4: Build binary to ensure it compiles
    print_step "Building todobi binary..."
    go build -o todobi
    print_success "Build successful"

    # Step 5: Run tests if they exist
    if ls *_test.go 1> /dev/null 2>&1; then
        print_step "Running tests..."
        go test ./...
        print_success "Tests passed"
    fi

    # Step 6: Create and push tag
    print_step "Creating git tag ${NEW_VERSION}..."
    git tag -a "$NEW_VERSION" -m "Release $NEW_VERSION"

    print_step "Pushing to GitHub..."
    git push origin master
    git push origin "$NEW_VERSION"
    print_success "Tag and code pushed to GitHub"

    # Step 7: Calculate SHA256 for new tarball
    print_step "Waiting for GitHub to process tag..."
    sleep 5

    TARBALL_URL="https://github.com/${REPO_ORG}/${REPO_NAME}/archive/${NEW_VERSION}.tar.gz"
    print_step "Calculating SHA256 for ${TARBALL_URL}..."

    SHA256=$(curl -sL "$TARBALL_URL" | shasum -a 256 | cut -d' ' -f1)

    if [[ -z $SHA256 ]]; then
        print_error "Failed to download tarball or calculate SHA256"
    fi

    print_success "SHA256: ${SHA256}"

    # Step 8: Update formula with new URL and SHA256
    print_step "Updating Homebrew formula..."

    if [[ ! -f $FORMULA_PATH ]]; then
        print_error "Formula not found: $FORMULA_PATH"
    fi

    # Update URL, SHA256, and version in the tap formula
    sed -i '' "s|url \".*\"|url \"${TARBALL_URL}\"|" "$FORMULA_PATH"
    sed -i '' "s|sha256 \".*\"|sha256 \"${SHA256}\"|" "$FORMULA_PATH"
    sed -i '' "s|version \".*\"|version \"${NEW_VERSION#v}\"|" "$FORMULA_PATH"

    # Update test assertion
    sed -i '' "s|assert_match \"todobi v.*\", shell_output(\"#{bin}/todobi --version\")|assert_match \"todobi v${NEW_VERSION#v}\", shell_output(\"#{bin}/todobi --version\")|" "$FORMULA_PATH"

    # Commit and push to homebrew tap
    cd "$HOMEBREW_TAP_PATH"
    git add Formula/todobi.rb
    git commit -m "Release todobi ${NEW_VERSION}"
    git push
    cd - > /dev/null

    print_success "Homebrew formula updated in tap"

    # Step 9: Generate changelog
    print_step "Generating changelog..."
    CHANGELOG=$(generate_changelog "$CURRENT_VERSION" "$NEW_VERSION")

    # Step 10: Create GitHub release
    print_step "Creating GitHub release..."

    # Build release binaries
    print_step "Building release binaries..."
    GOOS=darwin GOARCH=arm64 go build -o todobi-darwin-arm64
    GOOS=darwin GOARCH=amd64 go build -o todobi-darwin-amd64
    GOOS=linux GOARCH=amd64 go build -o todobi-linux-amd64

    # Create tarballs
    tar -czf todobi-${NEW_VERSION}-darwin-arm64.tar.gz todobi-darwin-arm64
    tar -czf todobi-${NEW_VERSION}-darwin-amd64.tar.gz todobi-darwin-amd64
    tar -czf todobi-${NEW_VERSION}-linux-amd64.tar.gz todobi-linux-amd64

    gh release create "$NEW_VERSION" \
        --repo "${REPO_ORG}/${REPO_NAME}" \
        --title "Release ${NEW_VERSION}" \
        --notes "$CHANGELOG" \
        todobi-${NEW_VERSION}-darwin-arm64.tar.gz \
        todobi-${NEW_VERSION}-darwin-amd64.tar.gz \
        todobi-${NEW_VERSION}-linux-amd64.tar.gz \
        --latest

    # Cleanup binaries
    rm -f todobi-darwin-* todobi-linux-* todobi-*.tar.gz

    print_success "GitHub release created with binaries"

    echo ""
    echo "========================================="
    echo -e "${GREEN}✓ Release ${NEW_VERSION} complete!${NC}"
    echo "========================================="
    echo ""
    echo "Users can now install/upgrade with:"
    echo "  brew install ${REPO_ORG}/homebrew-tap/todobi"
    echo "  brew upgrade todobi"
    echo ""
    echo "Or download binaries from:"
    echo "  https://github.com/${REPO_ORG}/${REPO_NAME}/releases/tag/${NEW_VERSION}"
    echo ""
}

# Run main function
main "$@"
