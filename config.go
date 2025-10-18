package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const configFileName = ".todobi.conf"

type Priority int

const (
	P0Critical Priority = iota // Revenue/growth blocker
	P1High                     // Core functionality
	P2Medium                   // Polish & quality
	P3Low                      // Nice to have
	PHHomelab                  // Homelab tasks
	PDev                       // Development tasks
)

func (p Priority) String() string {
	switch p {
	case P0Critical:
		return "🎯 P0 CRITICAL"
	case P1High:
		return "🔥 P1 HIGH"
	case P2Medium:
		return "⚡ P2 MEDIUM"
	case P3Low:
		return "📋 P3 LOW"
	case PHHomelab:
		return "🏠 HOMELAB"
	case PDev:
		return "🛠️  DEVELOPMENT"
	default:
		return "📌 TASK"
	}
}

func (p Priority) Color() string {
	switch p {
	case P0Critical:
		return "#f44336" // Red
	case P1High:
		return "#ff9800" // Orange
	case P2Medium:
		return "#ffc107" // Yellow
	case P3Low:
		return "#4caf50" // Green
	case PHHomelab:
		return "#2196f3" // Blue
	case PDev:
		return "#9c27b0" // Purple
	default:
		return "#666666"
	}
}

type Task struct {
	ID          string    `json:"id"`
	Content     string    `json:"content"`
	Description string    `json:"description,omitempty"`
	Priority    Priority  `json:"priority"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	URL         string    `json:"url,omitempty"`
}

type TaskGroup struct {
	Name     string `json:"name"`
	Priority Priority `json:"priority"`
	Tasks    []Task `json:"tasks"`
}

type Config struct {
	Groups     []TaskGroup `json:"groups"`
	LastUpdate time.Time   `json:"last_update"`
	Version    string      `json:"version"`
}

func DefaultConfig() *Config {
	now := time.Now()
	return &Config{
		Version:    "1.0.0",
		LastUpdate: now,
		Groups: []TaskGroup{
			{
				Name:     "Priority 0: Critical",
				Priority: P0Critical,
				Tasks:    []Task{},
			},
			{
				Name:     "Priority 1: High",
				Priority: P1High,
				Tasks:    []Task{},
			},
			{
				Name:     "Homelab Infrastructure",
				Priority: PHHomelab,
				Tasks:    []Task{},
			},
			{
				Name:     "Development",
				Priority: PDev,
				Tasks:    []Task{},
			},
		},
	}
}

func LoadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(home, configFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	c.LastUpdate = time.Now()
	path := filepath.Join(home, configFileName)

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (c *Config) GetAllTasks() []Task {
	var tasks []Task
	for _, group := range c.Groups {
		tasks = append(tasks, group.Tasks...)
	}
	return tasks
}

func (c *Config) GetPendingCount() int {
	count := 0
	for _, task := range c.GetAllTasks() {
		if !task.Completed {
			count++
		}
	}
	return count
}

func (c *Config) GetCompletedCount() int {
	count := 0
	for _, task := range c.GetAllTasks() {
		if task.Completed {
			count++
		}
	}
	return count
}

func (c *Config) GetProgress() int {
	total := len(c.GetAllTasks())
	if total == 0 {
		return 0
	}
	completed := c.GetCompletedCount()
	return (completed * 100) / total
}
