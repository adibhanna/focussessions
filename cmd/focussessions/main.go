package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/adibhanna/focussessions/internal/storage"
	"github.com/adibhanna/focussessions/internal/ui/dashboard"
	"github.com/adibhanna/focussessions/internal/ui/settings"
)

const version = "1.0.2"

func main() {
	// Check for version flag
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Printf("Focus Sessions v%s\n", version)
			fmt.Println("A beautiful CLI tool for managing focus sessions and tracking productivity")
			fmt.Println("https://github.com/adibhanna/focussessions")
			return
		case "--help", "-h":
			printHelp()
			return
		}
	}

	storage, err := storage.New()
	if err != nil {
		log.Fatal("Failed to initialize storage:", err)
	}

	if err := runApp(storage); err != nil {
		log.Fatal(err)
	}
}

func runApp(store *storage.Storage) error {
	// Check if this is first time setup
	if store.IsFirstTime() {
		fmt.Println("*** Welcome to Focus Sessions! ***")
		fmt.Println("Let's set up your preferences...")

		// Run first-time settings
		settingsModel, err := settings.New(store)
		if err != nil {
			return err
		}

		p := tea.NewProgram(settingsModel, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		settingsModel = finalModel.(settings.Model)
		fmt.Println("[OK] Setup complete! Let's start focusing!")
	}

	// Main app loop
	for {
		// Create the main dashboard
		dashboardModel, err := dashboard.New(store)
		if err != nil {
			return err
		}

		// Run the main dashboard
		p := tea.NewProgram(dashboardModel, tea.WithAltScreen())
		finalModel, err := p.Run()
		if err != nil {
			return err
		}

		// Check if we should quit or open settings
		dashboardModel = finalModel.(dashboard.Model)
		if dashboardModel.ShouldQuit() {
			fmt.Println(">>> See you next session!")
			return nil
		}

		// Check if user wants to open settings
		if dashboardModel.ShouldOpenSettings() {
			settingsModel, err := settings.New(store)
			if err != nil {
				return err
			}

			p := tea.NewProgram(settingsModel, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return err
			}
		}
	}
}

func printHelp() {
	fmt.Printf("Focus Sessions v%s\n", version)
	fmt.Println("A beautiful CLI tool for managing focus sessions and tracking productivity")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  focussessions           Start the interactive focus session manager")
	fmt.Println("  focussessions --version Show version information")
	fmt.Println("  focussessions --help    Show this help message")
	fmt.Println()
	fmt.Println("Features:")
	fmt.Println("  • Customizable timer sessions")
	fmt.Println("  • Daily progress tracking")
	fmt.Println("  • Weekly & monthly statistics")
	fmt.Println("  • Beautiful terminal UI")
	fmt.Println("  • Persistent storage")
	fmt.Println("  • Configurable goals")
	fmt.Println()
	fmt.Println("Config Files:")
	fmt.Println("  • Sessions: ~/.focussessions/sessions.json")
	fmt.Println("  • Settings: ~/.focussessions/config.json")
	fmt.Println()
	fmt.Println("For more information: https://github.com/adibhanna/focussessions")
}
