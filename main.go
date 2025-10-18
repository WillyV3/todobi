package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Handle commands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "seed":
			if err := SeedCommand(); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "version", "-v", "--version":
			fmt.Println("todobi v1.0.0")
			return
		case "help", "-h", "--help":
			printHelp()
			return
		default:
			fmt.Printf("Unknown command: %s\n", os.Args[1])
			printHelp()
			os.Exit(1)
		}
	}

	// Load or create config
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Printf("No config found. Run 'todobi seed' to create one.\n")
		fmt.Printf("Or create an empty config now? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response == "y" || response == "Y" {
			cfg = DefaultConfig()
			if err := cfg.Save(); err != nil {
				fmt.Printf("Error saving config: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✓ Created empty config")
		} else {
			os.Exit(0)
		}
	}

	// Run TUI
	m := NewModel(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	help := `todobi - Beautiful terminal task manager

Usage:
  todobi              Launch TUI interface
  todobi seed         Create sample config with weekend tasks
  todobi version      Show version
  todobi help         Show this help

Config file location: ~/.todobi.conf

Controls (in TUI):
  ↑/k        Move up
  ↓/j        Move down
  enter/␣    Toggle task completion
  d/x        Delete task
  tab        Switch between dashboard/list view
  r          Reload config from disk
  ?          Toggle help
  q/ctrl+c   Quit

For more info: https://github.com/WillyV3/todobi
`
	fmt.Print(help)
}
