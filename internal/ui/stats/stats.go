package stats

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"focussessions/internal/models"
	"focussessions/internal/storage"
)

type ViewType int

const (
	DayView ViewType = iota
	WeekView
	MonthView
	YearView
)

type Model struct {
	viewType      ViewType
	storage       *storage.Storage
	dayStats      models.DayStats
	weekStats     models.WeekStats
	monthStats    models.MonthStats
	yearStats     models.YearStats
	width         int
	height        int
	exportMessage string
	showMessage   bool
}

func New(viewType ViewType, storage *storage.Storage) (Model, error) {
	m := Model{
		viewType: viewType,
		storage:  storage,
	}

	now := time.Now()
	var err error

	switch viewType {
	case DayView:
		m.dayStats, err = storage.GetDayStats(now.Format("2006-01-02"))
	case WeekView:
		_, week := now.ISOWeek()
		m.weekStats, err = storage.GetWeekStats(now.Year(), week)
	case MonthView:
		m.monthStats, err = storage.GetMonthStats(now.Year(), int(now.Month()))
	case YearView:
		m.yearStats, err = storage.GetYearStats(now.Year())
	}

	if err != nil {
		return m, err
	}

	return m, nil
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
		case key.Matches(msg, keys.Back), key.Matches(msg, keys.Quit), key.Matches(msg, keys.Home):
			return m, tea.Quit
		case key.Matches(msg, keys.Export):
			return m, m.exportStats()
		}

	case exportResultMsg:
		m.exportMessage = msg.message
		m.showMessage = true
		// Clear message after 3 seconds
		return m, tea.Tick(time.Second*3, func(t time.Time) tea.Msg {
			return clearMessageMsg{}
		})

	case clearMessageMsg:
		m.showMessage = false
		m.exportMessage = ""
		return m, nil
	}

	return m, nil
}

type clearMessageMsg struct{}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	containerStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Padding(2)

	var content string

	switch m.viewType {
	case DayView:
		content = m.renderDayView()
	case WeekView:
		content = m.renderWeekView()
	case MonthView:
		content = m.renderMonthView()
	case YearView:
		content = m.renderYearView()
	}

	help := m.renderHelp()

	fullContent := lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		help,
	)

	return containerStyle.Render(fullContent)
}

func (m Model) renderDayView() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF7CCB")).
		MarginBottom(2)

	statsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FDFF8C")).
		MarginBottom(1)

	sessionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888")).
		PaddingLeft(2)

	date, _ := time.Parse("2006-01-02", m.dayStats.Date)
	title := titleStyle.Render(fmt.Sprintf("📊 Daily Stats - %s", date.Format("Monday, January 2, 2006")))

	stats := statsStyle.Render(fmt.Sprintf(
		"Sessions: %d | Total Time: %d mins",
		m.dayStats.SessionsCount,
		m.dayStats.TotalMinutes,
	))

	var sessions string
	if len(m.dayStats.Sessions) == 0 {
		sessions = sessionStyle.Render("No sessions yet today. Time to focus! 🚀")
	} else {
		sessions = "\nSession History:\n"
		for i, session := range m.dayStats.Sessions {
			var status string
			var sessionInfo string

			if session.Active {
				// Currently active session
				elapsed := session.ElapsedSeconds / 60
				if session.Paused {
					status = "⏸️"
					sessionInfo = fmt.Sprintf(
						"%s Session %d: In Progress (Paused) - %d/%d min",
						status,
						i+1,
						elapsed,
						session.Duration,
					)
				} else {
					status = "▶️"
					sessionInfo = fmt.Sprintf(
						"%s Session %d: In Progress - %d/%d min",
						status,
						i+1,
						elapsed,
						session.Duration,
					)
				}
			} else if session.Completed {
				status = "✅"
				sessionInfo = fmt.Sprintf(
					"%s Session %d: %s - %s (%d min)",
					status,
					i+1,
					session.StartTime.Format("3:04 PM"),
					session.EndTime.Format("3:04 PM"),
					session.Duration,
				)
			} else {
				// Cancelled or incomplete
				actualDuration := 0
				if !session.EndTime.IsZero() && !session.StartTime.IsZero() {
					actualDuration = int(session.EndTime.Sub(session.StartTime).Minutes())
				}
				if session.ElapsedSeconds > 0 {
					actualDuration = session.ElapsedSeconds / 60
				}

				status = "⚠️"
				sessionInfo = fmt.Sprintf(
					"%s Session %d: Cancelled - %s (%d/%d min)",
					status,
					i+1,
					session.StartTime.Format("3:04 PM"),
					actualDuration,
					session.Duration,
				)
			}
			sessions += sessionStyle.Render(sessionInfo) + "\n"
		}
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		stats,
		sessions,
	)
}

