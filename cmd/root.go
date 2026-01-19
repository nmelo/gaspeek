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
	Long: `gaspeek reads recent output from tmux windows.

Examples:
  gp editor                    # Capture last 100 lines from 'editor' window
  gp -n 50 editor              # Capture last 50 lines
  gp --all                     # Capture from all windows
  gp --all --detect            # Only windows running Claude`,
	RunE: runPeek,
}

func Execute() error {
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
