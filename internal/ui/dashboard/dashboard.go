package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/adibhanna/focussessions/internal/models"
	"github.com/adibhanna/focussessions/internal/storage"
	"github.com/adibhanna/focussessions/internal/ui/help"
)

type tickMsg time.Time
type exportResultMsg struct {
	success bool
	message string
}
type clearExportMsg struct{}

type ViewState int

const (
	HomeView ViewState = iota
	StatsView
	StatsDetailDaily
	StatsDetailWeekly
	StatsDetailMonthly
	StatsDetailYearly
	HelpView
)

type Model struct {
	storage       *storage.Storage
	config        models.Config
	todayStats    models.DayStats
	weekStats     models.WeekStats
	monthStats    models.MonthStats
	yearStats     models.YearStats
	activeSession *models.Session
	viewState     ViewState
	width         int
	height        int

	// Timer state
	timerRunning  bool
	timerPaused   bool
	timerElapsed  int
	timerDuration int
	timerProgress progress.Model

	// Sub-models
	helpModel help.Model

	// Export state
	exportMessage string
	showExportMsg bool

	shouldQuit   bool
	openSettings bool
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

	now := time.Now()
	_, week := now.ISOWeek()
	weekStats, err := storage.GetWeekStats(now.Year(), week)
	if err != nil {
		weekStats = models.WeekStats{
			Week:          week,
			Year:          now.Year(),
			SessionsCount: 0,
			TotalMinutes:  0,
		}
	}

	monthStats, err := storage.GetMonthStats(now.Year(), int(now.Month()))
	if err != nil {
		monthStats = models.MonthStats{
			Month:         now.Format("2006-01"),
			Year:          now.Year(),
			SessionsCount: 0,
			TotalMinutes:  0,
		}
	}

	yearStats, err := storage.GetYearStats(now.Year())
	if err != nil {
		yearStats = models.YearStats{
			Year:          now.Year(),
			SessionsCount: 0,
			TotalMinutes:  0,
		}
	}

	activeSession, err := storage.GetActiveSession()
	if err != nil {
		activeSession = nil
	}

	prog := progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	prog.Width = 40

	m := Model{
		storage:       storage,
		config:        config,
		todayStats:    todayStats,
		weekStats:     weekStats,
		monthStats:    monthStats,
		yearStats:     yearStats,
		activeSession: activeSession,
		viewState:     HomeView,
		timerProgress: prog,
		timerDuration: config.SessionDuration * 60,
		helpModel:     help.New(),
	}

	// If there's an active session, set up timer state
	if activeSession != nil {
		m.timerRunning = true
		m.timerPaused = activeSession.Paused
		m.timerDuration = activeSession.Duration * 60

		// Calculate elapsed time including time passed while app was closed
		if !activeSession.Paused {
			// If session wasn't paused, add the time that passed since last save
			timeSinceStart := time.Since(activeSession.StartTime)
			m.timerElapsed = int(timeSinceStart.Seconds())

			// Ensure we don't exceed the duration
			if m.timerElapsed > m.timerDuration {
				m.timerElapsed = m.timerDuration
			}
		} else {
			// If paused, use the saved elapsed time
			m.timerElapsed = activeSession.ElapsedSeconds
		}
	}

	return m, nil
}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	// Start the tick if timer is running
	if m.activeSession != nil && m.timerRunning && !m.timerPaused {
		cmds = append(cmds, tickCmd())
	}

	// Start progress bar animation
	cmds = append(cmds, m.timerProgress.Init())

	return tea.Batch(cmds...)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) exportStats() tea.Cmd {
	return func() tea.Msg {
		report, err := m.storage.ExportAllStats()
		if err != nil {
			return exportResultMsg{success: false, message: fmt.Sprintf("Export failed: %v", err)}
		}

		// Save to file
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return exportResultMsg{success: false, message: fmt.Sprintf("Failed to get home directory: %v", err)}
		}

		timestamp := time.Now().Format("2006-01-02-150405")
		filename := fmt.Sprintf("focussessions-stats-%s.txt", timestamp)
		filePath := filepath.Join(homeDir, "Downloads", filename)

		err = os.WriteFile(filePath, []byte(report), 0644)
		if err != nil {
			// Try alternative location if Downloads doesn't exist
			filePath = filepath.Join(homeDir, filename)
			err = os.WriteFile(filePath, []byte(report), 0644)
			if err != nil {
				return exportResultMsg{success: false, message: fmt.Sprintf("Failed to save file: %v", err)}
			}
		}

		return exportResultMsg{success: true, message: fmt.Sprintf("[OK] Exported to %s", filePath)}
	}
}

