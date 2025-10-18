package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
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
	Notes       string    `json:"notes,omitempty"`
}

// TaskItem wraps Task with category name for display
type TaskItem struct {
	Task
	CategoryName string
}

// Implement list.Item interface for TaskItem
func (t TaskItem) Title() string {
	priorityStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Priority.Color())).
		Bold(true)

	categoryStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666")).
		Italic(true)

	checkbox := "[ ]"
	if t.Done {
		checkbox = "[x]"
	}

	return fmt.Sprintf("%s %-4s %s %s",
		checkbox,
		priorityStyle.Render(t.Priority.String()),
		t.Content,
		categoryStyle.Render(fmt.Sprintf("[%s]", t.CategoryName)),
	)
}

func (t TaskItem) Description() string {
	age := time.Since(t.CreatedAt)
	days := int(age.Hours() / 24)

	var ageStr string
	if days == 0 {
		ageStr = "Created today"
	} else if days == 1 {
		ageStr = "1 day old"
	} else {
		ageStr = fmt.Sprintf("%d days old", days)
	}

	if t.Done {
		return fmt.Sprintf("Completed: %s • %s", t.CompletedAt.Format("2006-01-02 15:04"), ageStr)
	}
	return ageStr
}

func (t TaskItem) FilterValue() string {
	return t.Content
}

// Implement list.Item interface for Category
func (c Category) Title() string {
	return c.Name
}

func (c Category) Description() string {
	return fmt.Sprintf("ID: %s", c.ID)
}

func (c Category) FilterValue() string {
	return c.Name
}

// Category for organizing tasks
type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Config stores all tasks and categories
type Config struct {
	Categories          []Category `json:"categories"`
	Tasks               []Task     `json:"tasks"`
	LastUpdate          time.Time  `json:"last_update"`
	Version             string     `json:"version"`
	GitHubSetupComplete bool       `json:"github_setup_complete,omitempty"`
}

type viewMode int

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

// syncResultMsg is sent when the GitHub sync completes
type syncResultMsg struct {
	success bool
	error   string
}

// pullResultMsg is sent when the GitHub pull completes
type pullResultMsg struct {
	success      bool
	error        string
	remoteConfig *Config
	hasConflict  bool
}

// firstRunStep tracks the first-run setup flow
type firstRunStep int

const (
	welcomeStep firstRunStep = iota
	hasRepoPromptStep
	createRepoPromptStep
	pullingStep
	pushingStep
	completeStep
)

// Model is the Bubble Tea model
type model struct {
	config           *Config
	width            int
	height           int
	mode             viewMode
	prevMode         viewMode
	ready            bool
	statusMsg        string
	statusUntil      time.Time
	categoryInput    textinput.Model
	taskInputs       []textinput.Model
	formFocus        int
	list             list.Model
	completedList    list.Model
	categoryList     list.Model
	taskToDelete     *Task
	categoryToDelete *Category
	editingCategory  *Category
	editingTask      *Task
	notesTextarea    textarea.Model
	configChanged    bool
	syncInProgress   bool
	pullInProgress   bool
	remoteConfig     *Config
	spinner          spinner.Model
	firstRunStep     firstRunStep
	firstRunError    string
}

func main() {
	// Check for seed flag
	if len(os.Args) > 1 && os.Args[1] == "seed" {
		cfg := seedWeekendTasks()
		if err := saveConfig(cfg); err != nil {
			fmt.Printf("Error seeding config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Config seeded with weekend tasks!")
		os.Exit(0)
	}

	// Check for pull flag (for initial setup on new machine)
	if len(os.Args) > 1 && os.Args[1] == "--pull" {
		fmt.Println("Pulling config from GitHub...")
		if err := pullConfigFromGitHub(); err != nil {
			fmt.Printf("Error pulling config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Config pulled successfully!")
		os.Exit(0)
	}

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
		notesTextarea: textarea.New(),
		firstRunStep:  welcomeStep,
	}

	// Check if this is first run (GitHub not set up yet)
	if !cfg.GitHubSetupComplete {
		m.mode = firstRunView
	}

	m.categoryInput.Placeholder = "Category name"
	m.categoryInput.CharLimit = 50

	m.taskInputs[0] = textinput.New()
	m.taskInputs[0].Placeholder = "Task content"
	m.taskInputs[0].CharLimit = 200

	m.taskInputs[1] = textinput.New()
	m.taskInputs[1].Placeholder = "Priority (0-3)"
	m.taskInputs[1].CharLimit = 1

	m.notesTextarea.Placeholder = "Add notes here..."
	m.notesTextarea.CharLimit = 2000
	m.notesTextarea.SetHeight(10)

	// Initialize lists
	m.list = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	m.list.Title = "Tasks"
	m.list.SetShowStatusBar(false)
	m.list.SetFilteringEnabled(false)

	// Disable default keybindings we don't want
	m.list.KeyMap.GoToStart.SetEnabled(false)
	m.list.KeyMap.GoToEnd.SetEnabled(false)

	m.list.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("T"), key.WithHelp("T", "new task")),
			key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "new category")),
			key.NewBinding(key.WithKeys("x", "space"), key.WithHelp("x/space", "toggle done")),
			key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit task")),
			key.NewBinding(key.WithKeys("enter", "i"), key.WithHelp("enter/i", "view details")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		}
	}
	m.list.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "categories")),
			key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "completed")),
			key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "sync github")),
			key.NewBinding(key.WithKeys(""), key.WithHelp("", "todobi - simple terminal task manager - builtbywilly.com")),
		}
	}

	m.completedList = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	m.completedList.Title = "Completed Tasks"
	m.completedList.SetShowStatusBar(false)
	m.completedList.SetFilteringEnabled(false)

	m.categoryList = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	m.categoryList.Title = "Categories"
	m.categoryList.SetShowStatusBar(false)
	m.categoryList.SetFilteringEnabled(false)

	// Initialize spinner
	m.spinner = spinner.New()
	m.spinner.Spinner = spinner.Pulse
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#4ec9b0"))

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

