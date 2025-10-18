package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// SeedCommand initializes config with weekend tasks
func SeedCommand() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(home, configFileName)

	// Check if config exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Print("Config file already exists. Overwrite? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	cfg := SeedWeekendTasks()
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ Created config file: %s\n", configPath)
	fmt.Printf("  Total tasks: %d\n", len(cfg.GetAllTasks()))
	fmt.Printf("  Groups: %d\n", len(cfg.Groups))
	fmt.Println("\nRun 'todobi' to view your tasks!")

	return nil
}
