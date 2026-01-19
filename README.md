# gp

CLI tool to capture output from Claude agents running in tmux windows.

Part of the [gastown](https://github.com/steveyegge/gastown) ecosystem. Claude detection logic adapted from gastown.

## Installation

**Go:**
```bash
go install github.com/nmelo/gaspeek@latest
mv $(go env GOPATH)/bin/gaspeek $(go env GOPATH)/bin/gp
```

## Usage

From inside tmux, capture recent output from a window:

```bash
gp editor
```

Capture from all windows in the current session:

```bash
gp --all
```

Only capture from windows running Claude:

```bash
gp --all --detect
```

Limit the number of lines captured:

```bash
gp -n 50 editor
```

Target a different session:

```bash
gp -s work --all
```

## Flags

```
-n, --lines INT        Number of lines to capture (default: 100)
-s, --session NAME     Target session (default: current)
-a, --all              Capture from all windows
-d, --detect           Only capture from windows running Claude
```

## How it works

Uses tmux `capture-pane` to grab the scrollback buffer from target windows. Output includes a header showing window name and session for each captured pane.

Claude detection checks `pane_current_command` for `node`, `claude`, or version patterns, plus child process inspection when the pane shows a shell.

## See also

[gn](https://github.com/nmelo/gasnudge) - the companion tool for sending nudge messages to Claude agents in tmux windows
