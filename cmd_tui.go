package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// --- Styles ---

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(highlight).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Foreground(special).
			Bold(true).
			Padding(0, 1)

	listStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(subtle).
			MarginRight(1)

	detailStyle = lipgloss.NewStyle().
			PaddingLeft(1)

	logStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(subtle)
)

// --- Model ---

type model struct {
	projectName string
	tasks       []TaskView
	cursor      int
	logs        []string
	ready       bool
	viewport    viewport.Model
	logViewport viewport.Model
	width       int
	height      int
	err         error
	logFile     *os.File
	logReader   *bufio.Reader
	dataDir     string
}

type tasksUpdatedMsg []TaskView
type logMsg string
type errMsg error

// --- Init ---

func initialModel(projectName, dataDir string) model {
	m := model{
		projectName: projectName,
		dataDir:     dataDir,
		tasks:       []TaskView{},
		logs:        []string{},
	}

	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.loadTasksCmd(),
		m.tickTasksCmd(),
		m.waitForLogCmd(),
		tea.EnterAltScreen,
	)
}

// --- Update ---

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Layout calculations
		headerHeight := 2  // Header + border
		footerHeight := 10 // Log pane height
		contentHeight := m.height - headerHeight - footerHeight

		if !m.ready {
			m.viewport = viewport.New(m.width/2-2, contentHeight)
			m.logViewport = viewport.New(m.width-2, footerHeight-2)
			m.ready = true
		} else {
			m.viewport.Width = m.width/2 - 2
			m.viewport.Height = contentHeight
			m.logViewport.Width = m.width - 2
			m.logViewport.Height = footerHeight - 2
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}
		}
		// Refresh details on cursor move
		m.viewport.SetContent(m.renderDetails())

	case tasksUpdatedMsg:
		m.tasks = msg
		// Keep cursor in bounds
		if m.cursor >= len(m.tasks) {
			m.cursor = len(m.tasks) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.viewport.SetContent(m.renderDetails())
		// Schedule next update
		cmds = append(cmds, m.tickTasksCmd())

	case logMsg:
		if msg != "" {
			m.logs = append(m.logs, string(msg))
			// Keep only last 100 logs
			if len(m.logs) > 100 {
				m.logs = m.logs[len(m.logs)-100:]
			}
			// Auto-scroll
			m.logViewport.SetContent(strings.Join(m.logs, "\n"))
			m.logViewport.GotoBottom()
		}
		// Listen for next log line
		cmds = append(cmds, m.readNextLogCmd())

	case errMsg:
		m.err = msg
	}

	return m, tea.Batch(cmds...)
}

// --- View ---

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}
	if !m.ready {
		return "Initializing TUI..."
	}

	// Layout
	header := headerStyle.Render(fmt.Sprintf("QuickPlan Control Room: %s", m.projectName))

	// Left Pane: Task List
	taskList := m.renderTaskList()
	leftPane := listStyle.
		Width(m.width/2 - 2).
		Height(m.viewport.Height).
		Render(taskList)

	// Right Pane: Details
	rightPane := detailStyle.
		Width(m.width/2 - 2).
		Height(m.viewport.Height).
		Render(m.viewport.View())

	// Bottom Pane: Logs
	logPane := logStyle.
		Width(m.width - 2).
		Height(m.logViewport.Height + 2).
		Render(m.logViewport.View())

	// Combine
	mainView := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	fullView := lipgloss.JoinVertical(lipgloss.Left, header, mainView, logPane)

	return fullView
}

// --- Render Helpers ---

func (m model) renderTaskList() string {
	var s strings.Builder
	for i, task := range m.tasks {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		statusIcon := "○"
		if task.Status == "DONE" {
			statusIcon = "●"
		} else if task.Status == "IN_PROGRESS" {
			statusIcon = "◐"
		} else if task.Status == "BLOCKED" {
			statusIcon = "✖"
		}

		line := fmt.Sprintf("%s %s %s", cursor, statusIcon, task.Text)
		if m.cursor == i {
			line = lipgloss.NewStyle().Foreground(highlight).Render(line)
		}
		s.WriteString(line + "\n")
	}
	if len(m.tasks) == 0 {
		s.WriteString("No tasks found.")
	}
	return s.String()
}

func (m model) renderDetails() string {
	if len(m.tasks) == 0 {
		return "No task selected."
	}

	if m.cursor >= len(m.tasks) {
		return ""
	}

	task := m.tasks[m.cursor]

	var s strings.Builder
	s.WriteString(fmt.Sprintf("ID: %s\n", task.ID))
	s.WriteString(fmt.Sprintf("Status: %s\n", task.Status))
	s.WriteString(fmt.Sprintf("Assigned To: %s\n", task.AssignedTo))
	s.WriteString("\n--- Behavior ---\n")
	s.WriteString(fmt.Sprintf("Role: %s\n", task.Behavior.Role))
	s.WriteString(fmt.Sprintf("Strategy: %s\n", task.Behavior.Strategy))
	s.WriteString(fmt.Sprintf("Environment: %s (%s)\n", task.Behavior.Environment.Provider, task.Behavior.Environment.Image))

	if len(task.DependsOn) > 0 {
		s.WriteString("\n--- Dependencies ---\n")
		for _, dep := range task.DependsOn {
			s.WriteString(fmt.Sprintf("- %s\n", dep))
		}
	}

	return s.String()
}

// --- Commands ---

func (m model) loadTasksCmd() tea.Cmd {
	return func() tea.Msg {
		projectManager := NewProjectDataManager(m.dataDir, NewVersionManager(version))
		views, _, err := projectManager.GetTaskViews(m.projectName)
		if err != nil {
			return errMsg(err)
		}
		return tasksUpdatedMsg(views)
	}
}

func (m model) tickTasksCmd() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		projectManager := NewProjectDataManager(m.dataDir, NewVersionManager(version))
		views, _, err := projectManager.GetTaskViews(m.projectName)
		if err != nil {
			return errMsg(err)
		}
		return tasksUpdatedMsg(views)
	})
}

// waitForLogCmd opens the file and seeks to end
func (m *model) waitForLogCmd() tea.Cmd {
	return func() tea.Msg {
		logPath := filepath.Join(m.dataDir, "events.jsonl")

		// Wait for file to exist
		for {
			f, err := os.Open(logPath)
			if err == nil {
				m.logFile = f
				m.logFile.Seek(0, io.SeekEnd)
				m.logReader = bufio.NewReader(m.logFile)
				return logMsg("") // Trigger read loop
			}
			time.Sleep(1 * time.Second)
		}
	}
}

// readNextLogCmd blocks until a line is available
func (m *model) readNextLogCmd() tea.Cmd {
	return func() tea.Msg {
		if m.logReader == nil {
			return logMsg("")
		}

		for {
			line, err := m.logReader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					time.Sleep(200 * time.Millisecond)
					continue
				}
				return errMsg(err)
			}

			// Parse
			var event struct {
				Timestamp string `json:"timestamp"`
				Component string `json:"component"`
				Message   string `json:"message"`
			}
			if err := json.Unmarshal([]byte(line), &event); err == nil {
				return logMsg(fmt.Sprintf("[%s] %s: %s", event.Timestamp, event.Component, event.Message))
			}
			return logMsg(line)
		}
	}
}

// --- Main Command ---

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the control room TUI",
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir, err := getDataDir()
		if err != nil {
			return err
		}

		projectName, err := getCurrentProject()
		if err != nil {
			return err
		}

		p := tea.NewProgram(initialModel(projectName, dataDir), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running TUI: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
