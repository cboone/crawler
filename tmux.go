package crawler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/cboone/crawler/internal/tmuxcli"
)

const minTmuxVersion = "3.0"

// resolveTmuxPath determines the tmux binary path by checking, in order:
// 1. WithTmuxPath option
// 2. CRAWLER_TMUX environment variable
// 3. $PATH lookup
//
// Returns the resolved path and whether it was explicitly configured.
func resolveTmuxPath(t testing.TB, configured string) (path string, explicit bool) {
	t.Helper()

	if configured != "" {
		return configured, true
	}

	if envPath := os.Getenv("CRAWLER_TMUX"); envPath != "" {
		return envPath, true
	}

	found, err := exec.LookPath("tmux")
	if err != nil {
		t.Skip("crawler: open: tmux not found")
	}
	return found, false
}

// checkTmuxVersion verifies the tmux version meets the minimum requirement.
func checkTmuxVersion(t testing.TB, tmuxPath string, explicit bool) {
	t.Helper()

	version, err := tmuxcli.Version(tmuxPath)
	if err != nil {
		if explicit {
			t.Fatalf("crawler: open: %v", err)
		}
		t.Skipf("crawler: open: %v", err)
	}

	if !versionAtLeast(version, minTmuxVersion) {
		msg := fmt.Sprintf("crawler: open: tmux version %s is below minimum %s", version, minTmuxVersion)
		if explicit {
			t.Fatal(msg)
		}
		t.Skip(msg)
	}
}

// versionAtLeast returns true if version >= minVersion.
// Handles version strings like "3.4", "next-3.5", "3.3a".
var versionRe = regexp.MustCompile(`(\d+)\.(\d+)`)

func versionAtLeast(version, minVersion string) bool {
	parseMajorMinor := func(v string) (int, int, bool) {
		m := versionRe.FindStringSubmatch(v)
		if m == nil {
			return 0, 0, false
		}
		major, _ := strconv.Atoi(m[1])
		minor, _ := strconv.Atoi(m[2])
		return major, minor, true
	}

	vMajor, vMinor, ok1 := parseMajorMinor(version)
	mMajor, mMinor, ok2 := parseMajorMinor(minVersion)
	if !ok1 || !ok2 {
		return false
	}

	if vMajor != mMajor {
		return vMajor > mMajor
	}
	return vMinor >= mMinor
}

// generateSocketPath creates a unique, filesystem-safe socket path.
func generateSocketPath(t testing.TB) string {
	t.Helper()

	sanitized := sanitizeName(t.Name())

	// Generate random suffix.
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("crawler: open: failed to generate random bytes: %v", err)
	}
	suffix := hex.EncodeToString(b)

	name := fmt.Sprintf("crawler-%s-%s.sock", sanitized, suffix)
	path := filepath.Join(os.TempDir(), name)

	// Handle collision: if file exists, regenerate.
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path
		}
		if _, err := rand.Read(b); err != nil {
			t.Fatalf("crawler: open: failed to generate random bytes: %v", err)
		}
		suffix = hex.EncodeToString(b)
		name = fmt.Sprintf("crawler-%s-%s.sock", sanitized, suffix)
		path = filepath.Join(os.TempDir(), name)
	}

	// Extremely unlikely: 10 collisions in a row.
	t.Fatalf("crawler: open: could not generate unique socket path after 10 attempts")
	return ""
}

// sanitizeName replaces characters that are not filesystem-safe.
func sanitizeName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	s := b.String()
	// Truncate to avoid overly long socket paths (Unix has a 104/108 char limit).
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}

// writeConfig writes a tmux config file with the needed session options.
func writeConfig(configPath string, opts options) error {
	histLimit := opts.historyLimit
	if histLimit == 0 {
		histLimit = defaultHistoryLimit
	}

	config := fmt.Sprintf("set-option -g history-limit %d\nset-option -g remain-on-exit on\nset-option -g status off\n", histLimit)
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		return fmt.Errorf("crawler: open: failed to write tmux config: %w", err)
	}
	return nil
}

// startSession starts a new tmux session with the given configuration.
func startSession(runner *tmuxcli.Runner, binary string, opts options) error {
	args := []string{
		"new-session", "-d",
		"-x", strconv.Itoa(opts.width),
		"-y", strconv.Itoa(opts.height),
	}

	// Set working directory if specified.
	if opts.dir != "" {
		args = append(args, "-c", opts.dir)
	}

	// Build the command to run.
	args = append(args, "--", binary)
	args = append(args, opts.args...)

	if _, err := runner.Run(args...); err != nil {
		return fmt.Errorf("crawler: open: failed to start tmux session: %w", err)
	}

	return nil
}

// setSessionEnv sets environment variables on the tmux session.
func setSessionEnv(runner *tmuxcli.Runner, env []string) error {
	for _, e := range env {
		if _, err := runner.Run("set-environment", e); err != nil {
			return fmt.Errorf("crawler: open: failed to set environment: %w", err)
		}
	}
	return nil
}

// capturePaneContent captures the visible pane content.
func capturePaneContent(runner *tmuxcli.Runner, pane string) (string, error) {
	return runner.Run("capture-pane", "-p", "-t", pane)
}

// capturePaneScrollback captures the full scrollback buffer.
func capturePaneScrollback(runner *tmuxcli.Runner, pane string) (string, error) {
	return runner.Run("capture-pane", "-p", "-t", pane, "-S", "-", "-E", "-")
}

// sendKeys sends key sequences to the pane.
func sendKeys(runner *tmuxcli.Runner, pane string, keys []string) error {
	args := append([]string{"send-keys", "-t", pane}, keys...)
	_, err := runner.Run(args...)
	return err
}

// resizeWindow resizes the terminal window.
func resizeWindow(runner *tmuxcli.Runner, pane string, width, height int) error {
	_, err := runner.Run("resize-window", "-t", pane, "-x", strconv.Itoa(width), "-y", strconv.Itoa(height))
	return err
}

// paneState holds the dead status and exit code of a pane.
type paneState struct {
	dead       bool
	exitStatus int
}

// getPaneState queries the pane state.
func getPaneState(runner *tmuxcli.Runner, pane string) (paneState, error) {
	output, err := runner.Run("list-panes", "-t", pane, "-F", "#{pane_dead} #{pane_dead_status}")
	if err != nil {
		return paneState{}, err
	}

	line := strings.TrimSpace(output)
	parts := strings.SplitN(line, " ", 2)

	dead := parts[0] == "1"
	status := 0
	if dead && len(parts) >= 2 {
		status, _ = strconv.Atoi(parts[1])
	}

	return paneState{dead: dead, exitStatus: status}, nil
}

// getCursorPosition queries the cursor position.
func getCursorPosition(runner *tmuxcli.Runner, pane string) (row, col int, err error) {
	output, err := runner.Run("display-message", "-p", "-t", pane, "#{cursor_x} #{cursor_y}")
	if err != nil {
		return 0, 0, err
	}

	line := strings.TrimSpace(output)
	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("unexpected display-message output: %q", line)
	}

	col, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parsing cursor_x: %w", err)
	}
	row, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parsing cursor_y: %w", err)
	}

	return row, col, nil
}

// killServer kills the tmux server.
func killServer(runner *tmuxcli.Runner) error {
	_, err := runner.Run("kill-server")
	return err
}
