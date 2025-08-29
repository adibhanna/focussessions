package timer

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"focussessions/internal/models"
	"focussessions/internal/storage"
)

type tickMsg time.Time

type Model struct {
	duration       int
	elapsed        int
	running        bool
	paused         bool
	finished       bool
	cancelled      bool
	exitToMenu     bool
	progress       progress.Model
	storage        *storage.Storage
	currentSession *models.Session
	width          int
	height         int
	isResuming     bool
}

func New(duration int, storage *storage.Storage) Model {
	prog := progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	prog.Width = 60

	return Model{
		duration: duration * 60, // Convert to seconds
		elapsed:  0,
		running:  false,
		paused:   false,
		finished: false,
		progress: prog,
		storage:  storage,
	}
}

func NewFromSession(session *models.Session, storage *storage.Storage) Model {
	prog := progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	prog.Width = 60

	return Model{
		duration:       session.Duration * 60,
		elapsed:        session.ElapsedSeconds,
		running:        true,
		paused:         session.Paused,
		finished:       false,
		progress:       prog,
		storage:        storage,
		currentSession: session,
		isResuming:     true,
	}
}

func (m Model) Init() tea.Cmd {
	if m.isResuming && m.running && !m.paused {
		return tickCmd()
	}
	return nil
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = min(msg.Width-20, 80)
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Start) && !m.running && !m.finished:
			// Deactivate any other active sessions first
			m.storage.DeactivateAllSessions()

			m.running = true
			m.paused = false
			m.currentSession = &models.Session{
				ID:             uuid.New().String(),
				StartTime:      time.Now(),
				Duration:       m.duration / 60,
				Date:           time.Now().Format("2006-01-02"),
				Week:           getWeekNumber(time.Now()),
				Month:          time.Now().Format("2006-01"),
				Year:           time.Now().Year(),
				Active:         true,
				ElapsedSeconds: 0,
				Paused:         false,
			}
			m.storage.SaveSession(*m.currentSession)
			return m, tickCmd()

		case key.Matches(msg, keys.Pause) && m.running && !m.paused:
			m.paused = true
			if m.currentSession != nil {
				m.currentSession.Paused = true
				m.currentSession.ElapsedSeconds = m.elapsed
				m.storage.SaveSession(*m.currentSession)
			}
			return m, nil

		case key.Matches(msg, keys.Resume) && m.running && m.paused:
			m.paused = false
			if m.currentSession != nil {
				m.currentSession.Paused = false
				m.storage.SaveSession(*m.currentSession)
			}
			return m, tickCmd()

		case key.Matches(msg, keys.Cancel) && m.running:
			m.running = false
			m.cancelled = true
			m.finished = true
			if m.currentSession != nil {
				m.currentSession.EndTime = time.Now()
				m.currentSession.Completed = false
				m.currentSession.Active = false
				m.storage.SaveSession(*m.currentSession)
			}
			return m, tea.Quit

		case key.Matches(msg, keys.Menu) && m.running && !m.finished:
			// Save current state and exit to menu
			m.exitToMenu = true
			if m.currentSession != nil {
				m.currentSession.ElapsedSeconds = m.elapsed
				m.currentSession.Paused = m.paused
				m.storage.SaveSession(*m.currentSession)
			}
			return m, tea.Quit

		case key.Matches(msg, keys.Home):
			// Save current state and exit to home/menu
			m.exitToMenu = true
			if m.currentSession != nil && m.running && !m.finished {
				m.currentSession.ElapsedSeconds = m.elapsed
				m.currentSession.Paused = true // Pause when going home
				m.storage.SaveSession(*m.currentSession)
			}
			return m, tea.Quit

		case key.Matches(msg, keys.Quit):
			if m.running && !m.finished {
				// Save state when quitting
				if m.currentSession != nil {
					m.currentSession.ElapsedSeconds = m.elapsed
					m.currentSession.Paused = true
					m.storage.SaveSession(*m.currentSession)
				}
			}
			return m, tea.Quit

		case key.Matches(msg, keys.Back) && m.finished:
			return m, tea.Quit
		}

	case tickMsg:
		if m.running && !m.paused && !m.finished {
			m.elapsed++

			// Save progress periodically (every 10 seconds)
			if m.elapsed%10 == 0 && m.currentSession != nil {
				m.currentSession.ElapsedSeconds = m.elapsed
				m.storage.SaveSession(*m.currentSession)
			}

			if m.elapsed >= m.duration {
				m.finished = true
				m.running = false
				if m.currentSession != nil {
					m.currentSession.EndTime = time.Now()
					m.currentSession.Completed = true
					m.currentSession.Active = false
					m.storage.SaveSession(*m.currentSession)
				}
				return m, tea.Batch(
					tea.Println("🎉 Session completed!"),
					tea.Quit,
				)
			}
			return m, tickCmd()
		}

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
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
		Align(lipgloss.Center, lipgloss.Center).
		Padding(2)

	// Show celebration screen if finished
	if m.finished && !m.cancelled {
		return containerStyle.Render(m.renderCompletionCelebration())
	}

	remaining := m.duration - m.elapsed
	minutes := remaining / 60
	seconds := remaining % 60

	percent := float64(m.elapsed) / float64(m.duration)

	var status string
	if !m.running && !m.finished {
		status = "Press 's' to start your focus session"
	} else if m.paused {
		status = "PAUSED - Press 'r' to resume"
	} else if m.finished {
		if m.cancelled {
			status = "Session cancelled"
		} else {
			status = "🎉 Session completed! Great job!"
		}
	} else {
		status = "Focus time! Stay in the zone..."
	}

	timerDisplay := fmt.Sprintf("%02d:%02d", minutes, seconds)

	timerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(2, 4).
		MarginBottom(2)

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888")).
		MarginBottom(2)

	progressBar := m.progress.ViewAs(percent)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		timerStyle.Render(timerDisplay),
		progressBar,
		statusStyle.Render(status),
		helpView(m.running),
	)

	return containerStyle.Render(content)
}

