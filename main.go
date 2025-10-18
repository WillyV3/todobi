package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	configFileName   = ".todobi.conf"
	minWidth         = 40
	minHeight        = 10
	contentPadding   = 2
	helpHeight       = 3
	headerHeight     = 2
)

// Priority levels
type Priority int

const (
	P0Critical Priority = iota
	P1High
	P2Medium
	P3Low
)

func (p Priority) String() string {
	switch p {
	case P0Critical:
		return "P0 CRITICAL"
	case P1High:
		return "P1 HIGH"
	case P2Medium:
		return "P2 MEDIUM"
	case P3Low:
		return "P3 LOW"
	default:
		return "TASK"
	}
}

func (p Priority) Color() string {
	switch p {
	case P0Critical:
		return "#d73a4a"
	case P1High:
		return "#fb8500"
	case P2Medium:
		return "#ffc107"
	case P3Low:
		return "#4caf50"
	default:
		return "#666666"
	}
}

// Task represents a todo item
type Task struct {
	ID          string    `json:"id"`
	Content     string    `json:"content"`
	Description string    `json:"description,omitempty"`
	Priority    Priority  `json:"priority"`
	Done        bool      `json:"done"`
	CreatedAt   time.Time `json:"created_at"`
}

// Config stores all tasks
type Config struct {
	Tasks      []Task    `json:"tasks"`
	LastUpdate time.Time `json:"last_update"`
	Version    string    `json:"version"`
}

// Model is the Bubble Tea model
type model struct {
	config      *Config
	width       int
	height      int
	cursor      int
	ready       bool
	showHelp    bool
	statusMsg   string
	statusUntil time.Time
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		cfg = defaultConfig()
		if err := saveConfig(cfg); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	}

	m := model{
		config: cfg,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// Config operations
func loadConfig() (*Config, error) {
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

func saveConfig(cfg *Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	cfg.LastUpdate = time.Now()
	path := filepath.Join(home, configFileName)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func defaultConfig() *Config {
	return &Config{
		Version: "1.2.0",
		Tasks: []Task{
			{
				ID:        "1",
				Content:   "Press 'x' to toggle completion",
				Priority:  P1High,
				CreatedAt: time.Now(),
			},
			{
				ID:        "2",
				Content:   "Press 'r' to reload config",
				Priority:  P2Medium,
				CreatedAt: time.Now(),
			},
			{
				ID:        "3",
				Content:   "Press '?' for help",
				Priority:  P3Low,
				CreatedAt: time.Now(),
			},
		},
	}
}

// Bubble Tea interface
func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleResize(msg)
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}
	return m, nil
}

func (m model) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = max(msg.Width, minWidth)
	m.height = max(msg.Height, minHeight)
	m.ready = true
	return m, nil
}

func (m model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		saveConfig(m.config)
		return m, tea.Quit

	case "?":
		m.showHelp = !m.showHelp
		return m, nil

	case "r":
		cfg, err := loadConfig()
		if err != nil {
			m.setStatus("Error reloading config")
		} else {
			m.config = cfg
			m.setStatus("Config reloaded")
		}
		return m, nil

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "down", "j":
		if m.cursor < len(m.config.Tasks)-1 {
			m.cursor++
		}
		return m, nil

	case "x", " ":
		if m.cursor < len(m.config.Tasks) {
			m.config.Tasks[m.cursor].Done = !m.config.Tasks[m.cursor].Done
			saveConfig(m.config)
			if m.config.Tasks[m.cursor].Done {
				m.setStatus("Task completed")
			} else {
				m.setStatus("Task reopened")
			}
		}
		return m, nil

	case "d":
		if m.cursor < len(m.config.Tasks) {
			m.config.Tasks = append(m.config.Tasks[:m.cursor], m.config.Tasks[m.cursor+1:]...)
			if m.cursor >= len(m.config.Tasks) && m.cursor > 0 {
				m.cursor--
			}
			saveConfig(m.config)
			m.setStatus("Task deleted")
		}
		return m, nil
	}

	return m, nil
}

func (m *model) setStatus(msg string) {
	m.statusMsg = msg
	m.statusUntil = time.Now().Add(2 * time.Second)
}

func (m model) View() string {
	if !m.ready {
		return "\nInitializing..."
	}

	var output strings.Builder

	output.WriteString(m.renderHeader())
	output.WriteString(m.renderTasks())
	output.WriteString(m.renderFooter())

	return output.String()
}

func (m model) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#4ec9b0")).
		Width(m.width - contentPadding).
		Align(lipgloss.Center)

	stats := m.getStats()
	subtitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999")).
		Width(m.width - contentPadding).
		Align(lipgloss.Center)

	return titleStyle.Render("TODO") + "\n" +
		subtitle.Render(stats) + "\n\n"
}

func (m model) renderTasks() string {
	if len(m.config.Tasks) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666")).
			Italic(true).
			Width(m.width - contentPadding).
			Align(lipgloss.Center)
		return emptyStyle.Render("No tasks. Config: ~/.todobi.conf") + "\n\n"
	}

	availableHeight := m.height - headerHeight - helpHeight - contentPadding
	var output strings.Builder

	for i, task := range m.config.Tasks {
		if i >= availableHeight {
			remaining := len(m.config.Tasks) - i
			moreStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#666")).
				Italic(true)
			output.WriteString(moreStyle.Render(fmt.Sprintf("... %d more tasks\n", remaining)))
			break
		}

		line := m.renderTask(task, i == m.cursor)
		output.WriteString(line + "\n")
	}

	output.WriteString("\n")
	return output.String()
}

func (m model) renderTask(task Task, selected bool) string {
	cursor := "  "
	if selected {
		cursor = "> "
	}

	checkbox := "[ ]"
	if task.Done {
		checkbox = "[x]"
	}

	priorityStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(task.Priority.Color())).
		Bold(true)

	priority := priorityStyle.Render(task.Priority.String())

	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#d4d4d4"))

	if task.Done {
		contentStyle = contentStyle.
			Foreground(lipgloss.Color("#666")).
			Strikethrough(true)
	}

	if selected {
		contentStyle = contentStyle.Bold(true)
	}

	maxContentWidth := m.width - 20 // cursor + checkbox + priority + padding
	content := task.Content
	if len(content) > maxContentWidth {
		content = content[:maxContentWidth-3] + "..."
	}

	return fmt.Sprintf("%s%s %-12s %s",
		cursor,
		checkbox,
		priority,
		contentStyle.Render(content),
	)
}

func (m model) renderFooter() string {
	if m.showHelp {
		return m.renderHelp()
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666"))

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4ec9b0"))

	status := ""
	if time.Now().Before(m.statusUntil) {
		status = statusStyle.Render(m.statusMsg) + " "
	}

	return status + helpStyle.Render("? help | x toggle | d delete | q quit")
}

func (m model) renderHelp() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#666")).
		Padding(0, 1).
		Width(m.width - contentPadding)

	help := []string{
		"Navigation:",
		"  up/k      Move up",
		"  down/j    Move down",
		"",
		"Actions:",
		"  x/space   Toggle task completion",
		"  d         Delete task",
		"  r         Reload config",
		"",
		"Other:",
		"  ?         Toggle help",
		"  q/ctrl+c  Quit",
	}

	return helpStyle.Render(strings.Join(help, "\n"))
}

func (m model) getStats() string {
	total := len(m.config.Tasks)
	done := 0
	for _, task := range m.config.Tasks {
		if task.Done {
			done++
		}
	}

	percent := 0
	if total > 0 {
		percent = (done * 100) / total
	}

	return fmt.Sprintf("%d/%d complete (%d%%)", done, total, percent)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
