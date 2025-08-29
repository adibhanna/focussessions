# Focus Sessions - Agent Guidelines

## Build & Test Commands
```bash
make build      # Build the binary
make test       # Run all tests (go test -v ./...)
make fmt        # Format code (go fmt ./...)
make lint       # Lint code (requires golangci-lint)
go test -v ./internal/ui/timer  # Run single package tests
```

## Code Style & Conventions
- **Imports**: Group stdlib, third-party (Charm libs), then internal packages
- **Error Handling**: Return errors up the chain, log.Fatal only in main()
- **Naming**: Use descriptive names (e.g., `SessionDuration` not `sd`)
- **Types**: Define structs in models package with JSON tags for persistence
- **UI Components**: Each UI component in separate package under internal/ui/
- **Storage**: All data operations through storage.Storage abstraction
- **Time**: Use time.Time for timestamps, int for durations in minutes/seconds
- **Dependencies**: Bubble Tea for TUI, Lipgloss for styling, Bubbles for components

## Project Structure
- `cmd/focussessions/` - Entry point
- `internal/models/` - Data structures  
- `internal/storage/` - Persistence layer
- `internal/ui/*/` - UI components (menu, timer, stats, settings)