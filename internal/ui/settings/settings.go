package settings

import (
	"fmt"
	"strconv"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"focussessions/internal/models"
	"focussessions/internal/storage"
)

type Model struct {
	storage      *storage.Storage
	config       models.Config
	inputs       []textinput.Model
	focusIndex   int
	saved        bool
	reset        bool
	confirmReset bool
	errorMsg     string
	width        int
	height       int
}

func New(storage *storage.Storage) (Model, error) {
	config, err := storage.GetConfig()
	if err != nil {
		return Model{}, err
	}

	inputs := make([]textinput.Model, 4)

	// Validation function to allow only numeric input
	numericValidation := func(text string) error {
		if text == "" {
			return nil // Allow empty input temporarily
		}
		for _, char := range text {
			if !unicode.IsDigit(char) {
				return fmt.Errorf("only numbers allowed")
			}
		}
		return nil
	}

	// Session Duration
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "60"
	inputs[0].SetValue(strconv.Itoa(config.SessionDuration))
	inputs[0].Focus()
	inputs[0].CharLimit = 3
	inputs[0].Width = 20
	inputs[0].Validate = numericValidation

	// Daily Session Goal
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "8"
	inputs[1].SetValue(strconv.Itoa(config.DailySessionGoal))
	inputs[1].CharLimit = 2
	inputs[1].Width = 20
	inputs[1].Validate = numericValidation

	// Work Start Hour
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "8"
	inputs[2].SetValue(strconv.Itoa(config.WorkStartHour))
	inputs[2].CharLimit = 2
	inputs[2].Width = 20
	inputs[2].Validate = numericValidation

	// Work End Hour
	inputs[3] = textinput.New()
	inputs[3].Placeholder = "16"
	inputs[3].SetValue(strconv.Itoa(config.WorkEndHour))
	inputs[3].CharLimit = 2
	inputs[3].Width = 20
	inputs[3].Validate = numericValidation

	return Model{
		storage:    storage,
		config:     config,
		inputs:     inputs,
		focusIndex: 0,
	}, nil
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Tab), key.Matches(msg, keys.Down):
			m.focusIndex++
			if m.focusIndex > len(m.inputs)-1 {
				m.focusIndex = 0
			}
			return m.updateFocus(), nil

		case key.Matches(msg, keys.ShiftTab), key.Matches(msg, keys.Up):
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs) - 1
			}
			return m.updateFocus(), nil

		case key.Matches(msg, keys.Save):
			if err := m.saveConfig(); err == nil {
				m.saved = true
				m.errorMsg = ""
				return m, tea.Quit
			} else {
				m.errorMsg = err.Error()
				m.saved = false
			}

		case key.Matches(msg, keys.Reset):
			if !m.confirmReset {
				m.confirmReset = true
				return m, nil
			} else {
				// Perform reset
				if err := m.resetAllData(); err == nil {
					m.reset = true
					return m, tea.Quit
				}
			}

		case key.Matches(msg, keys.Back), key.Matches(msg, keys.Quit):
			if m.confirmReset {
				m.confirmReset = false
				return m, nil
			}
			return m, tea.Quit
		}
	}

	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *Model) updateFocus() tea.Model {
	for i := range m.inputs {
		if i == m.focusIndex {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	return m
}

func (m *Model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		oldValue := m.inputs[i].Value()
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
		// Clear error message when user starts typing
		if m.inputs[i].Value() != oldValue {
			m.errorMsg = ""
		}
	}
	return tea.Batch(cmds...)
}

func (m *Model) saveConfig() error {
	// Validate session duration (1-180 minutes)
	durationStr := m.inputs[0].Value()
	if durationStr == "" {
		return fmt.Errorf("session duration is required")
	}
	duration, err := strconv.Atoi(durationStr)
	if err != nil || duration < 1 || duration > 180 {
		return fmt.Errorf("session duration must be between 1-180 minutes")
	}

	// Validate daily goal (1-24 sessions)
	goalStr := m.inputs[1].Value()
	if goalStr == "" {
		return fmt.Errorf("daily session goal is required")
	}
	goal, err := strconv.Atoi(goalStr)
	if err != nil || goal < 1 || goal > 24 {
		return fmt.Errorf("daily goal must be between 1-24 sessions")
	}

	// Validate start hour (0-23)
	startHourStr := m.inputs[2].Value()
	if startHourStr == "" {
		return fmt.Errorf("work start hour is required")
	}
	startHour, err := strconv.Atoi(startHourStr)
	if err != nil || startHour < 0 || startHour > 23 {
		return fmt.Errorf("start hour must be between 0-23")
	}

	// Validate end hour (0-23, must be greater than start hour)
	endHourStr := m.inputs[3].Value()
	if endHourStr == "" {
		return fmt.Errorf("work end hour is required")
	}
	endHour, err := strconv.Atoi(endHourStr)
	if err != nil || endHour < 0 || endHour > 23 {
		return fmt.Errorf("end hour must be between 0-23")
	}
	if endHour <= startHour {
		return fmt.Errorf("end hour must be greater than start hour")
	}

	m.config.SessionDuration = duration
	m.config.DailySessionGoal = goal
	m.config.WorkStartHour = startHour
	m.config.WorkEndHour = endHour

	return m.storage.SaveConfig(m.config)
}

func (m *Model) resetAllData() error {
	// Remove all data files
	if err := m.storage.ResetAllData(); err != nil {
		return err
	}

	// Reset to default config
	m.config = models.DefaultConfig()

	// Update input fields
	m.inputs[0].SetValue(strconv.Itoa(m.config.SessionDuration))
	m.inputs[1].SetValue(strconv.Itoa(m.config.DailySessionGoal))
	m.inputs[2].SetValue(strconv.Itoa(m.config.WorkStartHour))
	m.inputs[3].SetValue(strconv.Itoa(m.config.WorkEndHour))

	return nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	containerStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Align(lipgloss.Center, lipgloss.Center).
		Padding(4)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF7CCB")).
		MarginBottom(3).
		Align(lipgloss.Center)

	formStyle := lipgloss.NewStyle().
		Align(lipgloss.Left).
		MarginTop(2).
		MarginBottom(2)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FDFF8C")).
		MarginBottom(1)

	inputStyle := lipgloss.NewStyle().
		MarginBottom(2)

	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4CAF50")).
		Bold(true).
		MarginTop(2)

	title := titleStyle.Render("⚙️  Settings")

	labels := []string{
		"Session Duration (minutes):",
		"Daily Session Goal:",
		"Work Start Hour (24h format):",
		"Work End Hour (24h format):",
	}

	var form string
	for i, label := range labels {
		form += labelStyle.Render(label) + "\n"
		form += inputStyle.Render(m.inputs[i].View()) + "\n"
	}

	help := m.renderHelp()

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		formStyle.Render(form),
		help,
	)

	if m.saved {
		content += "\n" + successStyle.Render("✅ Settings saved successfully!")
	}

	if m.reset {
		content += "\n" + successStyle.Render("🔄 All data reset successfully!")
	}

	if m.confirmReset {
		warningStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true).
			MarginTop(2)
		content += "\n" + warningStyle.Render("⚠️  WARNING: This will delete ALL sessions and reset settings!")
	}

	if m.errorMsg != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true).
			MarginTop(2)
		content += "\n" + errorStyle.Render("❌ "+m.errorMsg)
	}

	return containerStyle.Render(content)
}

func (m Model) renderHelp() string {
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666")).
		MarginTop(2)

	if m.confirmReset {
		return helpStyle.Render("⚠️  Press 'r' again to confirm RESET (deletes all data) • b: cancel")
	}

	return helpStyle.Render("tab/↓: next field • shift+tab/↑: previous • s: save • r: reset all data • b: back • q: quit")
}

type keyMap struct {
	Tab      key.Binding
	ShiftTab key.Binding
	Up       key.Binding
	Down     key.Binding
	Save     key.Binding
	Reset    key.Binding
	Back     key.Binding
	Quit     key.Binding
}

var keys = keyMap{
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next field"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "previous field"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "previous field"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "next field"),
	),
	Save: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "save"),
	),
	Reset: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "reset all data"),
	),
	Back: key.NewBinding(
		key.WithKeys("b", "esc"),
		key.WithHelp("b", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