func (m Model) clearExportMsgAfterDelay() tea.Cmd {
	return tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
		return clearExportMsg{}
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.timerProgress.Width = min(msg.Width/3-10, 40)
		if m.viewState == HelpView {
			helpModel, _ := m.helpModel.Update(msg)
			m.helpModel = helpModel.(help.Model)
		}
		return m, nil

	case tea.KeyMsg:
		// Handle help view specially
		if m.viewState == HelpView {
			helpModel, _ := m.helpModel.Update(msg)
			m.helpModel = helpModel.(help.Model)
			if m.helpModel.ShouldQuit() {
				m.viewState = HomeView
			}
			// Don't process other keys when in help view, but don't break tick chain
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Quit):
			if m.timerRunning && m.activeSession != nil {
				// Save state when quitting
				m.activeSession.ElapsedSeconds = m.timerElapsed
				m.activeSession.Paused = m.timerPaused
				m.storage.SaveSession(*m.activeSession)
			}
			m.shouldQuit = true
			return m, tea.Quit

		case key.Matches(msg, keys.Home):
			m.viewState = HomeView
			return m, nil

		case key.Matches(msg, keys.Back):
			switch m.viewState {
			case StatsDetailDaily, StatsDetailWeekly, StatsDetailMonthly, StatsDetailYearly:
				// From detail views, go back to stats overview
				m.viewState = StatsView
			case StatsView:
				// From stats overview, go back to home
				m.viewState = HomeView
			case HelpView:
				// From help view, go back to home
				m.viewState = HomeView
			default:
				// From home or other views, do nothing (already at top level)
			}
			return m, nil

		case key.Matches(msg, keys.Help):
			m.viewState = HelpView
			// Ensure help model gets window size
			if m.width > 0 && m.height > 0 {
				sizeMsg := tea.WindowSizeMsg{Width: m.width, Height: m.height}
				helpModel, _ := m.helpModel.Update(sizeMsg)
				m.helpModel = helpModel.(help.Model)
			}
			return m, nil

		case key.Matches(msg, keys.Stats):
			if m.viewState == StatsView {
				// Toggle back to home if already in stats view
				m.viewState = HomeView
			} else {
				m.viewState = StatsView
				// Refresh all stats
				now := time.Now()

				// Refresh daily stats
				todayStats, err := m.storage.GetDayStats(now.Format("2006-01-02"))
				if err == nil {
					m.todayStats = todayStats
				}

				// Refresh weekly stats
				_, week := now.ISOWeek()
				weekStats, err := m.storage.GetWeekStats(now.Year(), week)
				if err == nil {
					m.weekStats = weekStats
				}

				// Refresh monthly stats
				monthStats, err := m.storage.GetMonthStats(now.Year(), int(now.Month()))
				if err == nil {
					m.monthStats = monthStats
				}

				// Refresh yearly stats
				yearStats, err := m.storage.GetYearStats(now.Year())
				if err == nil {
					m.yearStats = yearStats
				}
			}
			return m, nil

		// Drill-down navigation (only available in stats view)
		case key.Matches(msg, keys.Daily) && m.viewState == StatsView:
			m.viewState = StatsDetailDaily
			return m, nil

		case key.Matches(msg, keys.Weekly) && m.viewState == StatsView:
			m.viewState = StatsDetailWeekly
			return m, nil

		case key.Matches(msg, keys.Monthly) && m.viewState == StatsView:
			m.viewState = StatsDetailMonthly
			return m, nil

		case key.Matches(msg, keys.Yearly) && m.viewState == StatsView:
			m.viewState = StatsDetailYearly
			return m, nil

		case key.Matches(msg, keys.Start) && !m.timerRunning:
			return m.startNewSession()

		case key.Matches(msg, keys.Pause) && m.timerRunning && !m.timerPaused:
			m.timerPaused = true
			if m.activeSession != nil {
				m.activeSession.Paused = true
				m.activeSession.ElapsedSeconds = m.timerElapsed
				m.storage.SaveSession(*m.activeSession)
			}
			return m, nil

		case key.Matches(msg, keys.Resume) && m.timerRunning && m.timerPaused:
			m.timerPaused = false
			if m.activeSession != nil {
				m.activeSession.Paused = false
				m.storage.SaveSession(*m.activeSession)
			}
			return m, tickCmd()

		case key.Matches(msg, keys.Cancel) && m.timerRunning:
			return m.cancelSession()

		case key.Matches(msg, keys.Settings):
			m.openSettings = true
			return m, tea.Quit

		case key.Matches(msg, keys.Export):
			// Only allow export in stats views
			if m.viewState == StatsView || m.viewState == StatsDetailDaily ||
				m.viewState == StatsDetailWeekly || m.viewState == StatsDetailMonthly ||
				m.viewState == StatsDetailYearly {
				return m, m.exportStats()
			}
		}

	case tickMsg:
		if m.timerRunning && !m.timerPaused {
			m.timerElapsed++

			// Save progress periodically
			if m.timerElapsed%10 == 0 && m.activeSession != nil {
				m.activeSession.ElapsedSeconds = m.timerElapsed
				m.storage.SaveSession(*m.activeSession)
			}

			// Check if session is complete
			if m.timerElapsed >= m.timerDuration {
				return m.completeSession()
			}

			return m, tickCmd()
		}
		// If timer is paused or not running, don't continue ticking
		return m, nil

	case progress.FrameMsg:
		progressModel, cmd := m.timerProgress.Update(msg)
		m.timerProgress = progressModel.(progress.Model)
		// Don't break the chain - the tick and progress should work independently
		return m, cmd

	case exportResultMsg:
		m.exportMessage = msg.message
		m.showExportMsg = true
		return m, m.clearExportMsgAfterDelay()

	case clearExportMsg:
		m.showExportMsg = false
		m.exportMessage = ""
		return m, nil
	}

	return m, nil
}

