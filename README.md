# 🍅 Zoneout - Pomodoro Timer

A beautiful, terminal-based Pomodoro timer written in Go with MP3 white noise support and transition sound effects.

## Features

- **Pomodoro Cycles**: Default 3 sessions of 25-minute focus + 5-minute breaks
- **Real-time Timer**: Live countdown display with minutes and seconds
- **Focus & Break Modes**: Automatic transitions between focus sessions and breaks
- **White Noise Support**: Play MP3 files during focus sessions
  - Automatic discovery of MP3 files in `./whitenoise/` directory
  - Seamless audio switching
- **Transition Sounds**: Audio feedback for mode and status changes
  - `start.mp3`: Plays when entering focus mode or resuming
  - `stop.mp3`: Plays when entering break mode or pausing
- **Daily Messages**: Random motivational messages from `./motd/` directory (refreshes every 24 hours)
- **Statistics**: Track completed sessions
- **Beautiful TUI**: Built with BubbleTea and Lipgloss for a modern terminal interface

## Installation

### Prerequisites

- Go 1.18 or higher
- Audio player:
  - **macOS**: `afplay` (built-in)
  - **Linux**: `ffplay` (from FFmpeg)
  - **Windows**: `ffplay` (from FFmpeg)

### Build from Source

```bash
go build -o zoneout
```

## Usage

```bash
./zoneout
```

### Screenshot

![Zoneout Screenshot](screenshot.png)

### Controls

| Key | Action |
|-----|--------|
| `SPACE` | Start/Pause |
| `r` | Reset |
| `>` | Skip to next phase |
| `a` | Open audio menu |
| `↑/↓` | Navigate menu |
| `enter` | Select audio |
| `q` | Quit |

## Configuration

### Directories

The app automatically creates these directories on first run:

- `./whitenoise/` - Add your MP3 files here (plays during focus sessions)
- `./sounds/` - Required files:
  - `start.mp3` - Plays on focus start/resume
  - `stop.mp3` - Plays on break start/pause
- `./motd/` - Add `.txt` files with motivational messages (one message per line)

### Default Settings

- **Total Sessions**: 3
- **Focus Duration**: 25 minutes
- **Break Duration**: 5 minutes

## Project Structure

```
zoneout/
├── main.go              # Entry point
├── motd.go              # Message of the day logic
├── models/
│   └── pomodoro.go      # Timer logic
├── ui/
│   └── model.go         # UI and interactions
├── audio/
│   └── player.go        # Audio playback
├── stats/
│   └── stats.go         # Session statistics
├── whitenoise/          # MP3 white noise files (user-provided)
├── sounds/              # Sound effect files (user-provided)
├── motd/                # Daily message files (user-provided)
├── go.mod               # Go module
└── README.md
```

## License

MIT
