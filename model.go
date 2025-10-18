package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

type viewMode int

const (
	dashboardView viewMode = iota
	listView
	addTaskView
)

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Toggle   key.Binding
	Delete   key.Binding
	Add      key.Binding
	View     key.Binding
	Help     key.Binding
	Quit     key.Binding
	Refresh  key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter/space", "toggle task"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d", "x"),
		key.WithHelp("d/x", "delete task"),
	),
	Add: key.NewBinding(
		key.WithKeys("a", "n"),
		key.WithHelp("a/n", "add task"),
	),
	View: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch view"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh/reload"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

type model struct {
	config       *Config
	mode         viewMode
	cursor       int
	width        int
	height       int
	progress     progress.Model
	showHelp     bool
	statusMsg    string
	statusExpire time.Time
}

type tickMsg time.Time

func NewModel(cfg *Config) model {
	prog := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(50),
	)

	return model{
		config:   cfg,
		mode:     dashboardView,
		progress: prog,
	}
}

func (m model) Init() tea.Cmd {
	return tick()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			m.config.Save()
			return m, tea.Quit

		case key.Matches(msg, keys.View):
			if m.mode == dashboardView {
				m.mode = listView
			} else {
				m.mode = dashboardView
			}
			return m, nil

		case key.Matches(msg, keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, keys.Refresh):
			cfg, err := LoadConfig()
			if err == nil {
				m.config = cfg
				m.setStatus("Configuration reloaded!")
			} else {
				m.setStatus(fmt.Sprintf("Error: %v", err))
			}
			return m, nil

		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case key.Matches(msg, keys.Down):
			tasks := m.getVisibleTasks()
			if m.cursor < len(tasks)-1 {
				m.cursor++
			}
			return m, nil

		case key.Matches(msg, keys.Toggle):
			if m.mode == listView {
				m.toggleTask()
				m.config.Save()
			}
			return m, nil

		case key.Matches(msg, keys.Delete):
			if m.mode == listView {
				m.deleteTask()
				m.config.Save()
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = msg.Width - 40
		return m, nil

	case tickMsg:
		return m, tick()
	}

	return m, nil
}

func (m *model) getVisibleTasks() []Task {
	var tasks []Task
	for _, group := range m.config.Groups {
		for _, task := range group.Tasks {
			if !task.Completed {
				tasks = append(tasks, task)
			}
		}
	}
	return tasks
}

func (m *model) toggleTask() {
	tasks := m.getVisibleTasks()
	if m.cursor >= len(tasks) {
		return
	}

	targetTask := tasks[m.cursor]

	// Find and toggle the task
	for gi, group := range m.config.Groups {
		for ti, task := range group.Tasks {
			if task.ID == targetTask.ID {
				m.config.Groups[gi].Tasks[ti].Completed = !task.Completed
				if m.config.Groups[gi].Tasks[ti].Completed {
					m.config.Groups[gi].Tasks[ti].CompletedAt = time.Now()
					m.setStatus("✓ Task completed!")
				} else {
					m.setStatus("Task reopened")
				}
				return
			}
		}
	}
}

func (m *model) deleteTask() {
	tasks := m.getVisibleTasks()
	if m.cursor >= len(tasks) {
		return
	}

	targetTask := tasks[m.cursor]

	for gi, group := range m.config.Groups {
		for ti, task := range group.Tasks {
			if task.ID == targetTask.ID {
				m.config.Groups[gi].Tasks = append(
					m.config.Groups[gi].Tasks[:ti],
					m.config.Groups[gi].Tasks[ti+1:]...,
				)
				m.setStatus("Task deleted")
				if m.cursor > 0 {
					m.cursor--
				}
				return
			}
		}
	}
}

func (m *model) setStatus(msg string) {
	m.statusMsg = msg
	m.statusExpire = time.Now().Add(3 * time.Second)
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	switch m.mode {
	case dashboardView:
		return m.dashboardView()
	case listView:
		return m.listView()
	default:
		return m.dashboardView()
	}
}