func (m Model) startNewSession() (tea.Model, tea.Cmd) {
	// Deactivate any existing sessions
	m.storage.DeactivateAllSessions()

	// Create new session
	session := &models.Session{
		ID:             uuid.New().String(),
		StartTime:      time.Now(),
		Duration:       m.config.SessionDuration,
		Date:           time.Now().Format("2006-01-02"),
		Week:           getWeekNumber(time.Now()),
		Month:          time.Now().Format("2006-01"),
		Year:           time.Now().Year(),
		Active:         true,
		ElapsedSeconds: 0,
		Paused:         false,
	}

	m.storage.SaveSession(*session)

	// Update timer state
	m.activeSession = session
	m.timerRunning = true
	m.timerPaused = false
	m.timerElapsed = 0
	m.timerDuration = m.config.SessionDuration * 60

	return m, tickCmd()
}

func (m Model) cancelSession() (tea.Model, tea.Cmd) {
	if m.activeSession != nil {
		m.activeSession.EndTime = time.Now()
		m.activeSession.Completed = false
		m.activeSession.Active = false
		m.activeSession.ElapsedSeconds = m.timerElapsed
		m.storage.SaveSession(*m.activeSession)
	}

	// Reset timer state
	m.activeSession = nil
	m.timerRunning = false
	m.timerPaused = false
	m.timerElapsed = 0

	// Refresh stats
	todayStats, _ := m.storage.GetDayStats(time.Now().Format("2006-01-02"))
	m.todayStats = todayStats

	return m, nil
}

func (m Model) completeSession() (tea.Model, tea.Cmd) {
	if m.activeSession != nil {
		m.activeSession.EndTime = time.Now()
		m.activeSession.Completed = true
		m.activeSession.Active = false
		m.activeSession.ElapsedSeconds = m.timerElapsed
		m.storage.SaveSession(*m.activeSession)
	}

	// Reset timer state
	m.activeSession = nil
	m.timerRunning = false
	m.timerPaused = false
	m.timerElapsed = 0

	// Refresh stats
	todayStats, _ := m.storage.GetDayStats(time.Now().Format("2006-01-02"))
	m.todayStats = todayStats

	now := time.Now()
	_, week := now.ISOWeek()
	weekStats, _ := m.storage.GetWeekStats(now.Year(), week)
	m.weekStats = weekStats

	// Check if daily goal is met
	if m.todayStats.SessionsCount >= m.config.DailySessionGoal {
		return m, tea.Printf("*** DAILY GOAL ACHIEVED! You completed %d/%d sessions! ***",
			m.todayStats.SessionsCount, m.config.DailySessionGoal)
	}

	return m, tea.Printf("*** Session completed! Great job! ***")
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	switch m.viewState {
	case StatsView:
		return m.renderStatsView()
	case StatsDetailDaily:
		return m.renderDailyDetailView()
	case StatsDetailWeekly:
		return m.renderWeeklyDetailView()
	case StatsDetailMonthly:
		return m.renderMonthlyDetailView()
	case StatsDetailYearly:
		return m.renderYearlyDetailView()
	case HelpView:
		return m.helpModel.View()
	default:
		return m.renderHomeView()
	}
}

func (m Model) renderHomeView() string {
	containerStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Padding(4)

	// Main timer section - larger and centered
	timerSection := m.renderCenterTimer()

	// Simple progress indicator
	progressSection := m.renderSimpleProgress()

	// Help at bottom
	help := m.renderHelp()

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		timerSection,
		progressSection,
		help,
	)

	return containerStyle.Render(content)
}

func (m Model) renderCenterTimer() string {
	timerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(2, 4).
		Align(lipgloss.Center).
		MarginBottom(3)

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888")).
		Align(lipgloss.Center).
		MarginBottom(2)

	var timerDisplay, status, progressBar string

	if m.timerRunning {
		remaining := m.timerDuration - m.timerElapsed
		minutes := remaining / 60
		seconds := remaining % 60

		// Create large ASCII art style numbers
		bigTime := m.renderBigTime(minutes, seconds)
		timerDisplay = timerStyle.Render(bigTime)

		percent := float64(m.timerElapsed) / float64(m.timerDuration)
		progressWidth := 60
		m.timerProgress.Width = progressWidth
		progressBar = m.timerProgress.ViewAs(percent)

		if m.timerPaused {
			status = statusStyle.Render("⏸️  Session Paused")
		} else {
			status = statusStyle.Render("🎯 Stay Focused!")
		}
	} else {
		timerDisplay = timerStyle.Render("Ready to Focus")
		progressWidth := 60
		m.timerProgress.Width = progressWidth
		progressBar = m.timerProgress.ViewAs(0)
		status = statusStyle.Render("Press 's' to start a session")
	}

	return lipgloss.JoinVertical(
		lipgloss.Center,
		timerDisplay,
		progressBar,
		status,
	)
}

