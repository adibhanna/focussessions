package menu

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"focussessions/internal/models"
	"focussessions/internal/storage"
)

type MenuChoice int

const (
	ResumeSession MenuChoice = iota
	StartSession
	ViewToday
	ViewWeek
	ViewMonth
	Settings
	Exit
)

type Model struct {
	choices       []string
	cursor        int
	selected      MenuChoice
	storage       *storage.Storage
	config        models.Config
	todayStats    models.DayStats
	activeSession *models.Session
	width         int
	height        int
	shouldQuit    bool
}

func New(storage *storage.Storage) (Model, error) {
	config, err := storage.GetConfig()
	if err != nil {
		return Model{}, err
	}

	todayStats, err := storage.GetDayStats(time.Now().Format("2006-01-02"))
	if err != nil {
		todayStats = models.DayStats{
			Date:          time.Now().Format("2006-01-02"),
			SessionsCount: 0,
			TotalMinutes:  0,
		}
	}

	activeSession, err := storage.GetActiveSession()
	if err != nil {
		// Log error but continue
		activeSession = nil
	}

	choices := []string{}

	// Add resume option if there's an active session
	if activeSession != nil {
		elapsed := activeSession.ElapsedSeconds
		remaining := (activeSession.Duration * 60) - elapsed
		minutes := remaining / 60

		status := "Paused"
		if !activeSession.Paused {
			status = "Active"
		}

		choices = append(choices, fmt.Sprintf("▶️  Resume Session (%s - %d min left)", status, minutes))
	}

	// Always show start new session, but it will check for active sessions
	choices = append(choices,
		"🚀 Start New Focus Session",
		"📊 Today's Progress",
		"📅 This Week's Stats",
		"📈 This Month's Stats",
		"⚙️  Settings",
		"👋 Exit",
	)

	return Model{
		choices:       choices,
		cursor:        0,
		storage:       storage,
		config:        config,
		todayStats:    todayStats,
		activeSession: activeSession,
	}, nil
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.choices) - 1
			}

		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}

		case key.Matches(msg, keys.Enter):
			// Map cursor position to MenuChoice
			if m.activeSession != nil {
				// If there's an active session, the menu has an extra item at the beginning
				m.selected = MenuChoice(m.cursor)
			} else {
				// No active session, so we need to adjust the mapping
				// Skip ResumeSession (0) and map directly to the correct choice
				switch m.cursor {
				case 0:
					m.selected = StartSession
				case 1:
					m.selected = ViewToday
				case 2:
					m.selected = ViewWeek
				case 3:
					m.selected = ViewMonth
				case 4:
					m.selected = Settings
				case 5:
					m.selected = Exit
				}
			}

			if m.selected == Exit {
				m.shouldQuit = true
			}
			return m, tea.Quit

		case key.Matches(msg, keys.Quit):
			m.shouldQuit = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	containerStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF7CCB")).
		MarginBottom(2).
		Align(lipgloss.Center)

	statsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FDFF8C")).
		MarginBottom(2).
		Align(lipgloss.Center)

	menuStyle := lipgloss.NewStyle().
		Padding(1, 2).
		MarginTop(1)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF7CCB")).
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888"))

	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4CAF50")).
		MarginBottom(1).
		Align(lipgloss.Center)

	currentDate := time.Now().Format("Monday, January 2, 2006")
	dateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888")).
		MarginBottom(1).
		Align(lipgloss.Center)

	title := titleStyle.Render("✨ Focus Sessions ✨")
	dateInfo := dateStyle.Render(currentDate)

	stats := statsStyle.Render(fmt.Sprintf(
		"Today: %d sessions | %d mins | Goal: %d sessions",
		m.todayStats.SessionsCount,
		m.todayStats.TotalMinutes,
		m.config.DailySessionGoal,
	))

	progressBar := m.renderProgressBar()

	var activeSessionInfo string
	if m.activeSession != nil && m.cursor != 0 { // Don't show if resume is selected
		elapsed := m.activeSession.ElapsedSeconds / 60
		total := m.activeSession.Duration
		activeSessionInfo = activeStyle.Render(fmt.Sprintf(
			"🔄 Session in progress: %d/%d minutes",
			elapsed,
			total,
		))
	}

	var menu string
	for i, choice := range m.choices {
		cursor := "  "
		style := normalStyle
		if m.cursor == i {
			cursor = "▶ "
			style = selectedStyle
		}
		menu += style.Render(cursor+choice) + "\n"
	}

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		dateInfo,
		stats,
		progressBar,
		activeSessionInfo,
		menuStyle.Render(menu),
		m.renderHelp(),
	)

	return containerStyle.Render(content)
}

func (m Model) renderProgressBar() string {
	progressStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666")).
		MarginBottom(2)

	completed := m.todayStats.SessionsCount
	goal := m.config.DailySessionGoal

	barWidth := 30
	filledWidth := int(float64(completed) / float64(goal) * float64(barWidth))
	if filledWidth > barWidth {
		filledWidth = barWidth
	}

	bar := "["
	for i := 0; i < barWidth; i++ {
		if i < filledWidth {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	bar += "]"

	return progressStyle.Render(bar)
}

func (m Model) renderHelp() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666")).
		MarginTop(2)

	return helpStyle.Render("↑/↓: navigate • enter: select • q: quit")
}

func (m Model) ShouldQuit() bool {
	return m.shouldQuit
}

func (m Model) GetSelected() MenuChoice {
	return m.selected
}

func (m Model) GetActiveSession() *models.Session {
	return m.activeSession
}

type keyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Quit  key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter", "select"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
