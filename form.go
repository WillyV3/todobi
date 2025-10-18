package main

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#4ec9b0"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#666"))
	cursorStyle  = focusedStyle
	noStyle      = lipgloss.NewStyle()

	focusedButton = focusedStyle.Render("[ Save ]")
	blurredButton = blurredStyle.Render("[ Save ]")
	cancelButton  = blurredStyle.Render("[ Cancel (Esc) ]")
)

type formModel struct {
	focusIndex  int
	inputs      []textinput.Model
	priority    Priority
	editingTask *Task // nil if creating new task
}

const (
	inputContent = iota
	inputDescription
	inputURL
	inputPriority
)

func newTaskForm(task *Task) formModel {
	m := formModel{
		inputs:      make([]textinput.Model, 4),
		editingTask: task,
	}

	// Content field
	t := textinput.New()
	t.Cursor.Style = cursorStyle
	t.Placeholder = "Task content (required)"
	t.Focus()
	t.PromptStyle = focusedStyle
	t.TextStyle = focusedStyle
	t.CharLimit = 200
	if task != nil {
		t.SetValue(task.Content)
	}
	m.inputs[inputContent] = t

	// Description field
	t = textinput.New()
	t.Cursor.Style = cursorStyle
	t.Placeholder = "Description (optional)"
	t.CharLimit = 500
	if task != nil {
		t.SetValue(task.Description)
	}
	m.inputs[inputDescription] = t

	// URL field
	t = textinput.New()
	t.Cursor.Style = cursorStyle
	t.Placeholder = "URL (optional)"
	t.CharLimit = 500
	if task != nil {
		t.SetValue(task.URL)
	}
	m.inputs[inputURL] = t

	// Priority field
	t = textinput.New()
	t.Cursor.Style = cursorStyle
	t.Placeholder = "Priority: 0=P0, 1=P1, 2=P2, 3=P3, 4=Homelab, 5=Dev"
	t.CharLimit = 1
	if task != nil {
		t.SetValue(string(rune('0' + task.Priority)))
	} else {
		t.SetValue("1") // Default to P1
	}
	m.inputs[inputPriority] = t

	if task != nil {
		m.priority = task.Priority
	} else {
		m.priority = P1High
	}

	return m
}

func (m formModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m formModel) Update(msg tea.Msg) (formModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, nil

		case "tab", "shift+tab", "up", "down":
			s := msg.String()

			// Cycle indexes
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
					continue
				}
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = noStyle
				m.inputs[i].TextStyle = noStyle
			}

			return m, tea.Batch(cmds...)
		}
	}

	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *formModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	// Update priority preview
	if len(m.inputs[inputPriority].Value()) > 0 {
		switch m.inputs[inputPriority].Value()[0] {
		case '0':
			m.priority = P0Critical
		case '1':
			m.priority = P1High
		case '2':
			m.priority = P2Medium
		case '3':
			m.priority = P3Low
		case '4':
			m.priority = PHHomelab
		case '5':
			m.priority = PDev
		}
	}

	return tea.Batch(cmds...)
}

func (m formModel) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#569cd6")).
		MarginBottom(1)

	if m.editingTask != nil {
		b.WriteString(titleStyle.Render("✏️  Edit Task"))
	} else {
		b.WriteString(titleStyle.Render("➕ New Task"))
	}
	b.WriteString("\n\n")

	labels := []string{
		"Content:",
		"Description:",
		"URL:",
		"Priority:",
	}

	for i := range m.inputs {
		label := labels[i]
		if i == m.focusIndex {
			label = focusedStyle.Render(label)
		} else {
			label = blurredStyle.Render(label)
		}

		b.WriteString(label)
		b.WriteString("\n")
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n")

		if i == inputPriority {
			preview := lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.priority.Color())).
				Render("  " + m.priority.String())
			b.WriteString(preview)
			b.WriteString("\n")
		}

		b.WriteString("\n")
	}

	button := blurredButton
	if m.focusIndex == len(m.inputs) {
		button = focusedButton
	}

	b.WriteString("\n")
	b.WriteString(button)
	b.WriteString("  ")
	b.WriteString(cancelButton)
	b.WriteString("\n\n")

	helpText := blurredStyle.Render("Tab: next field • Enter: save • Esc: cancel")
	b.WriteString(helpText)

	return lipgloss.NewStyle().
		Padding(1, 2).
		Render(b.String())
}

func (m formModel) getTask() *Task {
	content := strings.TrimSpace(m.inputs[inputContent].Value())
	if content == "" {
		return nil
	}

	var task Task
	if m.editingTask != nil {
		task = *m.editingTask
	} else {
		task.ID = generateTaskID()
		task.CreatedAt = time.Now()
	}

	task.Content = content
	task.Description = strings.TrimSpace(m.inputs[inputDescription].Value())
	task.URL = strings.TrimSpace(m.inputs[inputURL].Value())
	task.Priority = m.priority

	return &task
}