func (m Model) renderBigTime(minutes, seconds int) string {
	// ASCII art for digits 0-9
	digits := map[int][]string{
		0: {"███", "█ █", "█ █", "█ █", "███"},
		1: {" █ ", "██ ", " █ ", " █ ", "███"},
		2: {"███", "  █", "███", "█  ", "███"},
		3: {"███", "  █", "███", "  █", "███"},
		4: {"█ █", "█ █", "███", "  █", "  █"},
		5: {"███", "█  ", "███", "  █", "███"},
		6: {"███", "█  ", "███", "█ █", "███"},
		7: {"███", "  █", "  █", "  █", "  █"},
		8: {"███", "█ █", "███", "█ █", "███"},
		9: {"███", "█ █", "███", "  █", "███"},
	}

	colon := []string{" ", "█", " ", "█", " "}

	m1 := minutes / 10
	m2 := minutes % 10
	s1 := seconds / 10
	s2 := seconds % 10

	var lines []string
	for row := 0; row < 5; row++ {
		line := digits[m1][row] + " " + digits[m2][row] + " " + colon[row] + " " + digits[s1][row] + " " + digits[s2][row]
		lines = append(lines, line)
	}

	return lipgloss.JoinVertical(lipgloss.Center, lines...)
}

func (m Model) renderSimpleProgress() string {
	progressStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FDFF8C")).
		Align(lipgloss.Center).
		MarginTop(2)

	dateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888")).
		Align(lipgloss.Center).
		MarginBottom(1)

	completed := m.todayStats.SessionsCount
	goal := m.config.DailySessionGoal

	currentDate := time.Now().Format("Monday, January 2, 2006")
	progressText := fmt.Sprintf(
		"Today: %d/%d sessions • %dm",
		completed,
		goal,
		m.todayStats.TotalMinutes,
	)

	// Simple progress bar
	barWidth := 40
	filledWidth := int(float64(completed) / float64(goal) * float64(barWidth))
	if filledWidth > barWidth {
		filledWidth = barWidth
	}

	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filledWidth {
			bar += "■"
		} else {
			bar += "□"
		}
	}

	return lipgloss.JoinVertical(
		lipgloss.Center,
		dateStyle.Render(currentDate),
		progressStyle.Render(progressText),
		progressStyle.Render(bar),
	)
}

func (m Model) renderTimerSection(width int) string {
	sectionStyle := lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF7CCB")).
		Padding(1).
		MarginRight(1)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF7CCB")).
		Align(lipgloss.Center).
		MarginBottom(1)

	timerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(1, 2).
		Align(lipgloss.Center).
		MarginBottom(1)

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888")).
		Align(lipgloss.Center).
		MarginBottom(1)

	title := titleStyle.Render("🚀 Focus Timer")

	var timerDisplay, status, progressBar string

	if m.timerRunning {
		remaining := m.timerDuration - m.timerElapsed
		minutes := remaining / 60
		seconds := remaining % 60
		timerDisplay = timerStyle.Render(fmt.Sprintf("%02d:%02d", minutes, seconds))

		percent := float64(m.timerElapsed) / float64(m.timerDuration)
		progressBar = m.timerProgress.ViewAs(percent)

		if m.timerPaused {
			status = statusStyle.Render("⏸️  PAUSED")
		} else {
			status = statusStyle.Render("▶️  RUNNING")
		}
	} else {
		timerDisplay = timerStyle.Render("Ready")
		progressBar = m.timerProgress.ViewAs(0)
		status = statusStyle.Render("Press 's' to start")
	}

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		timerDisplay,
		progressBar,
		status,
	)

	return sectionStyle.Render(content)
}

func (m Model) renderDailySection(width int) string {
	sectionStyle := lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FDFF8C")).
		Padding(1).
		MarginRight(1)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FDFF8C")).
		Align(lipgloss.Center).
		MarginBottom(1)

	statsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888")).
		Align(lipgloss.Center).
		MarginBottom(1)

	title := titleStyle.Render("📊 Today's Progress")

	hoursWorked := float64(m.todayStats.TotalMinutes) / 60.0
	stats := fmt.Sprintf(
		"Sessions: %d / %d\nTime: %.1fh",
		m.todayStats.SessionsCount,
		m.config.DailySessionGoal,
		hoursWorked,
	)

	// Progress bar
	completed := m.todayStats.SessionsCount
	goal := m.config.DailySessionGoal
	barWidth := width - 6
	if barWidth < 10 {
		barWidth = 10
	}

	filledWidth := int(float64(completed) / float64(goal) * float64(barWidth))
	if filledWidth > barWidth {
		filledWidth = barWidth
	}

	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filledWidth {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	progressStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FDFF8C")).
		Align(lipgloss.Center)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		statsStyle.Render(stats),
		progressStyle.Render(bar),
	)

	return sectionStyle.Render(content)
}

func (m Model) renderWeeklySection(width int) string {
	sectionStyle := lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4CAF50")).
		Padding(1)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#4CAF50")).
		Align(lipgloss.Center).
		MarginBottom(1)

	statsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888")).
		Align(lipgloss.Center)

	title := titleStyle.Render("📅 This Week")

	hoursWorked := float64(m.weekStats.TotalMinutes) / 60.0
	stats := fmt.Sprintf(
		"Week %d\nSessions: %d\nTime: %.1fh",
		m.weekStats.Week,
		m.weekStats.SessionsCount,
		hoursWorked,
	)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		statsStyle.Render(stats),
	)

	return sectionStyle.Render(content)
}

func (m Model) renderDailyView() string {
	// Full screen daily view with session details
	containerStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF7CCB")).
		MarginBottom(2).
		Align(lipgloss.Center)

	date, _ := time.Parse("2006-01-02", m.todayStats.Date)
	title := titleStyle.Render(fmt.Sprintf("📊 Daily Stats - %s", date.Format("Monday, January 2, 2006")))

	statsSection := m.renderDailyStatsDetail()
	help := m.renderHelp()

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		statsSection,
		help,
	)

	return containerStyle.Render(content)
}