func (m Model) renderWeekView() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF7CCB")).
		MarginBottom(2)

	statsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FDFF8C")).
		MarginBottom(1)

	dayStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888")).
		PaddingLeft(2)

	title := titleStyle.Render(fmt.Sprintf("📅 Weekly Stats - Week %d, %d", m.weekStats.Week, m.weekStats.Year))

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
		"Total Sessions: %d | Total Time: %s",
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

	chart := m.renderWeekChart()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		stats,
		chart,
		days,
	)
}

func (m Model) renderWeekChart() string {
	chartStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666")).
		MarginTop(1).
		MarginBottom(1)

	maxSessions := 0
	for _, day := range m.weekStats.DailyStats {
		if day.SessionsCount > maxSessions {
			maxSessions = day.SessionsCount
		}
	}

	if maxSessions == 0 {
		return ""
	}

	chart := "\n"
	barHeight := 5

	dayMap := make(map[string]int)
	for _, day := range m.weekStats.DailyStats {
		date, _ := time.Parse("2006-01-02", day.Date)
		dayMap[date.Format("Mon")] = day.SessionsCount
	}

	days := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

	for row := barHeight; row > 0; row-- {
		for _, day := range days {
			sessions := dayMap[day]
			barLevel := int(float64(sessions) / float64(maxSessions) * float64(barHeight))
			if barLevel >= row {
				chart += "█ "
			} else {
				chart += "  "
			}
		}
		chart += "\n"
	}

	for _, day := range days {
		chart += day[:2] + " "
	}

	return chartStyle.Render(chart)
}

func (m Model) renderMonthView() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF7CCB")).
		MarginBottom(2)

	statsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FDFF8C")).
		MarginBottom(1)

	weekStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888")).
		PaddingLeft(2)

	monthTime, _ := time.Parse("2006-01", m.monthStats.Month)
	title := titleStyle.Render(fmt.Sprintf("📈 Monthly Stats - %s", monthTime.Format("January 2006")))

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

			weekInfo := fmt.Sprintf(
				"Week %d: %d sessions (%s)",
				week.Week,
				week.SessionsCount,
				timeStr,
			)
			weeks += weekStyle.Render(weekInfo) + "\n"
		}
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		stats,
		avgStats,
		weeks,
	)
}

func (m Model) renderYearView() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF7CCB")).
		MarginBottom(2)

	statsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FDFF8C")).
		MarginBottom(1)

	monthStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888")).
		PaddingLeft(2)

	title := titleStyle.Render(fmt.Sprintf("📊 Yearly Stats - %d", m.yearStats.Year))

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

			monthTime, _ := time.Parse("2006-01", month.Month)
			monthInfo := fmt.Sprintf(
				"%s: %d sessions (%s)",
				monthTime.Format("January"),
				month.SessionsCount,
				timeStr,
			)
			months += monthStyle.Render(monthInfo) + "\n"
		}
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		stats,
		avgStats,
		months,
	)
}

func (m Model) renderHelp() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666")).
		MarginTop(2)

	help := "Press 'e' to export • 'h' for home • 'b' to go back • 'q' to quit"

	if m.showMessage && m.exportMessage != "" {
		messageStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true)
		help = messageStyle.Render(m.exportMessage) + "\n" + help
	}

	return helpStyle.Render(help)
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

		return exportResultMsg{success: true, message: fmt.Sprintf("✅ Exported to %s", filePath)}
	}
}

type exportResultMsg struct {
	success bool
	message string
}

type keyMap struct {
	Back   key.Binding
	Quit   key.Binding
	Home   key.Binding
	Export key.Binding
}

var keys = keyMap{
	Back: key.NewBinding(
		key.WithKeys("b", "esc"),
		key.WithHelp("b", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Home: key.NewBinding(
		key.WithKeys("h"),
		key.WithHelp("h", "home"),
	),
	Export: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "export"),
	),
}
