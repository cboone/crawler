package tmuxcli_test

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/cboone/crawler/internal/tmuxcli"
)

func findTmux(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("tmux")
	if err != nil {
		t.Skip("tmux not found in PATH")
	}
	return path
}

func TestVersion(t *testing.T) {
	tmuxPath := findTmux(t)
	version, err := tmuxcli.Version(tmuxPath)
	if err != nil {
		t.Fatalf("Version() error: %v", err)
	}
	if version == "" {
		t.Fatal("Version() returned empty string")
	}
	// Should contain a number.
	if !strings.ContainsAny(version, "0123456789") {
		t.Errorf("Version() = %q, expected to contain digits", version)
	}
}

func TestRunnerBasic(t *testing.T) {
	tmuxPath := findTmux(t)

	// Create a temp socket path.
	socketPath := t.TempDir() + "/test.sock"

	runner := tmuxcli.New(tmuxPath, socketPath)

	if runner.SocketPath() != socketPath {
		t.Errorf("SocketPath() = %q, want %q", runner.SocketPath(), socketPath)
	}
	if runner.TmuxPath() != tmuxPath {
		t.Errorf("TmuxPath() = %q, want %q", runner.TmuxPath(), tmuxPath)
	}

	// Start a session.
	_, err := runner.Run("new-session", "-d", "-x", "80", "-y", "24", "-E", "--", "/bin/sh")
	if err != nil {
		t.Fatalf("Failed to start session: %v", err)
	}

	// Wait for session.
	if err := runner.WaitForSession(5 * time.Second); err != nil {
		t.Fatalf("WaitForSession: %v", err)
	}

	// List panes.
	output, err := runner.Run("list-panes", "-F", "#{pane_id}")
	if err != nil {
		t.Fatalf("list-panes: %v", err)
	}
	paneID := strings.TrimSpace(output)
	if paneID == "" {
		t.Fatal("no pane ID returned")
	}

	// Cleanup.
	_, _ = runner.Run("kill-server")
}

func TestRunnerError(t *testing.T) {
	tmuxPath := findTmux(t)
	socketPath := t.TempDir() + "/nonexistent.sock"

	runner := tmuxcli.New(tmuxPath, socketPath)

	// Trying to list panes on a non-existent session should fail.
	_, err := runner.Run("list-panes")
	if err == nil {
		t.Fatal("expected error for non-existent session")
	}

	// Should be a *tmuxcli.Error.
	tmuxErr, ok := err.(*tmuxcli.Error)
	if !ok {
		t.Fatalf("expected *tmuxcli.Error, got %T", err)
	}
	if tmuxErr.Op != "list-panes" {
		t.Errorf("Op = %q, want %q", tmuxErr.Op, "list-panes")
	}
}
