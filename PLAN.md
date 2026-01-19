# gaspeek Implementation Plan

## Overview

Create a CLI tool (`gp`) that reads recent output from tmux windows. This is the companion to [gn](https://github.com/nmelo/gasnudge) which sends messages to windows.

Together they form the nudge/peek pair for agent communication:
- `gn` sends messages TO windows
- `gp` reads output FROM windows

## Architecture

```
gaspeek/
├── main.go
├── go.mod
├── internal/
│   └── tmux/
│       └── tmux.go         # Window discovery + capture
└── cmd/
    └── root.go             # Cobra CLI definition
```

## Core Components

### 1. tmux Package (`internal/tmux/tmux.go`)

Can reuse most of gn's tmux package. Add capture function:

```go
// CaptureWindow captures recent output from a tmux window
// Uses: tmux capture-pane -p -t "session:window" -S -lines
func CaptureWindow(target string, lines int) (string, error) {
    out, err := run("capture-pane", "-p", "-t", target, "-S", fmt.Sprintf("-%d", lines))
    if err != nil {
        return "", err
    }
    return out, nil
}
```

### 2. CLI (`cmd/root.go`)

```
gp [flags] [window]

Flags:
  -n, --lines INT        Number of lines to capture (default: 100)
  -s, --session NAME     Target session (default: current)
  -a, --all              Capture from all windows (concatenated)
  -d, --detect           Only capture from windows running Claude
  -h, --help             Help
```

**Default behavior:**
- If window specified: capture from that window
- If no window and inside tmux: error (must specify window or use --all)
- If outside tmux: require `-s` flag

### 3. Output Format

When capturing single window:
```
[raw output from capture-pane]
```

When capturing multiple windows (--all):
```
=== window-name (session:index) ===
[output]

=== another-window (session:index) ===
[output]
```

## Reference Code

From gastown (`~/gastown/internal/tmux/tmux.go`):
```go
func (t *Tmux) CapturePane(session string, lines int) (string, error) {
    return t.run("capture-pane", "-p", "-t", session, "-S", fmt.Sprintf("-%d", lines))
}
```

From gn (`~/Desktop/Projects/gasnudge/internal/tmux/tmux.go`):
- `ListWindows()` - enumerate windows
- `GetCurrentContext()` - detect caller's context
- `IsClaudeRunning()` - detect Claude in window
- `MatchPattern()` - glob matching

## tmux Commands Used

```bash
# Capture last N lines from window
tmux capture-pane -p -t "SESSION:WINDOW" -S -100

# Capture entire scrollback
tmux capture-pane -p -t "SESSION:WINDOW" -S -
```

## Implementation Steps

1. Initialize Go module: `go mod init github.com/nmelo/gaspeek`
2. Copy tmux package from gn, add `CaptureWindow()` function
3. Create CLI with Cobra
4. Create main.go entry point
5. Build and test
6. Create README linking to gn
7. Create GitHub repo and push

## README Template

```markdown
# gp

CLI tool to read output from tmux windows.

Capture logic ripped straight out of [gastown](https://github.com/steveyegge/gastown).

## Installation

\`\`\`bash
go install github.com/nmelo/gaspeek@latest
\`\`\`

## Usage

Capture last 100 lines from a window:

\`\`\`bash
gp editor
gp -n 50 editor      # Last 50 lines
\`\`\`

Capture from all windows:

\`\`\`bash
gp --all
gp --all --detect    # Only windows running Claude
\`\`\`

## Flags

\`\`\`
-n, --lines INT        Number of lines to capture (default: 100)
-s, --session NAME     Target session (default: current)
-a, --all              Capture from all windows
-d, --detect           Only capture from windows running Claude
\`\`\`

## See also

[gn](https://github.com/nmelo/gasnudge) - the companion tool for sending messages to tmux windows
```