func (m Model) renderWeeklyView() string {
	// Full screen weekly view with daily breakdown
	containerStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF7CCB")).
		MarginBottom(2).
		Align(lipgloss.Center)

	title := titleStyle.Render(fmt.Sprintf("📅 Weekly Stats - Week %d, %d", m.weekStats.Week, m.weekStats.Year))

	statsSection := m.renderWeeklyStatsDetail()
	help := m.renderHelp()

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		statsSection,
		help,
	)

	return containerStyle.Render(content)
}

func (m Model) renderDailyStatsDetail() string {
	statsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FDFF8C")).
		MarginBottom(1)

	sessionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888")).
		PaddingLeft(2)

	stats := statsStyle.Render(fmt.Sprintf(
		"Completed Sessions: %d | Actual Time: %d mins",
		m.todayStats.SessionsCount,
		m.todayStats.TotalMinutes,
	))

	var sessions string
	if len(m.todayStats.Sessions) == 0 {
		sessions = sessionStyle.Render("No sessions yet today. Time to focus! 🚀")
	} else {
		sessions = "\nSession History:\n"
		for i, session := range m.todayStats.Sessions {
			var status string
			var sessionInfo string

			if session.Active {
				elapsed := session.ElapsedSeconds / 60
				if session.Paused {
					status = "⏸️"
					sessionInfo = fmt.Sprintf(
						"%s Session %d: In Progress (Paused) - %d/%d min",
						status, i+1, elapsed, session.Duration,
					)
				} else {
					status = "▶️"
					sessionInfo = fmt.Sprintf(
						"%s Session %d: In Progress - %d/%d min",
						status, i+1, elapsed, session.Duration,
					)
				}
			} else if session.Completed {
				status = "✅"
				// Calculate actual time spent
				actualDuration := session.ElapsedSeconds / 60
				if actualDuration == 0 && !session.EndTime.IsZero() && !session.StartTime.IsZero() {
					// Fallback to time difference if ElapsedSeconds not set
					actualDuration = int(session.EndTime.Sub(session.StartTime).Minutes())
				}
				// For completed sessions, use actual duration or planned duration as fallback
				if actualDuration == 0 {
					actualDuration = session.Duration
				}

				sessionInfo = fmt.Sprintf(
					"%s Session %d: %s - %s (%d min)",
					status, i+1,
					session.StartTime.Format("3:04 PM"),
					session.EndTime.Format("3:04 PM"),
					actualDuration,
				)
			} else {
				status = "⚠️"
				actualDuration := session.ElapsedSeconds / 60
				if actualDuration == 0 && !session.EndTime.IsZero() && !session.StartTime.IsZero() {
					actualDuration = int(session.EndTime.Sub(session.StartTime).Minutes())
				}

				if actualDuration > 0 {
					sessionInfo = fmt.Sprintf(
						"%s Session %d: Stopped early - %s (%d of %d min)",
						status, i+1,
						session.StartTime.Format("3:04 PM"),
						actualDuration, session.Duration,
					)
				} else {
					sessionInfo = fmt.Sprintf(
						"%s Session %d: Cancelled - %s (0 min)",
						status, i+1,
						session.StartTime.Format("3:04 PM"),
					)
				}
			}
			sessions += sessionStyle.Render(sessionInfo) + "\n"
		}
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		stats,
		sessions,
	)
}

func (m Model) renderWeeklyStatsDetail() string {
	statsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FDFF8C")).
		MarginBottom(1)

	dayStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888")).
		PaddingLeft(2)

	hours := m.weekStats.TotalMinutes / 60
	mins := m.weekStats.TotalMinutes % 60
	var timeStr string
	if hours > 0 {
		if mins > 0 {
			timeStr = fmt.Sprintf("%dh %dm", hours, mins)
		} else {
			timeStr = fmt.Sprintf("%dh", hours)
		}
	} else {
		timeStr = fmt.Sprintf("%dm", mins)
	}

	stats := statsStyle.Render(fmt.Sprintf(
		"Completed Sessions: %d | Actual Time: %s",
		m.weekStats.SessionsCount,
		timeStr,
	))

	var days string
	if len(m.weekStats.DailyStats) == 0 {
		days = dayStyle.Render("No sessions this week yet. Let's get started! 💪")
	} else {
		days = "\nDaily Breakdown:\n"
		for _, day := range m.weekStats.DailyStats {
			date, _ := time.Parse("2006-01-02", day.Date)

			hours := day.TotalMinutes / 60
			mins := day.TotalMinutes % 60
			var timeStr string
			if hours > 0 {
				if mins > 0 {
					timeStr = fmt.Sprintf("%dh %dm", hours, mins)
				} else {
					timeStr = fmt.Sprintf("%dh", hours)
				}
			} else {
				timeStr = fmt.Sprintf("%dm", mins)
			}

			dayInfo := fmt.Sprintf(
				"%s: %d sessions (%s)",
				date.Format("Monday"),
				day.SessionsCount,
				timeStr,
			)
			days += dayStyle.Render(dayInfo) + "\n"
		}
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		stats,
		days,
	)
}

