# todobi

> **todo**-**bi**onic: A beautiful, lightning-fast terminal task manager built with [Bubble Tea](https://github.com/charmbracelet/bubbletea)

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)

## Features

- 🎨 **Beautiful TUI** - Gorgeous terminal interface with progress bars and color-coded priorities
- ⚡ **Lightning Fast** - Native Go performance, instant startup
- 📊 **Dashboard View** - See your progress at a glance with visual progress bars
- 📋 **List View** - Focus mode for working through tasks one-by-one
- 🎯 **Priority System** - P0 (Critical) through P3 (Low), plus Homelab and Dev categories
- 💾 **Simple Config** - JSON config file at `~/.todobi.conf`
- ⌨️  **Vim Keybindings** - Navigate with j/k or arrow keys
- 🔄 **Live Reload** - Press 'r' to reload config changes without restarting

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap WillyV3/homebrew-tap
brew install todobi
```

### From Source

```bash
go install github.com/WillyV3/todobi@latest
```

### Binary Download

Download from [releases page](https://github.com/WillyV3/todobi/releases)

## Quick Start

```bash
# Create sample weekend tasks config
todobi seed

# Launch the TUI
todobi
```

## Usage

### Commands

```bash
todobi              # Launch TUI interface
todobi seed         # Create sample config with weekend tasks
todobi version      # Show version
todobi help         # Show help
```

### Keyboard Controls

| Key | Action |
|-----|--------|
| `↑/k` | Move up |
| `↓/j` | Move down |
| `enter/space` | Toggle task completion |
| `d/x` | Delete task |
| `tab` | Switch between dashboard/list view |
| `r` | Reload config from disk |
| `?` | Toggle help |
| `q/ctrl+c` | Quit |

## Configuration

Tasks are stored in `~/.todobi.conf` as JSON. You can edit this file manually or through the TUI.

### Example Config

```json
{
  "version": "1.0.0",
  "last_update": "2025-10-17T21:00:00Z",
  "groups": [
    {
      "name": "Priority 0: Critical",
      "priority": 0,
      "tasks": [
        {
          "id": "abc123",
          "content": "Fix production bug",
          "description": "Users can't login",
          "priority": 0,
          "completed": false,
          "created_at": "2025-10-17T10:00:00Z",
          "tags": ["bug", "urgent"],
          "url": "https://github.com/org/repo/issues/123"
        }
      ]
    }
  ]
}
```

### Priority Levels

- `0` - 🎯 **P0 CRITICAL** - Revenue/growth blockers (Red)
- `1` - 🔥 **P1 HIGH** - Core functionality (Orange)
- `2` - ⚡ **P2 MEDIUM** - Polish & quality (Yellow)
- `3` - 📋 **P3 LOW** - Nice to have (Green)
- `4` - 🏠 **HOMELAB** - Homelab tasks (Blue)
- `5` - 🛠️  **DEVELOPMENT** - Dev environment tasks (Purple)

## Screenshots

### Dashboard View
```
╔════════════════════════════════════════════════════════════════════╗
║           🚀 TODOBI - Task Dashboard                               ║
╚════════════════════════════════════════════════════════════════════╝

Progress: 5/18 completed (27%)
████████░░░░░░░░░░░░░░░░░░░░░░░░░░

🎯 P0 CRITICAL (4 pending)
  ☐ #1 - Consultation Booking Page UI
  ☐ #2 - Consultation Scheduling Calendar UI
  ☐ #3 - Subscription Upgrade Modal
  ☐ #4 - Subscription Tier Badge in Header
```

### List View
```
              📋 All Tasks

→ ☐ #1 - Consultation Booking Page UI
  ☐ #2 - Consultation Scheduling Calendar UI
  ☐ #3 - Subscription Upgrade Modal
  ☐ Fix existing homelab issues
    Diagnose and repair current homelab infrastructure problems
  ☐ Implement distributed file sharing
```

## Why todobi?

I built todobi because I wanted:
- **Fast** - No Electron, no web frameworks, just pure Go speed
- **Beautiful** - Terminal UIs can be gorgeous with the right libraries
- **Simple** - One JSON file, no database, no complexity
- **Flexible** - Edit tasks in TUI or directly in the config file
- **Homelab-friendly** - Built for managing infrastructure and dev work

## Tech Stack

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling
- [UUID](https://github.com/google/uuid) - Task IDs

## Roadmap

- [ ] Add task editing in TUI (currently must edit config file)
- [ ] Add task creation in TUI
- [ ] Search/filter tasks
- [ ] Task due dates
- [ ] Recurring tasks
- [ ] Export to markdown
- [ ] GitHub Issues integration
- [ ] Multiple config file support
- [ ] Task templates

## Contributing

PRs welcome! This is a weekend project but I'm happy to review contributions.

## License

MIT © 2025 WillyV3

## Acknowledgments

Built with the amazing [Charm](https://charm.sh) libraries. 💜
