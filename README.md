# todobi

> **todo**-**bi**onic: A terminal task manager built with [Bubble Tea](https://github.com/charmbracelet/bubbletea)

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue.svg)


Features

Clean, minimal interface

Sync with github for persistence across machines!

Respects terminal size (works at any dimension)

Vim keybindings (j/k navigation)

JSON config file

No emoji clutter

Single 436-line file

Installation


go build -o todobi

Usage: todobi

Config file: ~/.todobi.conf


Keybindings
Key	Action
up/k	Move up
down/j	Move down
x/space	Toggle task completion
d	Delete task
r	Reload config
?	Toggle help
q/ctrl+c	Quit

Config Format

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

Priority Levels

0 - P0 CRITICAL (red)
1 - P1 HIGH (orange)
2 - P2 MEDIUM (yellow)
3 - P3 LOW (green)

## Contributing

PRs welcome! This is a weekend project but I'm happy to review contributions.

## License

MIT © 2025 WillyV3

## Acknowledgments

Built with the amazing [Charm](https://charm.sh) libraries. 💜
