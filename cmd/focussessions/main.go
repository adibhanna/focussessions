package main

import (
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"focussessions/internal/storage"
	"focussessions/internal/ui/dashboard"
	"focussessions/internal/ui/settings"
)

func main() {
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