func (m *model) saveConfigAndMarkChanged() {
	saveConfig(m.config)
	m.configChanged = true
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

func seedWeekendTasks() *Config {
	return &Config{
		Version: "1.3.0",
		Categories: []Category{
			{ID: "gummy-agents", Name: "Gummy Agents"},
			{ID: "master-claude", Name: "Master Claude"},
			{ID: "eldercare", Name: "Eldercare"},
			{ID: "homelab", Name: "Homelab"},
			{ID: "tailscale", Name: "File Sharing"},
		},
		Tasks: []Task{
			{
				ID:         "1",
				Content:    "Pull down gummy-agents repo and review codebase",
				CategoryID: "gummy-agents",
				Priority:   P1High,
				CreatedAt:  time.Now(),
			},
			{
				ID:         "2",
				Content:    "Review and organize master-claude-work projects",
				CategoryID: "master-claude",
				Priority:   P1High,
				CreatedAt:  time.Now(),
			},
			{
				ID:         "3",
				Content:    "Address eldercare issues and documentation",
				CategoryID: "eldercare",
				Priority:   P0Critical,
				CreatedAt:  time.Now(),
			},
			{
				ID:         "4",
				Content:    "Homelab infrastructure maintenance and updates",
				CategoryID: "homelab",
				Priority:   P2Medium,
				CreatedAt:  time.Now(),
			},
			{
				ID:         "5",
				Content:    "Setup file sharing across tailscale network",
				CategoryID: "tailscale",
				Priority:   P2Medium,
				CreatedAt:  time.Now(),
			},
		},
	}
}

// Bubble Tea interface
func (m model) Init() tea.Cmd {
	return m.spinner.Tick
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
		m.categoryList.SetSize(m.width, listHeight)

		if !m.ready {
			m.ready = true
			m.updateLists()
		}
		return m, nil

	case syncResultMsg:
		m.syncInProgress = false
		if m.mode == firstRunView {
			// Handle first-run sync completion
			if msg.success {
				m.firstRunStep = completeStep
				m.firstRunError = ""
			} else {
				m.firstRunError = msg.error
				// Allow user to continue despite error
			}
			return m, nil
		}
		if msg.success {
			m.setStatus("Synced to GitHub successfully!")
			m.configChanged = false
		} else {
			m.setStatus("Sync failed: " + msg.error)
		}
		m.mode = m.prevMode
		return m, nil

	case pullResultMsg:
		m.pullInProgress = false
		if m.mode == firstRunView {
			// Handle first-run pull completion
			if msg.success {
				// Apply remote config without conflict checking on first run
				m.config = msg.remoteConfig
				m.updateLists()
				m.firstRunStep = completeStep
				m.firstRunError = ""
			} else {
				m.firstRunError = msg.error
				// Allow user to continue despite error
			}
			return m, nil
		}
		if msg.success {
			if msg.hasConflict {
				// Store remote config for conflict resolution
				m.remoteConfig = msg.remoteConfig
				m.setStatus("Conflict detected - choose merge strategy")
				m.mode = pullConfirmView
			} else {
				// No conflict, just apply the remote config
				m.config = msg.remoteConfig
				m.updateLists()
				m.configChanged = false
				m.setStatus("Pulled from GitHub successfully!")
				m.mode = m.prevMode
			}
		} else {
			m.setStatus("Pull failed: " + msg.error)
			m.mode = m.prevMode
		}
		return m, nil

	case tea.KeyMsg:
		// Form handling
		if m.mode == firstRunView {
			return m.handleFirstRun(msg)
		}
		if m.mode == categoryFormView {
			return m.handleCategoryForm(msg)
		}
		if m.mode == taskFormView {
			return m.handleTaskForm(msg)
		}
		if m.mode == editTaskView {
			return m.handleTaskEdit(msg)
		}
		if m.mode == taskDetailView {
			return m.handleTaskDetail(msg)
		}
		if m.mode == deleteConfirmView {
			return m.handleDeleteConfirm(msg)
		}
		if m.mode == categoryListView {
			return m.handleCategoryList(msg)
		}
		if m.mode == syncConfirmView {
			return m.handleSyncConfirm(msg)
		}
		if m.mode == pullConfirmView {
			return m.handlePullConfirm(msg)
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

		case "c":
			m.prevMode = m.mode
			m.mode = categoryListView
			m.updateCategoryList()
			return m, nil

		case "C":
			m.prevMode = m.mode
			m.mode = categoryFormView
			m.editingCategory = nil
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

		case "e":
			return m.startEditTask()

		case "enter", "i":
			return m.viewTaskDetail()

		case "G":
			m.prevMode = m.mode
			m.mode = syncConfirmView
			return m, nil

		case "g":
			m.prevMode = m.mode
			m.pullInProgress = true
			m.setStatus("Pulling from GitHub...")
			return m, tea.Batch(pullFromGitHubCmd(m.config), m.spinner.Tick)
		}
	}

	// Handle spinner tick messages
	if _, ok := msg.(spinner.TickMsg); ok && (m.syncInProgress || m.pullInProgress || m.mode == firstRunView) {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
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
	// Helper to find category name
	getCategoryName := func(categoryID string) string {
		for _, cat := range m.config.Categories {
			if cat.ID == categoryID {
				return cat.Name
			}
		}
		return "Unknown"
	}

	// Update active tasks list
	var activeTasks []TaskItem
	for _, task := range m.config.Tasks {
		if !task.Done {
			activeTasks = append(activeTasks, TaskItem{
				Task:         task,
				CategoryName: getCategoryName(task.CategoryID),
			})
		}
	}

	// Sort by category name, then by priority
	sort.Slice(activeTasks, func(i, j int) bool {
		if activeTasks[i].CategoryName != activeTasks[j].CategoryName {
			return activeTasks[i].CategoryName < activeTasks[j].CategoryName
		}
		return activeTasks[i].Priority < activeTasks[j].Priority
	})

	var activeItems []list.Item
	for _, task := range activeTasks {
		activeItems = append(activeItems, task)
	}
	m.list.SetItems(activeItems)

	// Update completed tasks list
	var completedTasks []TaskItem
	for _, task := range m.config.Tasks {
		if task.Done {
			completedTasks = append(completedTasks, TaskItem{
				Task:         task,
				CategoryName: getCategoryName(task.CategoryID),
			})
		}
	}

	// Sort completed tasks by category too
	sort.Slice(completedTasks, func(i, j int) bool {
		if completedTasks[i].CategoryName != completedTasks[j].CategoryName {
			return completedTasks[i].CategoryName < completedTasks[j].CategoryName
		}
		return completedTasks[i].CompletedAt.After(completedTasks[j].CompletedAt)
	})

	var completedItems []list.Item
	for _, task := range completedTasks {
		completedItems = append(completedItems, task)
	}
	m.completedList.SetItems(completedItems)
}

func (m *model) updateCategoryList() {
	var items []list.Item
	for _, cat := range m.config.Categories {
		items = append(items, cat)
	}
	m.categoryList.SetItems(items)
}

func (m model) toggleTask() (tea.Model, tea.Cmd) {
	var selectedTask Task
	found := false

	if m.mode == completedView {
		if item := m.completedList.SelectedItem(); item != nil {
			selectedTask = item.(TaskItem).Task
			found = true
		}
	} else {
		if item := m.list.SelectedItem(); item != nil {
			selectedTask = item.(TaskItem).Task
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

	m.saveConfigAndMarkChanged()
	m.updateLists()
	return m, nil
}

func (m model) confirmDelete() (tea.Model, tea.Cmd) {
	var selectedTask Task
	found := false

	if m.mode == completedView {
		if item := m.completedList.SelectedItem(); item != nil {
			selectedTask = item.(TaskItem).Task
			found = true
		}
	} else if m.mode == listView {
		if item := m.list.SelectedItem(); item != nil {
			selectedTask = item.(TaskItem).Task
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

	m.saveConfigAndMarkChanged()
	m.updateLists()
	m.setStatus("Task deleted")
	m.taskToDelete = nil
	m.mode = m.prevMode
	return m, nil
}

func (m model) handleDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.taskToDelete != nil {
			return m.deleteTask()
		} else if m.categoryToDelete != nil {
			return m.deleteCategory()
		}
	case "n", "N", "esc":
		m.taskToDelete = nil
		m.categoryToDelete = nil
		m.mode = m.prevMode
		return m, nil
	}
	return m, nil
}

func (m model) handleSyncConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.syncInProgress = true
		m.setStatus("Syncing to GitHub...")
		// Return both the sync command AND the spinner tick to start animation
		return m, tea.Batch(syncToGitHubCmd(), m.spinner.Tick)
	case "n", "N", "esc":
		m.mode = m.prevMode
		return m, nil
	}
	return m, nil
}

func (m model) handlePullConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "l", "L":
		// Keep local - discard remote
		m.remoteConfig = nil
		m.mode = m.prevMode
		m.setStatus("Kept local version")
		return m, nil
	case "r", "R":
		// Use remote - overwrite local
		if m.remoteConfig != nil {
			m.config = m.remoteConfig
			m.saveConfigAndMarkChanged()
			m.updateLists()
			m.remoteConfig = nil
			m.configChanged = false
			m.setStatus("Applied remote version")
		}
		m.mode = m.prevMode
		return m, nil
	case "m", "M":
		// Merge: combine tasks and categories
		if m.remoteConfig != nil {
			m.config = mergeConfigs(m.config, m.remoteConfig)
			m.saveConfigAndMarkChanged()
			m.updateLists()
			m.remoteConfig = nil
			m.configChanged = false
			m.setStatus("Merged local and remote")
		}
		m.mode = m.prevMode
		return m, nil
	case "esc":
		m.remoteConfig = nil
		m.mode = m.prevMode
		return m, nil
	}
	return m, nil
}

