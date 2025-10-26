# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`todobi` (todo-bionic) is a terminal-based task manager built with Bubble Tea (charmbracelet/bubbletea). It's a single-file Go application that provides a vim-keybinding-based interface for managing tasks across machines using GitHub sync.

**Key architectural decision**: The entire TUI application is contained in `main.go` (~2600 lines) - this is intentional for simplicity and should remain a single file.

## Build and Development Commands

```bash
# Build binary
go build -o todobi

# Run directly
./todobi

# Test build (creates todobi-test binary)
go build -o todobi-test

# Seed with weekend task examples
./todobi seed

# Pull config from GitHub (initial setup on new machine)
./todobi --pull

# Run tests (if any exist)
go test ./...
```

## Release Process

**CRITICAL**: Always use the automated release script for releases.

```bash
# Release with version bump
./scripts/release.sh [patch|minor|major] ["optional commit message"]

# Examples
./scripts/release.sh patch "Fix keybinding bug"
./scripts/release.sh minor "Add category filtering"
./scripts/release.sh major "Breaking: new config format"
```

**What the release script does:**
1. Commits any uncommitted changes
2. Calculates next version from git tags
3. Updates version in code if `Version` constant exists
4. Runs build and tests
5. Creates and pushes git tag
6. Downloads tarball and calculates SHA256
7. Updates Homebrew formula in `~/homebrew-tap`
8. Builds multi-platform binaries (darwin-arm64, darwin-amd64, linux-amd64)
9. Creates GitHub release with auto-generated changelog

**Do NOT:**
- Manually create tags with `git tag`
- Manually update the Homebrew formula
- Use `gh release create` manually

## Architecture

### State Management (Bubble Tea Pattern)

The application uses Bubble Tea's Elm Architecture with a single `model` struct that contains:

- `config`: Loaded from `~/.todobi.conf` (JSON format)
- `mode`: Current view state (listView, taskFormView, categoryFormView, etc.)
- `list`, `completedList`, `categoryList`: Bubble Tea list components
- `taskInputs`, `categoryInput`, `notesTextarea`: Input components
- GitHub sync state: `syncInProgress`, `pullInProgress`, `remoteConfig`

### View Modes (main.go:155-169)

```go
const (
    listView viewMode = iota
    categoryFormView
    taskFormView
    completedView
    deleteConfirmView
    categoryListView
    syncConfirmView
    pullConfirmView
    editTaskView
    taskDetailView
    firstRunView
)
```

The app switches between these modes - each has dedicated render and handler functions.

### Data Model

- **Task** (main.go:69-78): Core task with ID, Content, CategoryID, Priority (P0-P3), Done status, timestamps, and Notes
- **Category** (main.go:141-144): Organizes tasks by ID and Name
- **Config** (main.go:147-153): Persisted to `~/.todobi.conf`, contains all tasks, categories, and GitHub setup state

### GitHub Sync Architecture

**Two sync directions:**
1. **Push (G key)**: `syncToGitHubCmd()` → clones/creates `todobi-sync` private repo → copies config → commits and pushes
2. **Pull (g key)**: `pullFromGitHubCmd()` → clones repo → reads remote config → detects conflicts → shows merge UI

**Conflict resolution** (main.go:989-1027): When local and remote both have changes:
- L: Keep local (discard remote)
- R: Use remote (overwrite local)
- M: Merge (combines tasks by ID, newer wins)

**First-run setup** (main.go:1574-1615): Guides new users through GitHub setup:
1. Welcome screen
2. "Do you have existing repo?" prompt
3. Pull or create repo flow
4. Mark `GitHubSetupComplete` to prevent re-showing

### Category Tabs (main.go:231-297)

Categories are displayed as tabs at the top of the list view with wrapping support. The "All" tab shows all tasks; selecting a category filters tasks to only that category. Tab navigation uses tab/shift+tab keys.

### Task Detail View with Notes

Pressing `enter` or `i` on a task opens detail view (main.go:2331-2441) which shows:
- Task metadata in bordered box (content, category, priority, age, status)
- Multi-line notes textarea (using bubbles/textarea)
- Auto-save prompt when exiting with unsaved notes

## Config File Format

Location: `~/.todobi.conf`

```json
{
  "categories": [
    {"id": "work", "name": "Work"},
    {"id": "personal", "name": "Personal"}
  ],
  "tasks": [
    {
      "id": "1",
      "content": "Task description",
      "category_id": "work",
      "priority": 1,
      "done": false,
      "created_at": "2025-10-17T...",
      "completed_at": "2025-10-17T...",
      "notes": "Optional notes"
    }
  ],
  "last_update": "2025-10-17T...",
  "version": "1.3.0",
  "github_setup_complete": true
}
```

## Keybindings

### List View
- `j`/`k` or `↑`/`↓`: Navigate
- `tab`/`shift+tab`: Switch category tabs
- `x` or `space`: Toggle task completion
- `enter` or `i`: View task details
- `d`: Delete task (with confirmation)
- `T`: New task form
- `C`: New category form
- `c`: Manage categories
- `v`: Toggle completed tasks view
- `G`: Sync to GitHub (push)
- `g`: Pull from GitHub
- `r`: Reload config from disk
- `?`: Toggle help
- `q` or `ctrl+c`: Quit

### Task Detail View
- `ctrl+e`: Edit task properties
- `ctrl+s`: Save notes manually
- `esc`: Save notes and return (prompts if unsaved)

### Form Views
- `↑`/`↓` or `tab`: Navigate fields
- `enter`: Submit form
- `esc`: Cancel

## Common Development Patterns

### Adding a New View Mode

1. Add constant to `viewMode` enum (main.go:155)
2. Create `render{ViewName}()` function
3. Create `handle{ViewName}(msg tea.KeyMsg)` handler
4. Add case to `View()` switch (main.go:1573)
5. Add case to `Update()` switch (main.go:627)

### Modifying Task/Category Data

Always use `m.saveConfigAndMarkChanged()` after modifying `m.config` - this:
1. Saves to `~/.todobi.conf`
2. Sets `m.configChanged = true` (shows "Unsynced changes" in footer)

### GitHub CLI Requirements

The app requires `gh` CLI (https://cli.github.com) for GitHub operations. Commands check:
1. `gh --version` - CLI installed
2. `gh auth status` - User authenticated
3. Uses `gh auth git-credential` as credential helper for HTTPS git operations

## Testing First-Run Experience

```bash
# Test first-run flow
rm ~/.todobi.conf
./todobi

# Or manually edit config
# Set "github_setup_complete": false in ~/.todobi.conf
```

The first-run flow only triggers when `github_setup_complete` is false or missing in the config.

## Repository Structure

```
todobi/
├── main.go                    # Entire TUI application (~2600 lines)
├── scripts/
│   └── release.sh            # Automated release pipeline
├── test_first_run.sh         # Test script for first-run detection
├── go.mod                     # Go 1.25.3, Bubble Tea dependencies
├── README.md                  # User documentation
└── .git/                      # Git repo (github.com/WillyV3/todobi)
```

## Dependencies

- `github.com/charmbracelet/bubbletea` - TUI framework (Elm Architecture)
- `github.com/charmbracelet/bubbles` - Pre-built components (list, textarea, textinput, spinner)
- `github.com/charmbracelet/lipgloss` - Styling and layout

All use the Charm ecosystem for consistent terminal UI development.
