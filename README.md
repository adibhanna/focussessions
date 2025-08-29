# Focus Sessions 🎯

A beautiful CLI tool for managing focus sessions and tracking productivity. Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) for a delightful terminal UI experience.

## Features ✨

- **Customizable Timer Sessions**: Set your preferred session duration (default: 60 minutes)
- **Daily Progress Tracking**: See how many sessions you've completed today
- **Weekly & Monthly Statistics**: Review your productivity patterns over time
- **Beautiful Terminal UI**: Clean, intuitive interface with progress bars and visual feedback
- **Persistent Storage**: All your sessions are saved locally
- **Configurable Goals**: Set daily session targets to stay motivated
- **Work Hours Configuration**: Define your working hours for better tracking

## Installation 📦

```bash
# Clone the repository
git clone https://github.com/yourusername/focussessions.git
cd focussessions

# Build the application
go build -o focussessions cmd/focussessions/main.go

# Run the application
./focussessions
```

Or install directly with Go:

```bash
go install github.com/yourusername/focussessions/cmd/focussessions@latest
```

## Usage 🚀

Simply run the application:

```bash
focussessions
```

### Main Menu

Navigate the main menu using arrow keys or `j`/`k`:

- **Start Focus Session**: Begin a new timed focus session
- **Today's Progress**: View your sessions from today
- **This Week's Stats**: See your weekly productivity
- **This Month's Stats**: Review monthly performance
- **Settings**: Configure session duration, daily goals, and work hours
- **Exit**: Close the application

### During a Session

- `s` - Start the session
- `p` - Pause the timer
- `r` - Resume from pause
- `c` - Cancel the session
- `q` - Quit (saves session as incomplete)

### Settings Configuration

Customize your experience:

- **Session Duration**: Set how long each focus session lasts (1-180 minutes)
- **Daily Session Goal**: Target number of sessions per day (1-24)
- **Work Start Hour**: When your workday begins (0-23)
- **Work End Hour**: When your workday ends (0-23)

## Data Storage 📁

All session data and configuration is stored in:
- `~/.focussessions/sessions.json` - Your session history
- `~/.focussessions/config.json` - Your preferences

## Screenshots 📸

### Main Menu
```
✨ Focus Sessions ✨

Today: 3 sessions | 3.0 hours | Goal: 8 sessions
[██████████░░░░░░░░░░░░░░░░░░░]

▶ 🚀 Start Focus Session
  📊 Today's Progress
  📅 This Week's Stats
  📈 This Month's Stats
  ⚙️  Settings
  👋 Exit

↑/↓: navigate • enter: select • q: quit
```

### Active Timer
```
        25:00
        
[████████████░░░░░░░░░░░░░░░░]

Focus time! Stay in the zone...

s: start p: pause r: resume c: cancel q: quit
```

## Development 🛠

### Prerequisites

- Go 1.21 or higher
- Terminal with UTF-8 support

### Project Structure

```
focussessions/
├── cmd/
│   └── focussessions/
│       └── main.go          # Application entry point
├── internal/
│   ├── models/
│   │   └── session.go       # Data models
│   ├── storage/
│   │   └── storage.go       # Persistence layer
│   └── ui/
│       ├── menu/
│       │   └── menu.go      # Main menu component
│       ├── timer/
│       │   └── timer.go     # Timer component
│       ├── stats/
│       │   └── stats.go     # Statistics views
│       └── settings/
│           └── settings.go  # Settings management
├── go.mod
├── go.sum
└── README.md
```

### Building from Source

```bash
# Get dependencies
go mod download

# Build the binary
go build -o focussessions cmd/focussessions/main.go

# Run tests (if available)
go test ./...
```

## Contributing 🤝

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License 📄

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments 🙏

- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) by Charm
- Styled with [Lipgloss](https://github.com/charmbracelet/lipgloss)
- Progress bars from [Bubbles](https://github.com/charmbracelet/bubbles)

## Inspiration 💡

This tool was inspired by the Pomodoro Technique and the need for a simple, beautiful way to track focus sessions in the terminal.

---

Made with ❤️ for focused developers everywhere