// mergeConfigs combines local and remote configs intelligently
func mergeConfigs(local, remote *Config) *Config {
	merged := &Config{
		Version:    local.Version,
		LastUpdate: time.Now(),
	}

	// Merge categories by ID
	categoryMap := make(map[string]Category)
	for _, cat := range local.Categories {
		categoryMap[cat.ID] = cat
	}
	for _, cat := range remote.Categories {
		// Remote category takes precedence if exists in both
		categoryMap[cat.ID] = cat
	}
	for _, cat := range categoryMap {
		merged.Categories = append(merged.Categories, cat)
	}

	// Merge tasks by ID
	taskMap := make(map[string]Task)
	for _, task := range local.Tasks {
		taskMap[task.ID] = task
	}
	for _, task := range remote.Tasks {
		// Use newer task if it exists in both
		if existing, ok := taskMap[task.ID]; ok {
			if task.CreatedAt.After(existing.CreatedAt) {
				taskMap[task.ID] = task
			}
		} else {
			taskMap[task.ID] = task
		}
	}
	for _, task := range taskMap {
		merged.Tasks = append(merged.Tasks, task)
	}

	return merged
}

func (m model) deleteCategory() (tea.Model, tea.Cmd) {
	if m.categoryToDelete == nil {
		return m, nil
	}

	// Check if category has tasks
	tasksInCategory := 0
	for _, task := range m.config.Tasks {
		if task.CategoryID == m.categoryToDelete.ID {
			tasksInCategory++
		}
	}

	if tasksInCategory > 0 {
		m.setStatus(fmt.Sprintf("Cannot delete: %d tasks in category", tasksInCategory))
		m.categoryToDelete = nil
		m.mode = m.prevMode
		return m, nil
	}

	// Delete the category
	for i := range m.config.Categories {
		if m.config.Categories[i].ID == m.categoryToDelete.ID {
			m.config.Categories = append(m.config.Categories[:i], m.config.Categories[i+1:]...)
			break
		}
	}

	m.saveConfigAndMarkChanged()
	m.updateCategoryList()
	m.setStatus("Category deleted")
	m.categoryToDelete = nil
	m.mode = m.prevMode
	return m, nil
}