func (m Model) renderStatsView() string {
	containerStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF7CCB")).
		MarginBottom(1).
		Align(lipgloss.Center)

	dateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888")).
		MarginBottom(2).
		Align(lipgloss.Center)

	sectionStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#666")).
		Padding(0, 1).
		MarginRight(1).
		MarginBottom(1)

	currentYear := time.Now().Year()
	currentDate := time.Now().Format("Monday, January 2, 2006")

	title := titleStyle.Render(fmt.Sprintf("📊 Statistics Overview - %d", currentYear))
	dateInfo := dateStyle.Render(currentDate)

	// Create four sections
	dailySection := m.renderDailySummary()
	weeklySection := m.renderWeeklySummary()
	monthlySection := m.renderMonthlySummary()
	yearlySection := m.renderYearlySummary()

	// Calculate available width after container padding
	availableWidth := m.width - 4 // Account for container padding

	var content string
	if availableWidth > 200 {
		// Very wide screen - show four columns
		// Account for borders (2 chars each) and gaps between sections
		colWidth := (availableWidth - 12) / 4 // 3 gaps * 4 chars for borders
		content = lipgloss.JoinHorizontal(
			lipgloss.Top,
			sectionStyle.Width(colWidth).Render(dailySection),
			sectionStyle.Width(colWidth).Render(weeklySection),
			sectionStyle.Width(colWidth).Render(monthlySection),
			sectionStyle.Width(colWidth).Render(yearlySection),
		)
	} else if availableWidth > 100 {
		// Medium screen - show 2x2 grid
		colWidth := (availableWidth - 6) / 2 // 1 gap, 4 chars for borders
		row1 := lipgloss.JoinHorizontal(
			lipgloss.Top,
			sectionStyle.Width(colWidth).Render(dailySection),
			sectionStyle.Width(colWidth).Render(weeklySection),
		)
		row2 := lipgloss.JoinHorizontal(
			lipgloss.Top,
			sectionStyle.Width(colWidth).Render(monthlySection),
			sectionStyle.Width(colWidth).Render(yearlySection),
		)
		content = lipgloss.JoinVertical(lipgloss.Left, row1, row2)
	} else if availableWidth > 60 {
		// Narrow screen - stack in pairs
		row1 := lipgloss.JoinVertical(
			lipgloss.Left,
			sectionStyle.Width(availableWidth-2).Render(dailySection),
			sectionStyle.Width(availableWidth-2).Render(weeklySection),
		)
		row2 := lipgloss.JoinVertical(
			lipgloss.Left,
			sectionStyle.Width(availableWidth-2).Render(monthlySection),
			sectionStyle.Width(availableWidth-2).Render(yearlySection),
		)
		content = lipgloss.JoinVertical(lipgloss.Left, row1, row2)
	} else {
		// Very narrow screen - show only today and week
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			sectionStyle.Width(availableWidth-2).Render(dailySection),
			sectionStyle.Width(availableWidth-2).Render(weeklySection),
		)
	}

	help := m.renderHelp()

	fullContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		dateInfo,
		content,
		help,
	)

	return containerStyle.Render(fullContent)
}

func (m Model) renderDailySummary() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FDFF8C"))

	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888"))

	date := time.Now().Format("Monday, Jan 2")
	title := titleStyle.Render("📅 " + date)

	goalText := "sessions"
	if m.config.DailySessionGoal == 1 {
		goalText = "session"
	}
	content := contentStyle.Render(fmt.Sprintf(
		"\nSessions: %d\nTime: %dm\nGoal: %d %s",
		m.todayStats.SessionsCount,
		m.todayStats.TotalMinutes,
		m.config.DailySessionGoal,
		goalText,
	))

	return title + content
}

func (m Model) renderWeeklySummary() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#4CAF50"))

	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888"))

	hours := m.weekStats.TotalMinutes / 60
	mins := m.weekStats.TotalMinutes % 60
	var timeStr string
	if hours > 0 {
		if mins > 0 {
			timeStr = fmt.Sprintf("%dh %dm", hours, mins)
		} else {
			timeStr = fmt.Sprintf("%dh", hours)
		}
	} else {
		timeStr = fmt.Sprintf("%dm", mins)
	}

	title := titleStyle.Render(fmt.Sprintf("📅 Week %d", m.weekStats.Week))

	content := contentStyle.Render(fmt.Sprintf(
		"\nSessions: %d\nTime: %s\nAvg/day: %.1f",
		m.weekStats.SessionsCount,
		timeStr,
		float64(m.weekStats.SessionsCount)/7.0,
	))

	return title + content
}

func (m Model) renderMonthlySummary() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF6B6B"))

	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888"))

	hours := m.monthStats.TotalMinutes / 60
	mins := m.monthStats.TotalMinutes % 60
	var timeStr string
	if hours > 0 {
		if mins > 0 {
			timeStr = fmt.Sprintf("%dh %dm", hours, mins)
		} else {
			timeStr = fmt.Sprintf("%dh", hours)
		}
	} else {
		timeStr = fmt.Sprintf("%dm", mins)
	}

	monthTime, _ := time.Parse("2006-01", m.monthStats.Month)
	title := titleStyle.Render("📈 " + monthTime.Format("January"))

	content := contentStyle.Render(fmt.Sprintf(
		"\nSessions: %d\nTime: %s\nAvg/day: %.1f",
		m.monthStats.SessionsCount,
		timeStr,
		float64(m.monthStats.SessionsCount)/30.0,
	))

	return title + content
}

