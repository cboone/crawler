package crawler

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cboone/crawler/internal/tmuxcli"
)

// Terminal is a handle to a TUI program running inside a tmux session.
// It is created with Open and cleaned up automatically via t.Cleanup.
type Terminal struct {
	t          testing.TB
	runner     *tmuxcli.Runner
	socketPath string
	pane       string
	opts       options
}

const failureCaptureHistory = 3

// Open starts the binary in a new tmux session.
// Cleanup is automatic via t.Cleanup — no defer needed.
func Open(t testing.TB, binary string, userOpts ...Option) *Terminal {
	t.Helper()

	opts := defaultOptions()
	for _, o := range userOpts {
		o(&opts)
	}

	// Resolve and verify tmux.
	tmuxPath, explicit := resolveTmuxPath(t, opts.tmuxPath)
	checkTmuxVersion(t, tmuxPath, explicit)

	// Generate socket path.
	socketPath := generateSocketPath(t)

	// Create runner.
	runner := tmuxcli.New(tmuxPath, socketPath)

	// For environment variables, wrap the binary in /usr/bin/env.
	actualBinary := binary
	actualArgs := opts.args
	if len(opts.env) > 0 {
		actualArgs = make([]string, 0, len(opts.env)+1+len(opts.args))
		actualArgs = append(actualArgs, opts.env...)
		actualArgs = append(actualArgs, binary)
		actualArgs = append(actualArgs, opts.args...)
		actualBinary = "/usr/bin/env"
	}

	optsForSession := opts
	optsForSession.args = actualArgs

	// Write tmux config file and set it on the runner.
	configPath := socketPath + ".conf"
	if err := writeConfig(configPath, opts); err != nil {
		t.Fatalf("%v", err)
	}
	runner.SetConfigPath(configPath)

	if err := startSession(runner, actualBinary, optsForSession); err != nil {
		t.Fatalf("%v", err)
	}

	// Wait for the session to be ready.
	if err := runner.WaitForSession(5 * time.Second); err != nil {
		t.Fatalf("crawler: open: %v", err)
	}

	// Get the pane ID.
	output, err := runner.Run("list-panes", "-F", "#{pane_id}")
	if err != nil {
		t.Fatalf("crawler: open: failed to get pane ID: %v", err)
	}
	pane := strings.TrimSpace(output)

	term := &Terminal{
		t:          t,
		runner:     runner,
		socketPath: socketPath,
		pane:       pane,
		opts:       opts,
	}

	// Register cleanup.
	t.Cleanup(func() {
		_ = killServer(runner)
		os.Remove(configPath)
	})

	return term
}

// SendKeys sends raw tmux key sequences. Escape hatch for advanced use.
func (term *Terminal) SendKeys(keys ...string) {
	term.t.Helper()
	term.requireAlive("send-keys")
	if err := sendKeys(term.runner, term.pane, keys); err != nil {
		term.t.Fatalf("crawler: send-keys: %v", err)
	}
}

// Type sends a string as sequential keypresses.
func (term *Terminal) Type(s string) {
	term.t.Helper()
	term.requireAlive("send-keys")

	// Send the string literally via tmux send-keys -l (literal mode).
	args := []string{"send-keys", "-t", term.pane, "-l", s}
	if _, err := term.runner.Run(args...); err != nil {
		term.t.Fatalf("crawler: send-keys: %v", err)
	}
}

// Press sends one or more special keys.
func (term *Terminal) Press(keys ...Key) {
	term.t.Helper()
	strs := make([]string, len(keys))
	for i, k := range keys {
		strs[i] = string(k)
	}
	term.SendKeys(strs...)
}

// Screen captures the current terminal content and returns it.
func (term *Terminal) Screen() *Screen {
	term.t.Helper()
	return term.captureScreen("capture")
}

// captureScreen captures the current screen content and cursor position.
func (term *Terminal) captureScreen(op string) *Screen {
	term.t.Helper()
	term.requireAlive(op)

	raw, err := capturePaneContent(term.runner, term.pane)
	if err != nil {
		term.t.Fatalf("crawler: %s: %v", op, err)
	}

	scr := newScreen(raw, term.opts.width, term.opts.height)

	// Fetch cursor position (best-effort; don't fail if unavailable).
	row, col, cursorErr := getCursorPosition(term.runner, term.pane)
	if cursorErr == nil {
		scr.cursorRow = row
		scr.cursorCol = col
	}

	return scr
}