// syncToGitHubCmd returns a tea.Cmd that performs the GitHub sync asynchronously
func syncToGitHubCmd() tea.Cmd {
	return func() tea.Msg {
		home, err := os.UserHomeDir()
		if err != nil {
			return syncResultMsg{success: false, error: err.Error()}
		}

		configPath := filepath.Join(home, configFileName)
		repoName := "todobi-sync"

		// Check if gh CLI is installed
		if err := exec.Command("gh", "--version").Run(); err != nil {
			return syncResultMsg{success: false, error: "gh CLI not installed. Install from https://cli.github.com"}
		}

		// Create temp directory for git operations
		tmpDir := filepath.Join(os.TempDir(), "todobi-sync-tmp")
		os.RemoveAll(tmpDir)
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			return syncResultMsg{success: false, error: "Failed to create temp directory: " + err.Error()}
		}
		defer os.RemoveAll(tmpDir)

		// Check if repo exists
		checkCmd := exec.Command("gh", "repo", "view", repoName, "--json", "name")
		repoExists := checkCmd.Run() == nil

		if !repoExists {
			// Repo doesn't exist, create it
			createCmd := exec.Command("gh", "repo", "create", repoName, "--private", "--clone=false")
			createCmd.Stdin = nil  // Prevent password prompts
			output, err := createCmd.CombinedOutput()
			if err != nil {
				return syncResultMsg{success: false, error: fmt.Sprintf("Error creating repo: %s - %s", err.Error(), string(output))}
			}
			// Now clone the newly created repo
			cloneCmd := exec.Command("gh", "repo", "clone", repoName, tmpDir)
			cloneCmd.Stdin = nil  // Prevent password prompts
			cloneCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
			output, err = cloneCmd.CombinedOutput()
			if err != nil {
				return syncResultMsg{success: false, error: fmt.Sprintf("Error cloning new repo: %s - %s", err.Error(), string(output))}
			}
		} else {
			// Clone existing repo
			cloneCmd := exec.Command("gh", "repo", "clone", repoName, tmpDir)
			cloneCmd.Stdin = nil  // Prevent password prompts
			cloneCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
			output, err := cloneCmd.CombinedOutput()
			if err != nil {
				return syncResultMsg{success: false, error: fmt.Sprintf("Error cloning repo: %s - %s", err.Error(), string(output))}
			}
		}

		// Copy config file to repo
		destPath := filepath.Join(tmpDir, ".todobi.conf")
		data, err := os.ReadFile(configPath)
		if err != nil {
			return syncResultMsg{success: false, error: "Error reading config: " + err.Error()}
		}

		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return syncResultMsg{success: false, error: "Error writing config to repo: " + err.Error()}
		}

		// Git add, commit, push
		addCmd := exec.Command("git", "add", ".todobi.conf")
		addCmd.Dir = tmpDir
		if err := addCmd.Run(); err != nil {
			return syncResultMsg{success: false, error: "Error adding file: " + err.Error()}
		}

		commitCmd := exec.Command("git", "commit", "-m", fmt.Sprintf("Update tasks - %s", time.Now().Format("2006-01-02 15:04:05")))
		commitCmd.Dir = tmpDir
		commitCmd.Run() // Ignore error if nothing to commit

		pushCmd := exec.Command("git", "push")
		pushCmd.Dir = tmpDir
		if err := pushCmd.Run(); err != nil {
			return syncResultMsg{success: false, error: "Error pushing to GitHub: " + err.Error()}
		}

		return syncResultMsg{success: true}
	}
}

// pullFromGitHubCmd returns a tea.Cmd that pulls config from GitHub asynchronously
func pullFromGitHubCmd(localConfig *Config) tea.Cmd {
	return func() tea.Msg {
		repoName := "todobi-sync"

		// Check if gh CLI is installed
		if err := exec.Command("gh", "--version").Run(); err != nil {
			return pullResultMsg{success: false, error: "gh CLI not installed. Install from https://cli.github.com"}
		}

		// Check if repo exists
		checkCmd := exec.Command("gh", "repo", "view", repoName, "--json", "name")
		if checkCmd.Run() != nil {
			return pullResultMsg{success: false, error: "Remote repo 'todobi-sync' does not exist. Push to GitHub first with 'G'"}
		}

		// Create temp directory for git operations
		tmpDir := filepath.Join(os.TempDir(), "todobi-pull-tmp")
		os.RemoveAll(tmpDir)
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			return pullResultMsg{success: false, error: "Failed to create temp directory: " + err.Error()}
		}
		defer os.RemoveAll(tmpDir)

		// Clone the repo
		cloneCmd := exec.Command("gh", "repo", "clone", repoName, tmpDir)
		cloneCmd.Stdin = nil  // Prevent password prompts
		cloneCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err := cloneCmd.CombinedOutput()
		if err != nil {
			return pullResultMsg{success: false, error: fmt.Sprintf("Error cloning repo: %s - %s", err.Error(), string(output))}
		}

		// Read the remote config
		remotePath := filepath.Join(tmpDir, ".todobi.conf")
		data, err := os.ReadFile(remotePath)
		if err != nil {
			return pullResultMsg{success: false, error: "Error reading remote config: " + err.Error()}
		}

		var remoteConfig Config
		if err := json.Unmarshal(data, &remoteConfig); err != nil {
			return pullResultMsg{success: false, error: "Error parsing remote config: " + err.Error()}
		}

		// Check for conflicts: if local has changes AND remote is newer
		hasConflict := false
		if localConfig.LastUpdate.After(remoteConfig.LastUpdate) {
			// Local is newer - this is a conflict if remote also has changes
			// For simplicity, we'll consider it a conflict if timestamps differ
			hasConflict = !localConfig.LastUpdate.Equal(remoteConfig.LastUpdate)
		}

		return pullResultMsg{
			success:      true,
			remoteConfig: &remoteConfig,
			hasConflict:  hasConflict,
		}
	}
}