func (m Model) renderMonthlyView() string {
	containerStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF7CCB")).
		MarginBottom(2).
		Align(lipgloss.Center)

	monthTime, _ := time.Parse("2006-01", m.monthStats.Month)
	title := titleStyle.Render(fmt.Sprintf("📈 Monthly Stats - %s", monthTime.Format("January 2006")))

	statsSection := m.renderMonthlyStatsDetail()
	help := m.renderHelp()

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		statsSection,
		help,
	)

	return containerStyle.Render(content)
}

func (m Model) renderMonthlyStatsDetail() string {
	statsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FDFF8C")).
		MarginBottom(1)

	weekStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888")).
		PaddingLeft(2)

	hours := m.monthStats.TotalMinutes / 60
	mins := m.monthStats.TotalMinutes % 60
	var timeStr string
	if hours > 0 {
		if mins > 0 {
			timeStr = fmt.Sprintf("%dh %dm", hours, mins)
		} else {
			timeStr = fmt.Sprintf("%dh", hours)
		}
	} else {
		timeStr = fmt.Sprintf("%dm", mins)
	}

	stats := statsStyle.Render(fmt.Sprintf(
		"Total Sessions: %d | Total Time: %s",
		m.monthStats.SessionsCount,
		timeStr,
	))

	avgPerDay := float64(m.monthStats.SessionsCount) / 30.0
	avgStats := statsStyle.Render(fmt.Sprintf(
		"Average: %.1f sessions per day",
		avgPerDay,
	))

	var weeks string
	if len(m.monthStats.WeeklyStats) == 0 {
		weeks = weekStyle.Render("No sessions this month yet. Time to build momentum! 🎯")
	} else {
		weeks = "\nWeekly Breakdown:\n"
		for _, week := range m.monthStats.WeeklyStats {
			hours := week.TotalMinutes / 60
			mins := week.TotalMinutes % 60
			var weekTimeStr string
			if hours > 0 {
				if mins > 0 {
					weekTimeStr = fmt.Sprintf("%dh %dm", hours, mins)
				} else {
					weekTimeStr = fmt.Sprintf("%dh", hours)
				}
			} else {
				weekTimeStr = fmt.Sprintf("%dm", mins)
			}

			weekInfo := fmt.Sprintf(
				"Week %d: %d sessions (%s)",
				week.Week,
				week.SessionsCount,
				weekTimeStr,
			)
			weeks += weekStyle.Render(weekInfo) + "\n"
		}
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		stats,
		avgStats,
		weeks,
	)
}

func (m Model) renderDailyDetailView() string {
	containerStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF7CCB")).
		MarginBottom(2).
		Align(lipgloss.Center)

	date, _ := time.Parse("2006-01-02", m.todayStats.Date)
	title := titleStyle.Render(fmt.Sprintf("📅 Daily Details - %s", date.Format("Monday, January 2, 2006")))

	statsSection := m.renderDailyStatsDetail()
	help := m.renderHelp()

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		statsSection,
		help,
	)

	return containerStyle.Render(content)
}

func (m Model) renderWeeklyDetailView() string {
	containerStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF7CCB")).
		MarginBottom(2).
		Align(lipgloss.Center)

	title := titleStyle.Render(fmt.Sprintf("📅 Weekly Details - Week %d, %d", m.weekStats.Week, m.weekStats.Year))

	statsSection := m.renderWeeklyStatsDetail()
	help := m.renderHelp()

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		statsSection,
		help,
	)

	return containerStyle.Render(content)
}

func (m Model) renderYearlySummary() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00BFFF"))

	contentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888"))

	hours := m.yearStats.TotalMinutes / 60
	mins := m.yearStats.TotalMinutes % 60
	var timeStr string
	if hours > 0 {
		if mins > 0 {
			timeStr = fmt.Sprintf("%dh %dm", hours, mins)
		} else {
			timeStr = fmt.Sprintf("%dh", hours)
		}
	} else {
		timeStr = fmt.Sprintf("%dm", mins)
	}

	title := titleStyle.Render(fmt.Sprintf("📊 Year %d", m.yearStats.Year))

	content := contentStyle.Render(fmt.Sprintf(
		"\nSessions: %d\nTime: %s\nAvg/month: %.1f",
		m.yearStats.SessionsCount,
		timeStr,
		float64(m.yearStats.SessionsCount)/12.0,
	))

	return title + content
}

func (m Model) renderMonthlyDetailView() string {
	containerStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF7CCB")).
		MarginBottom(2).
		Align(lipgloss.Center)

	monthTime, _ := time.Parse("2006-01", m.monthStats.Month)
	title := titleStyle.Render(fmt.Sprintf("📈 Monthly Details - %s", monthTime.Format("January 2006")))

	statsSection := m.renderMonthlyStatsDetail()
	help := m.renderHelp()

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		statsSection,
		help,
	)

	return containerStyle.Render(content)
}

func (m Model) renderYearlyDetailView() string {
	containerStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF7CCB")).
		MarginBottom(2).
		Align(lipgloss.Center)

	title := titleStyle.Render(fmt.Sprintf("📊 Yearly Details - %d", m.yearStats.Year))

	statsSection := m.renderYearlyStatsDetail()
	help := m.renderHelp()

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		statsSection,
		help,
	)

	return containerStyle.Render(content)
}