// captureScreenRaw captures screen content without requiring the pane to be alive.
// Used in error reporting paths where the pane may have died.
func (term *Terminal) captureScreenRaw() *Screen {
	raw, err := capturePaneContent(term.runner, term.pane)
	if err != nil {
		return nil
	}
	scr := newScreen(raw, term.opts.width, term.opts.height)
	row, col, cursorErr := getCursorPosition(term.runner, term.pane)
	if cursorErr == nil {
		scr.cursorRow = row
		scr.cursorCol = col
	}
	return scr
}

// WaitFor polls the screen until the matcher succeeds or the timeout expires.
// On timeout it calls t.Fatal with a description of what was expected
// and the last screen content.
func (term *Terminal) WaitFor(m Matcher, wopts ...WaitOption) {
	term.t.Helper()
	_ = term.waitForInternal(m, wopts...)
}

// WaitForScreen has the same timeout behavior as WaitFor: it polls until the
// matcher succeeds or the timeout expires, calling t.Fatal on timeout. On
// success it returns the matching Screen.
func (term *Terminal) WaitForScreen(m Matcher, wopts ...WaitOption) *Screen {
	term.t.Helper()
	return term.waitForInternal(m, wopts...)
}

func (term *Terminal) waitForInternal(m Matcher, wopts ...WaitOption) *Screen {
	term.t.Helper()

	wo := waitOptions{}
	for _, o := range wopts {
		o(&wo)
	}

	timeout := term.opts.timeout
	if wo.timeout > 0 {
		timeout = wo.timeout
	} else if wo.timeout < 0 {
		term.t.Fatalf("crawler: wait-for: negative timeout: %v", wo.timeout)
	}

	pollInterval := term.opts.pollInterval
	if wo.pollInterval > 0 {
		pollInterval = wo.pollInterval
		if pollInterval < minPollInterval {
			pollInterval = minPollInterval
		}
	} else if wo.pollInterval < 0 {
		term.t.Fatalf("crawler: wait-for: negative poll interval: %v", wo.pollInterval)
	}

	deadline := time.Now().Add(timeout)
	var lastScreen *Screen
	lastDesc := "matcher condition"
	recentScreens := make([]*Screen, 0, failureCaptureHistory)

	for {
		// Check if pane is dead.
		state, err := getPaneState(term.runner, term.pane)
		if err == nil && state.dead {
			lastScreen = term.captureScreenRaw()
			recentScreens = appendRecentScreens(recentScreens, lastScreen, failureCaptureHistory)
			if lastScreen != nil {
				_, lastDesc = m(lastScreen)
			}
			term.t.Fatalf("crawler: wait-for: process exited unexpectedly (status %d)\n    waiting for: %s\n    recent screen captures (oldest to newest):\n%s",
				state.exitStatus, lastDesc, formatRecentScreens(recentScreens))
		}

		raw, captureErr := capturePaneContent(term.runner, term.pane)
		if captureErr != nil {
			term.t.Fatalf("crawler: wait-for: capture failed: %v", captureErr)
		}

		lastScreen = newScreen(raw, term.opts.width, term.opts.height)
		// Fetch cursor for cursor matchers.
		row, col, cursorErr := getCursorPosition(term.runner, term.pane)
		if cursorErr == nil {
			lastScreen.cursorRow = row
			lastScreen.cursorCol = col
		}
		recentScreens = appendRecentScreens(recentScreens, lastScreen, failureCaptureHistory)

		ok, desc := m(lastScreen)
		lastDesc = desc
		if ok {
			return lastScreen
		}

		if time.Now().After(deadline) {
			term.t.Fatalf("crawler: wait-for: timed out after %v\n    waiting for: %s\n    recent screen captures (oldest to newest):\n%s",
				timeout, lastDesc, formatRecentScreens(recentScreens))
		}

		time.Sleep(pollInterval)
	}
}