func (m model) dashboardView() string {
	var b strings.Builder

	// Header
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#4ec9b0")).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("#4ec9b0")).
		Width(m.width - 4).
		Align(lipgloss.Center)

	b.WriteString(titleStyle.Render("🚀 TODOBI - Task Dashboard"))
	b.WriteString("\n\n")

	// Progress bar
	total := len(m.config.GetAllTasks())
	completed := m.config.GetCompletedCount()
	percent := float64(0)
	if total > 0 {
		percent = float64(completed) / float64(total)
	}

	progressLabel := fmt.Sprintf("Progress: %d/%d completed (%d%%)", completed, total, int(percent*100))
	b.WriteString(lipgloss.NewStyle().Bold(true).Render(progressLabel))
	b.WriteString("\n")
	b.WriteString(m.progress.ViewAs(percent))
	b.WriteString("\n\n")

	// Task groups
	for _, group := range m.config.Groups {
		pendingCount := 0
		for _, task := range group.Tasks {
			if !task.Completed {
				pendingCount++
			}
		}

		if pendingCount == 0 {
			continue
		}

		// Group header
		headerStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(group.Priority.Color())).
			MarginTop(1)

		b.WriteString(headerStyle.Render(group.Priority.String()))
		b.WriteString(fmt.Sprintf(" (%d pending)\n", pendingCount))

		// Tasks
		count := 0
		for _, task := range group.Tasks {
			if task.Completed {
				continue
			}
			count++
			if count > 5 {
				b.WriteString(fmt.Sprintf("  ... and %d more\n", pendingCount-5))
				break
			}

			taskStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#d4d4d4")).
				MarginLeft(2)

			checkbox := "☐"
			if task.Completed {
				checkbox = "☑"
			}

			b.WriteString(taskStyle.Render(fmt.Sprintf("%s %s\n", checkbox, task.Content)))
		}
	}

	// Status message
	if time.Now().Before(m.statusExpire) {
		b.WriteString("\n")
		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ec9b0")).
			Italic(true)
		b.WriteString(statusStyle.Render(m.statusMsg))
	}

	// Help footer
	b.WriteString("\n\n")
	b.WriteString(m.helpView())

	return lipgloss.NewStyle().
		Padding(1, 2).
		Render(b.String())
}

func (m model) listView() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#569cd6")).
		Width(m.width - 4).
		Align(lipgloss.Center)

	b.WriteString(titleStyle.Render("📋 All Tasks"))
	b.WriteString("\n\n")

	tasks := m.getVisibleTasks()
	if len(tasks) == 0 {
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666")).
			Italic(true).
			Render("No pending tasks! 🎉"))
		b.WriteString("\n\n")
		b.WriteString(m.helpView())
		return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	}

	for i, task := range tasks {
		cursor := "  "
		if i == m.cursor {
			cursor = "→ "
		}

		checkbox := "☐"
		if task.Completed {
			checkbox = "☑"
		}

		taskStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#d4d4d4"))
		if i == m.cursor {
			taskStyle = taskStyle.Bold(true).Foreground(lipgloss.Color("#4ec9b0"))
		}

		line := fmt.Sprintf("%s%s %s", cursor, checkbox, task.Content)
		b.WriteString(taskStyle.Render(line))
		b.WriteString("\n")

		if task.Description != "" && i == m.cursor {
			descStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#999")).
				Italic(true).
				MarginLeft(5)
			b.WriteString(descStyle.Render(task.Description))
			b.WriteString("\n")
		}
	}

	// Status message
	if time.Now().Before(m.statusExpire) {
		b.WriteString("\n")
		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ec9b0")).
			Italic(true)
		b.WriteString(statusStyle.Render(m.statusMsg))
	}

	b.WriteString("\n\n")
	b.WriteString(m.helpView())

	return lipgloss.NewStyle().
		Padding(1, 2).
		Render(b.String())
}

func (m model) helpView() string {
	if !m.showHelp {
		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666"))
		return helpStyle.Render("Press ? for help • Tab to switch views • q to quit")
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999")).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#666")).
		Padding(0, 1)

	help := []string{
		"Navigation:",
		"  ↑/k      - Move up",
		"  ↓/j      - Move down",
		"  tab      - Switch view",
		"",
		"Actions:",
		"  enter/␣  - Toggle task",
		"  d/x      - Delete task",
		"  a/n      - Add task (coming soon)",
		"  r        - Reload config",
		"",
		"Other:",
		"  ?        - Toggle help",
		"  q/ctrl+c - Quit",
	}

	return helpStyle.Render(strings.Join(help, "\n"))
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Helper to generate task ID
func generateTaskID() string {
	return uuid.New().String()[:8]
}