func (m Model) renderYearlyStatsDetail() string {
	statsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FDFF8C")).
		MarginBottom(1)

	monthStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888")).
		PaddingLeft(2)

	hours := m.yearStats.TotalMinutes / 60
	mins := m.yearStats.TotalMinutes % 60
	var timeStr string
	if hours > 0 {
		if mins > 0 {
			timeStr = fmt.Sprintf("%dh %dm", hours, mins)
		} else {
			timeStr = fmt.Sprintf("%dh", hours)
		}
	} else {
		timeStr = fmt.Sprintf("%dm", mins)
	}

	stats := statsStyle.Render(fmt.Sprintf(
		"Total Sessions: %d | Total Time: %s",
		m.yearStats.SessionsCount,
		timeStr,
	))

	avgPerDay := float64(m.yearStats.SessionsCount) / 365.0
	avgPerMonth := float64(m.yearStats.SessionsCount) / 12.0
	avgStats := statsStyle.Render(fmt.Sprintf(
		"Average: %.1f sessions per day | %.1f sessions per month",
		avgPerDay,
		avgPerMonth,
	))

	var months string
	if len(m.yearStats.MonthlyStats) == 0 {
		months = monthStyle.Render("No sessions this year yet. Time to get started! 🎯")
	} else {
		months = "\nMonthly Breakdown:\n"
		for _, month := range m.yearStats.MonthlyStats {
			hours := month.TotalMinutes / 60
			mins := month.TotalMinutes % 60
			var monthTimeStr string
			if hours > 0 {
				if mins > 0 {
					monthTimeStr = fmt.Sprintf("%dh %dm", hours, mins)
				} else {
					monthTimeStr = fmt.Sprintf("%dh", hours)
				}
			} else {
				monthTimeStr = fmt.Sprintf("%dm", mins)
			}

			monthTime, _ := time.Parse("2006-01", month.Month)
			monthInfo := fmt.Sprintf(
				"%s: %d sessions (%s)",
				monthTime.Format("January"),
				month.SessionsCount,
				monthTimeStr,
			)
			months += monthStyle.Render(monthInfo) + "\n"
		}
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		stats,
		avgStats,
		months,
	)
}

func (m Model) renderHelp() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666")).
		MarginTop(2)

	var helpText string
	switch m.viewState {
	case StatsView:
		if m.width > 100 {
			helpText = "d: daily • w: weekly • m: monthly • y: yearly • e: export • b: back • ?: help • g: settings • q: quit"
		} else {
			helpText = "d/w/m/y: details • e: export • b: back • ?: help • q: quit"
		}
	case StatsDetailDaily, StatsDetailWeekly, StatsDetailMonthly, StatsDetailYearly:
		helpText = "e: export all stats • b: back • h: home • ?: help • q: quit"
	default:
		if m.timerRunning {
			if m.width > 80 {
				helpText = "p: pause • r: resume • c: cancel • t: stats • ?: help • g: settings • q: quit"
			} else {
				helpText = "p: pause • r: resume • c: cancel • t: stats • q: quit"
			}
		} else {
			if m.width > 80 {
				helpText = "s: start • t: stats • ?: help • g: settings • q: quit"
			} else {
				helpText = "s: start • t: stats • ?: help • q: quit"
			}
		}
	}

	// Show export message if present
	if m.showExportMsg && m.exportMessage != "" {
		messageStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true).
			MarginBottom(1)
		return lipgloss.JoinVertical(
			lipgloss.Left,
			messageStyle.Render(m.exportMessage),
			helpStyle.Render(helpText),
		)
	}

	return helpStyle.Render(helpText)
}

func (m Model) ShouldQuit() bool {
	return m.shouldQuit
}

func (m Model) ShouldOpenSettings() bool {
	return m.openSettings
}

func getWeekNumber(t time.Time) int {
	_, week := t.ISOWeek()
	return week
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type keyMap struct {
	Start    key.Binding
	Pause    key.Binding
	Resume   key.Binding
	Cancel   key.Binding
	Home     key.Binding
	Stats    key.Binding
	Daily    key.Binding
	Weekly   key.Binding
	Monthly  key.Binding
	Yearly   key.Binding
	Back     key.Binding
	Help     key.Binding
	Settings key.Binding
	Quit     key.Binding
	Export   key.Binding
}

var keys = keyMap{
	Start: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "start"),
	),
	Pause: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "pause"),
	),
	Resume: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "resume"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "cancel"),
	),
	Home: key.NewBinding(
		key.WithKeys("h"),
		key.WithHelp("h", "home"),
	),
	Stats: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "stats"),
	),
	Daily: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "daily details"),
	),
	Weekly: key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "weekly details"),
	),
	Monthly: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "monthly details"),
	),
	Yearly: key.NewBinding(
		key.WithKeys("y"),
		key.WithHelp("y", "yearly details"),
	),
	Back: key.NewBinding(
		key.WithKeys("b", "esc"),
		key.WithHelp("b", "back"),
	),
	Help: key.NewBinding(
		key.WithKeys("?", "f1"),
		key.WithHelp("?/f1", "help"),
	),
	Settings: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "settings"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Export: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "export stats"),
	),
}
