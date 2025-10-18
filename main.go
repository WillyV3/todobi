package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	configFileName = ".todobi.conf"
	minWidth       = 40
	minHeight      = 10
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
		return "P0"
	case P1High:
		return "P1"
	case P2Medium:
		return "P2"
	case P3Low:
		return "P3"
	default:
		return "P1"
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
	CategoryID  string    `json:"category_id"`
	Priority    Priority  `json:"priority"`
	Done        bool      `json:"done"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

// Implement list.Item interface
func (t Task) Title() string {
	priorityStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Priority.Color())).
		Bold(true)

	checkbox := "[ ]"
	if t.Done {
		checkbox = "[x]"
	}

	return fmt.Sprintf("%s %-4s %s", checkbox, priorityStyle.Render(t.Priority.String()), t.Content)
}

func (t Task) Description() string {
	if t.Done {
		return fmt.Sprintf("Completed: %s", t.CompletedAt.Format("2006-01-02 15:04"))
	}
	return ""
}

func (t Task) FilterValue() string {
	return t.Content
}

// Category for organizing tasks
type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Config stores all tasks and categories
type Config struct {
	Categories []Category `json:"categories"`
	Tasks      []Task     `json:"tasks"`
	LastUpdate time.Time  `json:"last_update"`
	Version    string     `json:"version"`
}

type viewMode int

const (
	listView viewMode = iota
	categoryFormView
	taskFormView
	completedView
	deleteConfirmView
)

// Model is the Bubble Tea model
type model struct {
	config         *Config
	width          int
	height         int
	mode           viewMode
	prevMode       viewMode
	ready          bool
	statusMsg      string
	statusUntil    time.Time
	categoryInput  textinput.Model
	taskInputs     []textinput.Model
	formFocus      int
	list           list.Model
	completedList  list.Model
	taskToDelete   *Task
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
		config:        cfg,
		categoryInput: textinput.New(),
		taskInputs:    make([]textinput.Model, 2),
	}

	m.categoryInput.Placeholder = "Category name"
	m.categoryInput.CharLimit = 50

	m.taskInputs[0] = textinput.New()
	m.taskInputs[0].Placeholder = "Task content"
	m.taskInputs[0].CharLimit = 200

	m.taskInputs[1] = textinput.New()
	m.taskInputs[1].Placeholder = "Priority (0-3)"
	m.taskInputs[1].CharLimit = 1

	// Initialize lists
	m.list = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	m.list.Title = "Tasks"
	m.list.SetShowStatusBar(false)
	m.list.SetFilteringEnabled(false)

	m.completedList = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	m.completedList.Title = "Completed Tasks"
	m.completedList.SetShowStatusBar(false)
	m.completedList.SetFilteringEnabled(false)

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
		Version: "1.3.0",
		Categories: []Category{
			{ID: "work", Name: "Work"},
			{ID: "personal", Name: "Personal"},
		},
		Tasks: []Task{
			{
				ID:         "1",
				Content:    "Press 'C' to create category",
				CategoryID: "work",
				Priority:   P1High,
				CreatedAt:  time.Now(),
			},
			{
				ID:         "2",
				Content:    "Press 'T' to create task",
				CategoryID: "work",
				Priority:   P2Medium,
				CreatedAt:  time.Now(),
			},
			{
				ID:         "3",
				Content:    "Press 'v' to view completed tasks",
				CategoryID: "personal",
				Priority:   P3Low,
				CreatedAt:  time.Now(),
			},
		},
	}
}

// Bubble Tea interface
func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = max(msg.Width, minWidth)
		m.height = max(msg.Height, minHeight)

		listHeight := m.height - 6
		m.list.SetSize(m.width, listHeight)
		m.completedList.SetSize(m.width, listHeight)

		if !m.ready {
			m.ready = true
			m.updateLists()
		}
		return m, nil

	case tea.KeyMsg:
		// Form handling
		if m.mode == categoryFormView {
			return m.handleCategoryForm(msg)
		}
		if m.mode == taskFormView {
			return m.handleTaskForm(msg)
		}
		if m.mode == deleteConfirmView {
			return m.handleDeleteConfirm(msg)
		}

		// Main view handling
		switch msg.String() {
		case "q", "ctrl+c":
			saveConfig(m.config)
			return m, tea.Quit

		case "r":
			cfg, err := loadConfig()
			if err != nil {
				m.setStatus("Error reloading config")
			} else {
				m.config = cfg
				m.updateLists()
				m.setStatus("Config reloaded")
			}
			return m, nil

		case "v":
			if m.mode == completedView {
				m.mode = listView
			} else {
				m.prevMode = m.mode
				m.mode = completedView
			}
			return m, nil

		case "C":
			m.prevMode = m.mode
			m.mode = categoryFormView
			m.categoryInput.Focus()
			m.categoryInput.SetValue("")
			return m, textinput.Blink

		case "T":
			m.prevMode = m.mode
			m.mode = taskFormView
			m.formFocus = 0
			m.taskInputs[0].Focus()
			m.taskInputs[1].Blur()
			m.taskInputs[0].SetValue("")
			m.taskInputs[1].SetValue("1")
			return m, textinput.Blink

		case "x", " ":
			return m.toggleTask()

		case "d":
			return m.confirmDelete()
		}
	}

	// Update the active list
	if m.mode == completedView {
		m.completedList, cmd = m.completedList.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.mode == listView {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) updateLists() {
	// Update active tasks list
	var activeItems []list.Item
	for _, task := range m.config.Tasks {
		if !task.Done {
			activeItems = append(activeItems, task)
		}
	}
	m.list.SetItems(activeItems)

	// Update completed tasks list
	var completedItems []list.Item
	for _, task := range m.config.Tasks {
		if task.Done {
			completedItems = append(completedItems, task)
		}
	}
	m.completedList.SetItems(completedItems)
}

func (m model) toggleTask() (tea.Model, tea.Cmd) {
	var selectedTask Task
	found := false

	if m.mode == completedView {
		if item := m.completedList.SelectedItem(); item != nil {
			selectedTask = item.(Task)
			found = true
		}
	} else {
		if item := m.list.SelectedItem(); item != nil {
			selectedTask = item.(Task)
			found = true
		}
	}

	if !found {
		return m, nil
	}

	// Find and toggle the task in config
	for i := range m.config.Tasks {
		if m.config.Tasks[i].ID == selectedTask.ID {
			m.config.Tasks[i].Done = !m.config.Tasks[i].Done
			if m.config.Tasks[i].Done {
				m.config.Tasks[i].CompletedAt = time.Now()
				m.setStatus("Task completed")
			} else {
				m.config.Tasks[i].CompletedAt = time.Time{}
				m.setStatus("Task reopened")
			}
			break
		}
	}

	saveConfig(m.config)
	m.updateLists()
	return m, nil
}

func (m model) confirmDelete() (tea.Model, tea.Cmd) {
	var selectedTask Task
	found := false

	if m.mode == completedView {
		if item := m.completedList.SelectedItem(); item != nil {
			selectedTask = item.(Task)
			found = true
		}
	} else if m.mode == listView {
		if item := m.list.SelectedItem(); item != nil {
			selectedTask = item.(Task)
			found = true
		}
	}

	if !found {
		return m, nil
	}

	m.taskToDelete = &selectedTask
	m.prevMode = m.mode
	m.mode = deleteConfirmView
	return m, nil
}

func (m model) deleteTask() (tea.Model, tea.Cmd) {
	if m.taskToDelete == nil {
		return m, nil
	}

	// Find and delete the task
	for i := range m.config.Tasks {
		if m.config.Tasks[i].ID == m.taskToDelete.ID {
			m.config.Tasks = append(m.config.Tasks[:i], m.config.Tasks[i+1:]...)
			break
		}
	}

	saveConfig(m.config)
	m.updateLists()
	m.setStatus("Task deleted")
	m.taskToDelete = nil
	m.mode = m.prevMode
	return m, nil
}

func (m model) handleDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m.deleteTask()
	case "n", "N", "esc":
		m.taskToDelete = nil
		m.mode = m.prevMode
		return m, nil
	}
	return m, nil
}

func (m model) handleCategoryForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		m.mode = m.prevMode
		m.categoryInput.Blur()
		return m, nil

	case "enter":
		name := strings.TrimSpace(m.categoryInput.Value())
		if name != "" {
			newCat := Category{
				ID:   generateID(),
				Name: name,
			}
			m.config.Categories = append(m.config.Categories, newCat)
			saveConfig(m.config)
			m.setStatus("Category created")
		}
		m.mode = m.prevMode
		m.categoryInput.Blur()
		return m, nil
	}

	m.categoryInput, cmd = m.categoryInput.Update(msg)
	return m, cmd
}

func (m model) handleTaskForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		m.mode = m.prevMode
		for i := range m.taskInputs {
			m.taskInputs[i].Blur()
		}
		return m, nil

	case "tab", "shift+tab":
		if msg.String() == "tab" {
			m.formFocus++
		} else {
			m.formFocus--
		}

		if m.formFocus < 0 {
			m.formFocus = len(m.taskInputs) + len(m.config.Categories) - 1
		} else if m.formFocus >= len(m.taskInputs)+len(m.config.Categories) {
			m.formFocus = 0
		}

		for i := range m.taskInputs {
			m.taskInputs[i].Blur()
		}

		if m.formFocus < len(m.taskInputs) {
			m.taskInputs[m.formFocus].Focus()
			return m, textinput.Blink
		}
		return m, nil

	case "enter":
		catIndex := m.formFocus - len(m.taskInputs)
		if catIndex >= 0 && catIndex < len(m.config.Categories) {
			content := strings.TrimSpace(m.taskInputs[0].Value())
			if content != "" {
				priority := P1High
				if p := m.taskInputs[1].Value(); p != "" {
					switch p[0] {
					case '0':
						priority = P0Critical
					case '2':
						priority = P2Medium
					case '3':
						priority = P3Low
					}
				}

				newTask := Task{
					ID:         generateID(),
					Content:    content,
					CategoryID: m.config.Categories[catIndex].ID,
					Priority:   priority,
					CreatedAt:  time.Now(),
				}
				m.config.Tasks = append(m.config.Tasks, newTask)
				saveConfig(m.config)
				m.updateLists()
				m.setStatus("Task created")
			}
			m.mode = m.prevMode
			for i := range m.taskInputs {
				m.taskInputs[i].Blur()
			}
		}
		return m, nil
	}

	if m.formFocus < len(m.taskInputs) {
		m.taskInputs[m.formFocus], cmd = m.taskInputs[m.formFocus].Update(msg)
	}
	return m, cmd
}

func (m *model) setStatus(msg string) {
	m.statusMsg = msg
	m.statusUntil = time.Now().Add(2 * time.Second)
}

func (m model) View() string {
	if !m.ready {
		return "\nInitializing..."
	}

	switch m.mode {
	case categoryFormView:
		return m.renderCategoryForm()
	case taskFormView:
		return m.renderTaskForm()
	case completedView:
		return m.renderCompletedView()
	case deleteConfirmView:
		return m.renderDeleteConfirm()
	default:
		return m.renderListView()
	}
}

func (m model) renderListView() string {
	var output strings.Builder

	output.WriteString(m.list.View())
	output.WriteString("\n")
	output.WriteString(m.renderFooter())

	return output.String()
}

func (m model) renderCompletedView() string {
	var output strings.Builder

	output.WriteString(m.completedList.View())
	output.WriteString("\n")
	output.WriteString(m.renderFooter())

	return output.String()
}

func (m model) renderCategoryForm() string {
	var output strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#4ec9b0"))

	output.WriteString(titleStyle.Render("New Category"))
	output.WriteString("\n\n")

	output.WriteString("Name:\n")
	output.WriteString(m.categoryInput.View())
	output.WriteString("\n\n")

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666"))
	output.WriteString(helpStyle.Render("enter: save | esc: cancel"))

	return lipgloss.NewStyle().Padding(1, 2).Render(output.String())
}

func (m model) renderTaskForm() string {
	var output strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#4ec9b0"))

	output.WriteString(titleStyle.Render("New Task"))
	output.WriteString("\n\n")

	// Task content input
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#999"))
	if m.formFocus == 0 {
		labelStyle = labelStyle.Foreground(lipgloss.Color("#4ec9b0"))
	}
	output.WriteString(labelStyle.Render("Content:"))
	output.WriteString("\n")
	output.WriteString(m.taskInputs[0].View())
	output.WriteString("\n\n")

	// Priority input
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#999"))
	if m.formFocus == 1 {
		labelStyle = labelStyle.Foreground(lipgloss.Color("#4ec9b0"))
	}
	output.WriteString(labelStyle.Render("Priority (0-3):"))
	output.WriteString("\n")
	output.WriteString(m.taskInputs[1].View())
	output.WriteString("\n\n")

	// Category selection
	output.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#999")).Render("Category:"))
	output.WriteString("\n")

	for i, cat := range m.config.Categories {
		catIndex := len(m.taskInputs) + i
		cursor := "  "
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("#666"))

		if m.formFocus == catIndex {
			cursor = "> "
			style = style.Foreground(lipgloss.Color("#4ec9b0")).Bold(true)
		}

		output.WriteString(cursor + style.Render(cat.Name) + "\n")
	}

	output.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666"))
	output.WriteString(helpStyle.Render("tab: next | enter: save | esc: cancel"))

	return lipgloss.NewStyle().Padding(1, 2).Render(output.String())
}

func (m model) renderDeleteConfirm() string {
	var output strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#d73a4a"))

	output.WriteString(titleStyle.Render("Delete Task?"))
	output.WriteString("\n\n")

	if m.taskToDelete != nil {
		taskStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#d4d4d4"))
		output.WriteString(taskStyle.Render(m.taskToDelete.Content))
		output.WriteString("\n\n")
	}

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666"))
	output.WriteString(helpStyle.Render("y: delete | n/esc: cancel"))

	return lipgloss.NewStyle().Padding(1, 2).Render(output.String())
}

func (m model) renderFooter() string {
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666"))
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4ec9b0"))

	status := ""
	if time.Now().Before(m.statusUntil) {
		status = statusStyle.Render(m.statusMsg) + " "
	}

	if m.mode == completedView {
		return status + helpStyle.Render("v: back | x: reopen | d: delete | q: quit")
	}

	return status + helpStyle.Render("C: category | T: task | v: completed | x: done | d: delete | q: quit")
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