// pullConfigFromGitHub is a helper for the --pull CLI flag
func pullConfigFromGitHub() error {
	repoName := "todobi-sync"

	// Check if gh CLI is installed
	if err := exec.Command("gh", "--version").Run(); err != nil {
		return fmt.Errorf("gh CLI not installed. Install from https://cli.github.com")
	}

	// Check if repo exists
	checkCmd := exec.Command("gh", "repo", "view", repoName, "--json", "name")
	if checkCmd.Run() != nil {
		return fmt.Errorf("remote repo 'todobi-sync' does not exist")
	}

	// Create temp directory
	tmpDir := filepath.Join(os.TempDir(), "todobi-pull-tmp")
	os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Clone the repo
	cloneCmd := exec.Command("gh", "repo", "clone", repoName, tmpDir)
	cloneCmd.Stdin = nil  // Prevent password prompts
	cloneCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if err := cloneCmd.Run(); err != nil {
		return fmt.Errorf("error cloning repo: %w", err)
	}

	// Read the remote config
	remotePath := filepath.Join(tmpDir, ".todobi.conf")
	data, err := os.ReadFile(remotePath)
	if err != nil {
		return fmt.Errorf("error reading remote config: %w", err)
	}

	// Write to local config path
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting home directory: %w", err)
	}

	localPath := filepath.Join(home, configFileName)
	if err := os.WriteFile(localPath, data, 0644); err != nil {
		return fmt.Errorf("error writing local config: %w", err)
	}

	return nil
}

func (m model) handleCategoryForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		m.mode = m.prevMode
		m.categoryInput.Blur()
		m.editingCategory = nil
		return m, nil

	case "enter":
		name := strings.TrimSpace(m.categoryInput.Value())
		if name != "" {
			if m.editingCategory != nil {
				// Edit existing category
				for i := range m.config.Categories {
					if m.config.Categories[i].ID == m.editingCategory.ID {
						m.config.Categories[i].Name = name
						break
					}
				}
				m.saveConfigAndMarkChanged()
				m.updateCategoryList()
				m.setStatus("Category updated")
			} else {
				// Create new category
				newCat := Category{
					ID:   generateID(),
					Name: name,
				}
				m.config.Categories = append(m.config.Categories, newCat)
				m.saveConfigAndMarkChanged()
				m.setStatus("Category created")
			}
		}
		m.mode = m.prevMode
		m.categoryInput.Blur()
		m.editingCategory = nil
		return m, nil
	}

	m.categoryInput, cmd = m.categoryInput.Update(msg)
	return m, cmd
}