func (m Model) renderCompletionCelebration() string {
	celebrationStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFD700")).
		Align(lipgloss.Center)

	// Create celebration ASCII art
	celebration := []string{
		"",
		"       ✨  🌟  ✨",
		"",
		"    ╔═══════════════════╗",
		"    ║  SESSION COMPLETE! ║",
		"    ╚═══════════════════╝",
		"",
		"          🎉 🎊 🎉",
		"",
		"      ┌─────────────┐",
		"      │   AMAZING   │",
		"      │    WORK!    │",
		"      └─────────────┘",
		"",
		fmt.Sprintf("     Duration: %d minutes", m.duration/60),
		"",
		"    You stayed focused!",
		"    Keep the momentum going!",
		"",
		"       🏆  💪  🚀",
		"",
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666")).
		MarginTop(2)
	help := helpStyle.Render("Press 'b' to go back • 'q' to quit")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		celebrationStyle.Render(lipgloss.JoinVertical(lipgloss.Center, celebration...)),
		help,
	)

	return content
}

func helpView(running bool) string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666")).
		MarginTop(2)

	var helpText string
	if !running {
		helpText = "s: start • h: home • q: quit"
	} else {
		helpText = "p: pause • r: resume • c: cancel • h: home • m: menu • q: quit"
	}

	return helpStyle.Render(helpText)
}

func (m Model) ExitedToMenu() bool {
	return m.exitToMenu
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
	Start  key.Binding
	Pause  key.Binding
	Resume key.Binding
	Cancel key.Binding
	Quit   key.Binding
	Back   key.Binding
	Menu   key.Binding
	Home   key.Binding
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
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Back: key.NewBinding(
		key.WithKeys("b", "esc"),
		key.WithHelp("b", "back"),
	),
	Menu: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "back to menu"),
	),
	Home: key.NewBinding(
		key.WithKeys("h"),
		key.WithHelp("h", "home"),
	),
}
