# todobi - Clean Version

A simple, professional terminal todo manager built with Bubble Tea.

## Features

- Clean, minimal interface
- Respects terminal size (works at any dimension)
- Vim keybindings (j/k navigation)
- JSON config file
- No emoji clutter
- Single 436-line file

## Installation

```bash
go build -o todobi
```

## Usage

```bash
./todobi
```

Config file: `~/.todobi.conf`

## Keybindings

| Key | Action |
|-----|--------|
| `up/k` | Move up |
| `down/j` | Move down |
| `x/space` | Toggle task completion |
| `d` | Delete task |
| `r` | Reload config |
| `?` | Toggle help |
| `q/ctrl+c` | Quit |

## Config Format

```json
{
  "tasks": [
    {
      "id": "1",
      "content": "Task description",
      "priority": 1,
      "done": false,
      "created_at": "2025-10-17T..."
    }
  ],
  "last_update": "2025-10-17T...",
  "version": "1.2.0"
}
```

### Priority Levels

- `0` - P0 CRITICAL (red)
- `1` - P1 HIGH (orange)
- `2` - P2 MEDIUM (yellow)
- `3` - P3 LOW (green)

## Code Standards Compliance

- Zero emoji
- Respects terminal size at all dimensions
- Clean Update/View/Init pattern
- Proper window size handling with min width/height guards
- Self-documenting function names
- Single file (436 lines)
- No hardcoded dimensions
- Proper lipgloss width calculations

## Architecture

The app follows Bubble Tea best practices:

1. **Model** - Stores state (config, dimensions, cursor)
2. **Init** - Returns nil (no initial commands needed)
3. **Update** - Handles WindowSizeMsg and KeyMsg with extracted handlers
4. **View** - Pure rendering with helper functions for header/tasks/footer

Window size handling:
- Always stores width/height from WindowSizeMsg
- Uses min guards (40 cols, 10 rows)
- Calculates available space dynamically
- Truncates content that exceeds terminal width
- Shows "..." for overflow tasks

## Why This Version?

The previous version (1.1.0) had:
- 5 files totaling 1200+ lines
- 26+ emoji violations
- Hardcoded progress bar width
- Complex nested Update() logic
- No minimum size handling

This version:
- 1 file, 436 lines
- Zero emoji
- Dynamic sizing everywhere
- Clean, flat logic
- Proper terminal size respect

## License

MIT
