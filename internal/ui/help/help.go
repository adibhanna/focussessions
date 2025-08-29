package help

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	width  int
	height int
	quit   bool
}

func New() Model {
	return Model{}
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
			m.quit = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m Model) View() string {
	// Use reasonable defaults if dimensions aren't set
	width := m.width
	height := m.height
	if width == 0 {
		width = 100
	}
	if height == 0 {
		height = 30
	}

	// Styles
	containerStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(2)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF7CCB")).
		Align(lipgloss.Center).
		MarginBottom(1)

	dateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888")).
		Align(lipgloss.Center).
		MarginBottom(2)

	sectionTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FDFF8C")).
		MarginBottom(1).
		MarginTop(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4CAF50")).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CCCCCC"))

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666")).
		MarginTop(2).
		Align(lipgloss.Center)

	// Content
	currentYear := time.Now().Year()
	currentDate := time.Now().Format("Monday, January 2, 2006")

	title := titleStyle.Render(fmt.Sprintf("🆘 Focus Sessions Help - %d", currentYear))
	dateInfo := dateStyle.Render(currentDate)

	// Timer Controls Section
	timerSection := sectionTitleStyle.Render("⏱️  Timer Controls")
	timerContent := fmt.Sprintf("%s - %s\n%s - %s\n%s - %s\n%s - %s",
		keyStyle.Render("s"), descStyle.Render("Start a new focus session"),
		keyStyle.Render("p"), descStyle.Render("Pause the current session"),
		keyStyle.Render("r"), descStyle.Render("Resume a paused session"),
		keyStyle.Render("c"), descStyle.Render("Cancel the current session"))

	// Navigation Section
	navSection := sectionTitleStyle.Render("🧭 Navigation")
	navContent := fmt.Sprintf("%s - %s\n%s - %s\n%s - %s\n%s - %s\n%s - %s\n%s - %s\n%s - %s\n%s - %s",
		keyStyle.Render("h"), descStyle.Render("Return to home/main menu"),
		keyStyle.Render("t"), descStyle.Render("Toggle stats view"),
		keyStyle.Render("d"), descStyle.Render("View daily details (from stats view)"),
		keyStyle.Render("w"), descStyle.Render("View weekly details (from stats view)"),
		keyStyle.Render("m"), descStyle.Render("View monthly details (from stats view)"),
		keyStyle.Render("y"), descStyle.Render("View yearly details (from stats view)"),
		keyStyle.Render("b / esc"), descStyle.Render("Go back to previous view"),
		keyStyle.Render("? / f1"), descStyle.Render("Show this help page"))

	// Settings & App Section
	appSection := sectionTitleStyle.Render("⚙️  Settings & App")
	appContent := fmt.Sprintf("%s - %s\n%s - %s",
		keyStyle.Render("g"), descStyle.Render("Open settings"),
		keyStyle.Render("q / Ctrl+C"), descStyle.Render("Quit the application"))

	// Menu Navigation Section
	menuSection := sectionTitleStyle.Render("📋 Menu Navigation")
	menuContent := fmt.Sprintf("%s - %s\n%s - %s\n%s - %s",
		keyStyle.Render("↑ / k"), descStyle.Render("Move up in menus"),
		keyStyle.Render("↓ / j"), descStyle.Render("Move down in menus"),
		keyStyle.Render("Enter / Space"), descStyle.Render("Select menu item"))

	// Data Recovery Section
	recoverySection := sectionTitleStyle.Render("🔄 Data Recovery")
	recoveryContent := descStyle.Render(
		"Don't worry about accidentally quitting the app during a session!\n" +
			"Your progress is automatically saved and you can resume where you\n" +
			"left off. Active sessions are paused when you quit and will appear\n" +
			"in the main menu for easy resuming.\n\n" +
			"All session data is stored locally in ~/.focussessions/ as JSON files")

	// About Section
	aboutSection := sectionTitleStyle.Render("ℹ️  About Focus Sessions")
	aboutContent := descStyle.Render(
		"Focus Sessions is a productivity timer application that helps you\n" +
			"maintain focus using the Pomodoro Technique. Track your daily,\n" +
			"weekly, monthly, and yearly progress to build better focus habits.\n\n" +
			"Default session duration: 60 minutes\n" +
			"Customize settings with 'g' key")

	footer := footerStyle.Render("Press 'h' for home • 'b/esc' to go back • 'q' to quit")

	// Combine all sections
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		dateInfo,
		timerSection,
		timerContent,
		navSection,
		navContent,
		appSection,
		appContent,
		menuSection,
		menuContent,
		recoverySection,
		recoveryContent,
		aboutSection,
		aboutContent,
		footer,
	)

	return containerStyle.Render(content)
}

func (m Model) ShouldQuit() bool {
	return m.quit
}

type keyMap struct {
	Back key.Binding
	Quit key.Binding
	Home key.Binding
}

var keys = keyMap{
	Back: key.NewBinding(
		key.WithKeys("b", "esc"),
		key.WithHelp("b/esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Home: key.NewBinding(
		key.WithKeys("h"),
		key.WithHelp("h", "home"),
	),
}