func (m model) handleCategoryList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "e":
		if item := m.categoryList.SelectedItem(); item != nil {
			cat := item.(Category)
			m.editingCategory = &cat
			m.prevMode = categoryListView
			m.mode = categoryFormView
			m.categoryInput.SetValue(cat.Name)
			m.categoryInput.Focus()
			return m, textinput.Blink
		}
		return m, nil

	case "d":
		if item := m.categoryList.SelectedItem(); item != nil {
			cat := item.(Category)
			m.categoryToDelete = &cat
			m.prevMode = categoryListView
			m.mode = deleteConfirmView
		}
		return m, nil

	case "esc", "q":
		m.mode = listView
		return m, nil

	default:
		// Pass unhandled keys to the list for navigation
		m.categoryList, cmd = m.categoryList.Update(msg)
		return m, cmd
	}
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

	case "up", "down":
		// Navigate with arrow keys
		if msg.String() == "down" {
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
		// Progress through form or submit
		catIndex := m.formFocus - len(m.taskInputs)

		// If we're on a category, submit the form
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
				m.saveConfigAndMarkChanged()
				m.updateLists()
				m.setStatus("Task created")
			}
			m.mode = m.prevMode
			for i := range m.taskInputs {
				m.taskInputs[i].Blur()
			}
			return m, nil
		}

		// Otherwise, progress to next field
		m.formFocus++
		if m.formFocus >= len(m.taskInputs)+len(m.config.Categories) {
			m.formFocus = len(m.taskInputs) + len(m.config.Categories) - 1
		}

		for i := range m.taskInputs {
			m.taskInputs[i].Blur()
		}

		if m.formFocus < len(m.taskInputs) {
			m.taskInputs[m.formFocus].Focus()
			return m, textinput.Blink
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
	case firstRunView:
		return m.renderFirstRun()
	case categoryFormView:
		return m.renderCategoryForm()
	case taskFormView:
		return m.renderTaskForm()
	case editTaskView:
		return m.renderEditTaskForm()
	case taskDetailView:
		return m.renderTaskDetailView()
	case completedView:
		return m.renderCompletedView()
	case deleteConfirmView:
		return m.renderDeleteConfirm()
	case categoryListView:
		return m.renderCategoryList()
	case syncConfirmView:
		return m.renderSyncConfirm()
	case pullConfirmView:
		return m.renderPullConfirm()
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

func (m model) renderCategoryList() string {
	var output strings.Builder

	output.WriteString(m.categoryList.View())
	output.WriteString("\n")

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666"))
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4ec9b0"))

	status := ""
	if time.Now().Before(m.statusUntil) {
		status = statusStyle.Render(m.statusMsg) + " "
	}

	output.WriteString(status + helpStyle.Render("e: edit | d: delete | esc: back"))

	return output.String()
}

func (m model) renderCategoryForm() string {
	var output strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#4ec9b0"))

	if m.editingCategory != nil {
		output.WriteString(titleStyle.Render("Edit Category"))
	} else {
		output.WriteString(titleStyle.Render("New Category"))
	}
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
	output.WriteString(helpStyle.Render("arrows: navigate | enter: next/save | esc: cancel"))

	return lipgloss.NewStyle().Padding(1, 2).Render(output.String())
}

func (m model) renderDeleteConfirm() string {
	var output strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#d73a4a"))

	if m.taskToDelete != nil {
		output.WriteString(titleStyle.Render("Delete Task?"))
		output.WriteString("\n\n")

		taskStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#d4d4d4"))
		output.WriteString(taskStyle.Render(m.taskToDelete.Content))
		output.WriteString("\n\n")
	} else if m.categoryToDelete != nil {
		output.WriteString(titleStyle.Render("Delete Category?"))
		output.WriteString("\n\n")

		catStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#d4d4d4"))
		output.WriteString(catStyle.Render(m.categoryToDelete.Name))
		output.WriteString("\n\n")
	}

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666"))
	output.WriteString(helpStyle.Render("y: delete | n/esc: cancel"))

	return lipgloss.NewStyle().Padding(1, 2).Render(output.String())
}

func (m model) renderSyncConfirm() string {
	var output strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#4ec9b0"))

	output.WriteString(titleStyle.Render("Sync to GitHub?"))
	output.WriteString("\n\n")

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#d4d4d4"))

	output.WriteString(infoStyle.Render("This will sync your .todobi.conf to a private GitHub repo"))
	output.WriteString("\n")
	output.WriteString(infoStyle.Render("named 'todobi-sync'."))
	output.WriteString("\n\n")

	if m.syncInProgress {
		output.WriteString(fmt.Sprintf("%s %s", m.spinner.View(), infoStyle.Render("Syncing to GitHub...")))
	} else {
		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666"))
		output.WriteString(helpStyle.Render("y: sync | n/esc: cancel"))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(output.String())
}

func (m model) renderPullConfirm() string {
	var output strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#4ec9b0"))

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#d4d4d4"))

	warningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ffc107")).
		Bold(true)

	if m.pullInProgress {
		output.WriteString(titleStyle.Render("Pulling from GitHub"))
		output.WriteString("\n\n")
		output.WriteString(fmt.Sprintf("%s %s", m.spinner.View(), infoStyle.Render("Fetching remote config...")))
	} else if m.remoteConfig != nil {
		// Show conflict resolution UI
		output.WriteString(warningStyle.Render("Sync Conflict Detected!"))
		output.WriteString("\n\n")
		output.WriteString(infoStyle.Render("Both local and remote have changes."))
		output.WriteString("\n")
		output.WriteString(infoStyle.Render("Choose how to resolve:"))
		output.WriteString("\n\n")

		optionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4ec9b0"))
		output.WriteString(optionStyle.Render("L: "))
		output.WriteString(infoStyle.Render("Keep Local (discard remote changes)"))
		output.WriteString("\n")
		output.WriteString(optionStyle.Render("R: "))
		output.WriteString(infoStyle.Render("Use Remote (overwrite local changes)"))
		output.WriteString("\n")
		output.WriteString(optionStyle.Render("M: "))
		output.WriteString(infoStyle.Render("Merge (combine both, newer tasks win)"))
		output.WriteString("\n\n")

		helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666"))
		output.WriteString(helpStyle.Render("esc: cancel"))
	}

	return lipgloss.NewStyle().Padding(1, 2).Render(output.String())
}

func (m model) renderFooter() string {
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666"))
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4ec9b0"))
	warningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffc107")).Bold(true)

	status := ""
	if time.Now().Before(m.statusUntil) {
		status = statusStyle.Render(m.statusMsg) + " "
	} else if m.configChanged {
		status = warningStyle.Render("Unsynced changes - Press G to sync ") + " "
	}

	var helpText string
	if m.mode == completedView {
		helpText = "v: back | e: edit | i: details | x: reopen | d: delete | g: pull | G: push | q: quit"
	} else {
		helpText = "c: categories | C: new category | T: task | e: edit | i: details | v: completed | x: done | d: delete | g: pull | G: push | q: quit"
	}

	// Wrap help text to terminal width
	availableWidth := m.width - lipgloss.Width(status)
	if availableWidth < 40 {
		availableWidth = m.width
		status = ""
	}

	wrappedHelp := wrapText(helpText, availableWidth)
	return status + helpStyle.Render(wrappedHelp)
}

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	// If text fits, return as-is
	if len(text) <= width {
		return text
	}

	// Split by pipe separator and wrap
	parts := strings.Split(text, " | ")
	var lines []string
	var currentLine string

	for _, part := range parts {
		testLine := currentLine
		if currentLine != "" {
			testLine += " | "
		}
		testLine += part

		if len(testLine) <= width {
			currentLine = testLine
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = part
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n")
}

func (m model) startEditTask() (tea.Model, tea.Cmd) {
	var selectedTask Task
	found := false

	if m.mode == completedView {
		if item := m.completedList.SelectedItem(); item != nil {
			selectedTask = item.(TaskItem).Task
			found = true
		}
	} else {
		if item := m.list.SelectedItem(); item != nil {
			selectedTask = item.(TaskItem).Task
			found = true
		}
	}

	if !found {
		return m, nil
	}

	// Set up edit mode
	m.editingTask = &selectedTask
	m.prevMode = m.mode
	m.mode = editTaskView
	m.formFocus = 0

	// Populate form fields with current task data
	m.taskInputs[0].SetValue(selectedTask.Content)
	m.taskInputs[0].Focus()
	m.taskInputs[1].SetValue(fmt.Sprintf("%d", selectedTask.Priority))
	m.taskInputs[1].Blur()

	return m, textinput.Blink
}

func (m model) viewTaskDetail() (tea.Model, tea.Cmd) {
	var selectedTask Task
	found := false

	if m.mode == completedView {
		if item := m.completedList.SelectedItem(); item != nil {
			selectedTask = item.(TaskItem).Task
			found = true
		}
	} else {
		if item := m.list.SelectedItem(); item != nil {
			selectedTask = item.(TaskItem).Task
			found = true
		}
	}

	if !found {
		return m, nil
	}

	// Set up detail view
	m.editingTask = &selectedTask
	m.prevMode = m.mode
	m.mode = taskDetailView

	// Initialize textarea with current notes
	m.notesTextarea.SetValue(selectedTask.Notes)
	m.notesTextarea.Focus()

	return m, textarea.Blink
}

func (m model) handleTaskEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		m.mode = m.prevMode
		m.editingTask = nil
		for i := range m.taskInputs {
			m.taskInputs[i].Blur()
		}
		return m, nil

	case "up", "down":
		// Navigate with arrow keys
		if msg.String() == "down" {
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
		// Progress through form or submit
		catIndex := m.formFocus - len(m.taskInputs)

		// If we're on a category, submit the form
		if catIndex >= 0 && catIndex < len(m.config.Categories) {
			content := strings.TrimSpace(m.taskInputs[0].Value())
			if content != "" && m.editingTask != nil {
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

				// Find and update the task in config
				for i := range m.config.Tasks {
					if m.config.Tasks[i].ID == m.editingTask.ID {
						m.config.Tasks[i].Content = content
						m.config.Tasks[i].Priority = priority
						m.config.Tasks[i].CategoryID = m.config.Categories[catIndex].ID
						break
					}
				}

				m.saveConfigAndMarkChanged()
				m.updateLists()
				m.setStatus("Task updated")
			}
			m.mode = m.prevMode
			m.editingTask = nil
			for i := range m.taskInputs {
				m.taskInputs[i].Blur()
			}
			return m, nil
		}

		// Otherwise, progress to next field
		m.formFocus++
		if m.formFocus >= len(m.taskInputs)+len(m.config.Categories) {
			m.formFocus = len(m.taskInputs) + len(m.config.Categories) - 1
		}

		for i := range m.taskInputs {
			m.taskInputs[i].Blur()
		}

		if m.formFocus < len(m.taskInputs) {
			m.taskInputs[m.formFocus].Focus()
			return m, textinput.Blink
		}
		return m, nil
	}

	if m.formFocus < len(m.taskInputs) {
		m.taskInputs[m.formFocus], cmd = m.taskInputs[m.formFocus].Update(msg)
	}
	return m, cmd
}

func (m model) handleTaskDetail(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "esc":
		// Save notes before exiting
		if m.editingTask != nil {
			notes := strings.TrimSpace(m.notesTextarea.Value())
			for i := range m.config.Tasks {
				if m.config.Tasks[i].ID == m.editingTask.ID {
					if m.config.Tasks[i].Notes != notes {
						m.config.Tasks[i].Notes = notes
						m.saveConfigAndMarkChanged()
						m.setStatus("Notes saved")
					}
					break
				}
			}
		}
		m.mode = m.prevMode
		m.editingTask = nil
		m.notesTextarea.Blur()
		return m, nil

	case "ctrl+s":
		// Manual save with Ctrl+S
		if m.editingTask != nil {
			notes := strings.TrimSpace(m.notesTextarea.Value())
			for i := range m.config.Tasks {
				if m.config.Tasks[i].ID == m.editingTask.ID {
					m.config.Tasks[i].Notes = notes
					m.saveConfigAndMarkChanged()
					m.setStatus("Notes saved")
					break
				}
			}
		}
		return m, nil
	}

	m.notesTextarea, cmd = m.notesTextarea.Update(msg)
	return m, cmd
}

func (m model) renderEditTaskForm() string {
	var output strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#4ec9b0"))

	output.WriteString(titleStyle.Render("Edit Task"))
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

		// Highlight current category
		if m.editingTask != nil && cat.ID == m.editingTask.CategoryID && m.formFocus != catIndex {
			cursor = "* "
		}

		if m.formFocus == catIndex {
			cursor = "> "
			style = style.Foreground(lipgloss.Color("#4ec9b0")).Bold(true)
		}

		output.WriteString(cursor + style.Render(cat.Name) + "\n")
	}

	output.WriteString("\n")
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666"))
	output.WriteString(helpStyle.Render("arrows: navigate | enter: next/save | esc: cancel"))

	return lipgloss.NewStyle().Padding(1, 2).Render(output.String())
}

