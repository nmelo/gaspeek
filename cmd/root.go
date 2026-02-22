package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/nmelo/gaspeek/internal/tmux"
	"github.com/spf13/cobra"
)

var (
	linesFlag   int
	sessionFlag string
	allFlag     bool
	detectFlag  bool
)

var rootCmd = &cobra.Command{
	Use:   "gp [flags] [window]",
	Short: "Read output from tmux windows",
	Long: `gaspeek (gp) reads recent output from Claude agents in tmux windows.

BEHAVIOR:
  - Non-intrusive: reads scrollback buffer without sending any input
  - Single window mode: specify window name as argument
  - Multi-window mode: use --all to capture from all windows
  - Excludes caller's own window by default in multi-window mode
  - Output includes headers showing window name and session

CLAUDE DETECTION (--detect flag):
  Identifies Claude by pane_current_command matching:
  - "claude" or "node" (direct process)
  - Version pattern like "2.1.25"
  - Child processes of shells (inspects via pgrep)

USE CASES FOR AGENT COORDINATION:
  - Check agent status without interrupting their work
  - Monitor swarm progress across multiple workers
  - Debug agent behavior by reviewing recent output
  - Verify an agent received and processed a message
  - Gather context before sending follow-up instructions

EXAMPLES:
  gp worker-1                        # Last 100 lines from 'worker-1'
  gp -n 50 worker-1                  # Last 50 lines
  gp -n 200 worker-1                 # Last 200 lines (more context)
  gp --all                           # Capture from all windows
  gp --all --detect                  # Only windows running Claude
  gp -s swarm --all                  # All windows in 'swarm' session
  gp -s swarm --all --detect         # Claude windows in 'swarm' session

OUTPUT FORMAT:
  Single window: Raw output only
  Multi-window:  Headers like "=== worker-1 (session:0) ===" before each

RELATED TOOLS:
  gn (gasnudge) - Interrupt agents urgently (sends Escape + Enter)
  ga (gasadd)   - Queue messages without interrupting
  gm (gasmail)  - Persistent messaging via beads database`,
	RunE: runPeek,
}

func Execute(version string) error {
	rootCmd.Version = version
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().IntVarP(&linesFlag, "lines", "n", 100, "Number of lines to capture")
	rootCmd.Flags().StringVarP(&sessionFlag, "session", "s", "", "Target session (default: current)")
	rootCmd.Flags().BoolVarP(&allFlag, "all", "a", false, "Capture from all windows")
	rootCmd.Flags().BoolVarP(&detectFlag, "detect", "d", false, "Only capture from windows running Claude")
}

func runPeek(cmd *cobra.Command, args []string) error {
	var session string
	var currentWindowIndex int

	if tmux.IsInsideTmux() {
		var err error
		session, currentWindowIndex, _, err = tmux.GetCurrentContext()
		if err != nil {
			return fmt.Errorf("failed to get tmux context: %w", err)
		}
		if sessionFlag != "" {
			session = sessionFlag
		}
	} else {
		if sessionFlag == "" {
			return fmt.Errorf("not inside tmux; use -s/--session to specify target session")
		}
		session = sessionFlag
		currentWindowIndex = -1
	}

	if !tmux.SessionExists(session) {
		return fmt.Errorf("session %q does not exist", session)
	}

	// Single window mode
	if len(args) > 0 && !allFlag {
		windowName := args[0]
		windows, err := tmux.ListWindows(session)
		if err != nil {
			return fmt.Errorf("failed to list windows: %w", err)
		}

		var target *tmux.Window
		for _, w := range windows {
			if w.Name == windowName || fmt.Sprintf("%d", w.Index) == windowName {
				target = &w
				break
			}
		}
		if target == nil {
			return fmt.Errorf("window %q not found in session %q", windowName, session)
		}

		output, err := tmux.CaptureWindow(fmt.Sprintf("%s:%d", session, target.Index), linesFlag)
		if err != nil {
			return fmt.Errorf("failed to capture window: %w", err)
		}
		fmt.Print(output)
		return nil
	}

	// Multi-window mode (--all)
	if !allFlag {
		return fmt.Errorf("specify a window name or use --all to capture from all windows")
	}

	windows, err := tmux.ListWindows(session)
	if err != nil {
		return fmt.Errorf("failed to list windows: %w", err)
	}

	var targets []tmux.Window
	for _, w := range windows {
		// Exclude current window
		if currentWindowIndex >= 0 && w.Index == currentWindowIndex {
			continue
		}

		if detectFlag && !tmux.IsClaudeRunning(w) {
			continue
		}

		targets = append(targets, w)
	}

	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "No windows to capture")
		return nil
	}

	var outputs []string
	for _, w := range targets {
		target := fmt.Sprintf("%s:%d", session, w.Index)
		output, err := tmux.CaptureWindow(target, linesFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to capture %s: %v\n", w.Name, err)
			continue
		}
		header := fmt.Sprintf("=== %s (%s:%d) ===", w.Name, session, w.Index)
		outputs = append(outputs, header+"\n"+output)
	}

	fmt.Print(strings.Join(outputs, "\n"))
	return nil
}
