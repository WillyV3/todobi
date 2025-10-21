# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

todobi is a terminal task manager built with Go and Bubble Tea (charmbracelet/bubbletea). The entire application is intentionally contained in a single file (`main.go`) as a design principle, emphasizing simplicity and maintainability.

## Build and Development Commands

```bash
# Standard build
go build -o todobi

# Run the application
./todobi

# Run with seeded sample data
./todobi seed

# Test first-run experience
./test_first_run.sh

# Build for specific platforms
GOOS=darwin GOARCH=arm64 go build -o todobi-darwin-arm64
GOOS=linux GOARCH=amd64 go build -o todobi-linux-amd64

# Update dependencies
go mod tidy
go mod download
```

## Release Process

Use the automated release script:

```bash
# Patch release (1.2.3 -> 1.2.4)
./scripts/release.sh patch

# Minor release (1.2.3 -> 1.3.0)
./scripts/release.sh minor "optional commit message"

# Major release (1.2.3 -> 2.0.0)
./scripts/release.sh major
```

The script handles: version bumping, changelog generation, git tagging, binary compilation for multiple platforms, GitHub release creation, and Homebrew formula updates.

## Architecture

### Single-File Design

All application code lives in `main.go` (~2,400 lines). This is intentional. When making changes, maintain this architecture by adding functions to `main.go` rather than creating new files.

### Bubble Tea MVC Pattern

The application follows standard Bubble Tea architecture:

- `model` struct - Application state
- `Init() tea.Cmd` - Initialization (main.go:474)
- `Update(msg tea.Msg) (tea.Model, tea.Cmd)` - Message/event handler (main.go:478)
- `View() string` - Rendering (main.go:1443)

### State Machine (View Modes)

The app uses a view mode system (11 different modes):

```go
const (
    listView           // Main task list
    categoryFormView   // Creating/editing category
    taskFormView       // Creating/editing task
    completedView      // Completed tasks
    deleteConfirmView  // Deletion confirmation
    categoryListView   // Category management
    syncConfirmView    // GitHub sync confirmation
    pullConfirmView    // GitHub pull confirmation
    editTaskView       // Task editing
    taskDetailView     // Task detail with notes
    firstRunView       // First-run setup
)
```

When adding new views, follow this pattern:
1. Add a new constant to the viewMode enum
2. Add rendering logic to `View()` function
3. Add state transitions in `Update()` function
4. Create a dedicated render function (e.g., `renderMyNewView()`)

### Data Models

Core types:

- `Config` - Root configuration with Categories, Tasks, GitHub setup state
- `Task` - Task with ID, Content, CategoryID, Priority (P0-P3), Done status, Notes
- `Category` - Simple ID/Name structure
- `Priority` - Enum (0=P0Critical, 1=P1High, 2=P2Medium, 3=P3Low)

### Configuration File

Location: `~/.todobi.conf`
Format: JSON with versioning support

When modifying config structure:
1. Update the `Config` struct in main.go
2. Consider backwards compatibility (old configs should still load)
3. Use `configChanged` flag to trigger saves
4. Call `saveConfig()` to persist changes

### GitHub Integration

The app supports cross-machine sync via GitHub:

- `syncToGitHubCmd()` - Push config to GitHub (main.go:983)
- `pullFromGitHubCmd()` - Pull config from GitHub (main.go:1108)
- `mergeConfigs()` - Merge local and remote configs (main.go:905)

First-run setup uses a dedicated state machine (firstRunStep) with steps: welcomeStep, hasRepoPromptStep, createRepoPromptStep, pullingStep, pushingStep, completeStep.

When working with GitHub features:
- GitHub operations run as tea.Cmd (asynchronous)
- Results return via syncResultMsg or pullResultMsg
- Always handle both success and error cases
- The app uses `gh` CLI for GitHub operations

### Terminal Responsiveness

The app must respect terminal dimensions at all sizes (minimum: 40 cols x 10 rows):

- Handle `tea.WindowSizeMsg` to update model.width/height
- Use lipgloss for all styling and layout
- Pass width/height to lipgloss.NewStyle().Width()/.Height()
- Test at various terminal sizes

### UI Components

Wrapped Bubble Tea components:
- `list.Model` - For tasks and categories (charmbracelet/bubbles/list)
- `textinput.Model` - For category and task input
- `textarea.Model` - For task notes editing
- `spinner.Model` - For async operations (sync/pull)

When updating lists (tasks, categories), call the corresponding update function:
- `updateLists()` - Refresh task lists after changes
- `updateCategoryList()` - Refresh category list

### Status Messages

Use `setStatus(msg string)` to show temporary status messages. They auto-clear after 3 seconds.

## Code Style

- Zero emoji in output (design principle)
- Vim keybindings (j/k for navigation)
- Self-documenting function names
- Clean separation between Update/View/Init
- Dynamic sizing throughout (never hardcode dimensions)

## Testing

No formal unit tests currently exist. The `test_first_run.sh` script tests the first-run experience by removing the config file and running the app.

When testing changes:
1. Build with `go build -o todobi`
2. Test at different terminal sizes (resize while running)
3. Test with empty config (delete `~/.todobi.conf`)
4. Test GitHub sync if modifying that feature

## Key Files

- `main.go` - All application code
- `go.mod` - Go module dependencies
- `.goreleaser.yml` - Multi-platform release configuration
- `scripts/release.sh` - Full release automation pipeline
- `~/.todobi.conf` - User config file (not in repo)