func (m model) renderTaskDetailView() string {
	if m.editingTask == nil {
		return "No task selected"
	}

	// Helper to find category name
	getCategoryName := func(categoryID string) string {
		for _, cat := range m.config.Categories {
			if cat.ID == categoryID {
				return cat.Name
			}
		}
		return "Unknown"
	}

	var output strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#4ec9b0"))

	output.WriteString(titleStyle.Render("Task Details"))
	output.WriteString("\n\n")

	// Create a bordered box for task info
	infoStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4ec9b0")).
		Padding(1, 2).
		Width(60)

	var info strings.Builder
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999")).
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#d4d4d4"))

	priorityStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.editingTask.Priority.Color())).
		Bold(true)

	info.WriteString(labelStyle.Render("Content: "))
	info.WriteString(valueStyle.Render(m.editingTask.Content))
	info.WriteString("\n\n")

	info.WriteString(labelStyle.Render("Category: "))
	info.WriteString(valueStyle.Render(getCategoryName(m.editingTask.CategoryID)))
	info.WriteString("\n\n")

	info.WriteString(labelStyle.Render("Priority: "))
	info.WriteString(priorityStyle.Render(m.editingTask.Priority.String()))
	info.WriteString("\n\n")

	info.WriteString(labelStyle.Render("Created: "))
	info.WriteString(valueStyle.Render(m.editingTask.CreatedAt.Format("2006-01-02 15:04")))
	info.WriteString("\n\n")

	age := time.Since(m.editingTask.CreatedAt)
	days := int(age.Hours() / 24)
	var ageStr string
	if days == 0 {
		ageStr = "Created today"
	} else if days == 1 {
		ageStr = "1 day old"
	} else {
		ageStr = fmt.Sprintf("%d days old", days)
	}
	info.WriteString(labelStyle.Render("Age: "))
	info.WriteString(valueStyle.Render(ageStr))
	info.WriteString("\n\n")

	info.WriteString(labelStyle.Render("Status: "))
	if m.editingTask.Done {
		doneStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4caf50"))
		info.WriteString(doneStyle.Render("Completed"))
		if !m.editingTask.CompletedAt.IsZero() {
			info.WriteString(valueStyle.Render(fmt.Sprintf(" (%s)", m.editingTask.CompletedAt.Format("2006-01-02 15:04"))))
		}
	} else {
		pendingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffc107"))
		info.WriteString(pendingStyle.Render("Pending"))
	}

	output.WriteString(infoStyle.Render(info.String()))
	output.WriteString("\n\n")

	// Notes section
	notesLabelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4ec9b0")).
		Bold(true)

	output.WriteString(notesLabelStyle.Render("Notes:"))
	output.WriteString("\n")
	output.WriteString(m.notesTextarea.View())
	output.WriteString("\n\n")

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666"))
	output.WriteString(helpStyle.Render("ctrl+s: save notes | esc: save and return"))

	return lipgloss.NewStyle().Padding(1, 2).Render(output.String())
}