// WaitExit waits for the TUI process to exit and returns its exit code.
// Useful for testing that a program terminates cleanly.
func (term *Terminal) WaitExit(wopts ...WaitOption) int {
	term.t.Helper()

	wo := waitOptions{}
	for _, o := range wopts {
		o(&wo)
	}

	timeout := term.opts.timeout
	if wo.timeout > 0 {
		timeout = wo.timeout
	} else if wo.timeout < 0 {
		term.t.Fatalf("crawler: wait-exit: negative timeout: %v", wo.timeout)
	}

	pollInterval := term.opts.pollInterval
	if wo.pollInterval > 0 {
		pollInterval = wo.pollInterval
		if pollInterval < minPollInterval {
			pollInterval = minPollInterval
		}
	} else if wo.pollInterval < 0 {
		term.t.Fatalf("crawler: wait-exit: negative poll interval: %v", wo.pollInterval)
	}

	deadline := time.Now().Add(timeout)
	recentScreens := make([]*Screen, 0, failureCaptureHistory)
	for {
		state, err := getPaneState(term.runner, term.pane)
		if err != nil {
			term.t.Fatalf("crawler: wait-exit: %v", err)
		}
		if state.dead {
			return state.exitStatus
		}
		recentScreens = appendRecentScreens(recentScreens, term.captureScreenRaw(), failureCaptureHistory)
		if time.Now().After(deadline) {
			term.t.Fatalf("crawler: wait-exit: timed out after %v\n    pane still alive\n    recent screen captures (oldest to newest):\n%s",
				timeout, formatRecentScreens(recentScreens))
		}
		time.Sleep(pollInterval)
	}
}

// Resize changes the terminal dimensions.
// This sends a SIGWINCH to the running program.
func (term *Terminal) Resize(width, height int) {
	term.t.Helper()
	term.requireAlive("resize")
	if err := resizeWindow(term.runner, term.pane, width, height); err != nil {
		term.t.Fatalf("crawler: resize: %v", err)
	}
	term.opts.width = width
	term.opts.height = height
}

// Scrollback captures the full scrollback buffer, not just the visible screen.
//
// The returned Screen has one line per scrollback row (oldest to newest).
// Its height (and len(Lines())) reflects the total number of captured lines,
// which is typically larger than the pane's visible height. Width is the
// maximum line width across all captured lines. Callers should use
// len(s.Lines()) to reason about scrollback length, rather than relying on
// the visible height returned by s.Size().
func (term *Terminal) Scrollback() *Screen {
	term.t.Helper()
	term.requireAlive("capture")

	raw, err := capturePaneScrollback(term.runner, term.pane)
	if err != nil {
		term.t.Fatalf("crawler: capture: scrollback: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(raw, "\n"), "\n")
	maxWidth := 0
	for _, l := range lines {
		if len(l) > maxWidth {
			maxWidth = len(l)
		}
	}

	return newScreen(raw, maxWidth, len(lines))
}

// requireAlive checks that the pane process is still running and calls t.Fatal
// if it has exited.
func (term *Terminal) requireAlive(op string) {
	term.t.Helper()

	state, err := getPaneState(term.runner, term.pane)
	if err != nil {
		return
	}
	if state.dead {
		term.t.Fatalf("crawler: %s: process exited unexpectedly (status %d)", op, state.exitStatus)
	}
}

func appendRecentScreens(screens []*Screen, scr *Screen, max int) []*Screen {
	if scr == nil {
		return screens
	}
	screens = append(screens, scr)
	if len(screens) > max {
		screens = screens[len(screens)-max:]
	}
	return screens
}

func formatRecentScreens(screens []*Screen) string {
	if len(screens) == 0 {
		return "    (no screen captured)"
	}

	var b strings.Builder
	for i, scr := range screens {
		fmt.Fprintf(&b, "    capture %d/%d:\n%s", i+1, len(screens), formatScreenBox(scr))
		if i < len(screens)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// formatScreenBox formats a screen capture with a box border for error messages.
func formatScreenBox(scr *Screen) string {
	if scr == nil {
		return "    (no screen captured)"
	}

	width, _ := scr.Size()
	if width == 0 {
		width = 80
	}

	var b strings.Builder
	border := strings.Repeat("\u2500", width)

	fmt.Fprintf(&b, "    \u250c%s\u2510\n", border)
	for _, line := range scr.Lines() {
		padded := line
		if len(padded) < width {
			padded += strings.Repeat(" ", width-len(padded))
		}
		fmt.Fprintf(&b, "    \u2502%s\u2502\n", padded)
	}
	fmt.Fprintf(&b, "    \u2514%s\u2518", border)

	return b.String()
}