// handleFirstRun manages the first-run setup flow
func (m model) handleFirstRun(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.firstRunStep {
	case welcomeStep:
		// Any key continues to next step
		m.firstRunStep = hasRepoPromptStep
		return m, nil

	case hasRepoPromptStep:
		switch msg.String() {
		case "y", "Y":
			// User has existing repo, start pulling
			m.firstRunStep = pullingStep
			m.pullInProgress = true
			return m, tea.Batch(pullFromGitHubCmd(m.config), m.spinner.Tick)
		case "n", "N":
			// User doesn't have repo, ask if they want to create one
			m.firstRunStep = createRepoPromptStep
			return m, nil
		case "esc", "ctrl+c":
			// Skip GitHub setup for now
			m.config.GitHubSetupComplete = true
			m.saveConfigAndMarkChanged()
			m.mode = listView
			m.updateLists()
			m.setStatus("GitHub sync skipped - you can sync later with 'G' or 'g'")
			return m, nil
		}

	case createRepoPromptStep:
		switch msg.String() {
		case "y", "Y":
			// Create new repo by pushing current config
			m.firstRunStep = pushingStep
			m.syncInProgress = true
			return m, tea.Batch(syncToGitHubCmd(), m.spinner.Tick)
		case "n", "N":
			// Skip GitHub setup
			m.config.GitHubSetupComplete = true
			m.saveConfigAndMarkChanged()
			m.mode = listView
			m.updateLists()
			m.setStatus("GitHub sync skipped - you can sync later with 'G' or 'g'")
			return m, nil
		case "esc", "ctrl+c":
			// Skip GitHub setup
			m.config.GitHubSetupComplete = true
			m.saveConfigAndMarkChanged()
			m.mode = listView
			m.updateLists()
			m.setStatus("GitHub sync skipped - you can sync later with 'G' or 'g'")
			return m, nil
		}

	case pullingStep, pushingStep:
		// If there's an error, allow any key to continue with local tasks
		if m.firstRunError != "" {
			m.config.GitHubSetupComplete = true
			m.saveConfigAndMarkChanged()
			m.mode = listView
			m.updateLists()
			m.setStatus("Continuing with local tasks - sync later with 'G' or 'g'")
			return m, nil
		}

	case completeStep:
		// Any key transitions to main view
		m.config.GitHubSetupComplete = true
		m.saveConfigAndMarkChanged()
		m.mode = listView
		m.updateLists()
		return m, nil
	}

	return m, nil
}

// renderFirstRun displays the first-run setup UI
func (m model) renderFirstRun() string {
	var output strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#4ec9b0")).
		Align(lipgloss.Center)

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#d4d4d4"))

	highlightStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4ec9b0")).
		Bold(true)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666"))

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#d73a4a")).
		Bold(true)

	switch m.firstRunStep {
	case welcomeStep:
		output.WriteString(titleStyle.Render("Welcome to todobi!"))
		output.WriteString("\n\n")
		output.WriteString(infoStyle.Render("todobi syncs your tasks across machines using GitHub."))
		output.WriteString("\n")
		output.WriteString(infoStyle.Render("Your tasks are stored in a private repo called 'todobi-sync'."))
		output.WriteString("\n\n")
		output.WriteString(helpStyle.Render("Press any key to continue..."))

	case hasRepoPromptStep:
		output.WriteString(titleStyle.Render("GitHub Setup"))
		output.WriteString("\n\n")
		output.WriteString(infoStyle.Render("Do you have an existing todobi-sync repo on GitHub?"))
		output.WriteString("\n\n")
		output.WriteString(highlightStyle.Render("Y: "))
		output.WriteString(infoStyle.Render("Yes, pull my tasks from GitHub"))
		output.WriteString("\n")
		output.WriteString(highlightStyle.Render("N: "))
		output.WriteString(infoStyle.Render("No, I'm starting fresh"))
		output.WriteString("\n\n")
		output.WriteString(helpStyle.Render("esc: skip GitHub sync for now"))

	case createRepoPromptStep:
		output.WriteString(titleStyle.Render("Create GitHub Repo"))
		output.WriteString("\n\n")
		output.WriteString(infoStyle.Render("Would you like to create a todobi-sync repo now?"))
		output.WriteString("\n")
		output.WriteString(infoStyle.Render("This will create a private GitHub repo and sync your tasks."))
		output.WriteString("\n\n")
		output.WriteString(highlightStyle.Render("Y: "))
		output.WriteString(infoStyle.Render("Yes, create repo and sync"))
		output.WriteString("\n")
		output.WriteString(highlightStyle.Render("N: "))
		output.WriteString(infoStyle.Render("No, continue without sync"))
		output.WriteString("\n\n")
		output.WriteString(helpStyle.Render("esc: skip GitHub sync for now"))

	case pullingStep:
		output.WriteString(titleStyle.Render("Pulling from GitHub"))
		output.WriteString("\n\n")
		output.WriteString(fmt.Sprintf("%s %s", m.spinner.View(), infoStyle.Render("Pulling your tasks from GitHub...")))
		if m.firstRunError != "" {
			output.WriteString("\n\n")
			output.WriteString(errorStyle.Render("Error: " + m.firstRunError))
			output.WriteString("\n\n")
			output.WriteString(helpStyle.Render("Press any key to continue with local tasks..."))
		}

	case pushingStep:
		output.WriteString(titleStyle.Render("Creating GitHub Repo"))
		output.WriteString("\n\n")
		output.WriteString(fmt.Sprintf("%s %s", m.spinner.View(), infoStyle.Render("Creating private repo on GitHub...")))
		if m.firstRunError != "" {
			output.WriteString("\n\n")
			output.WriteString(errorStyle.Render("Error: " + m.firstRunError))
			output.WriteString("\n\n")
			output.WriteString(helpStyle.Render("Press any key to continue with local tasks..."))
		}

	case completeStep:
		output.WriteString(titleStyle.Render("Setup Complete!"))
		output.WriteString("\n\n")
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4caf50"))
		output.WriteString(successStyle.Render("✓ GitHub sync configured successfully!"))
		output.WriteString("\n\n")
		output.WriteString(infoStyle.Render("Your tasks will now sync across all your machines."))
		output.WriteString("\n\n")
		output.WriteString(helpStyle.Render("Press any key to continue..."))
	}

	return lipgloss.NewStyle().Padding(2, 4).Render(output.String())
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
