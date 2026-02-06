// Package tmuxcli provides low-level tmux command execution and socket-path
// management. It is internal to the crawler package.
package tmuxcli

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Runner executes tmux commands against a specific server socket.
type Runner struct {
	tmuxPath   string
	socketPath string
	configPath string
}

// New creates a Runner bound to the given tmux binary and socket path.
func New(tmuxPath, socketPath string) *Runner {
	return &Runner{
		tmuxPath:   tmuxPath,
		socketPath: socketPath,
	}
}

// SetConfigPath sets the path to a tmux config file. When set, all tmux
// invocations will include -f <configPath> before other arguments.
func (r *Runner) SetConfigPath(path string) {
	r.configPath = path
}

// Run executes a tmux command with the given arguments and returns its
// combined stdout output. If the command fails, it returns an error
// containing stderr.
func (r *Runner) Run(args ...string) (string, error) {
	return r.RunContext(context.Background(), args...)
}

// RunContext executes a tmux command with the given context and arguments.
func (r *Runner) RunContext(ctx context.Context, args ...string) (string, error) {
	var fullArgs []string
	if r.configPath != "" {
		fullArgs = append(fullArgs, "-f", r.configPath)
	}
	fullArgs = append(fullArgs, "-S", r.socketPath)
	fullArgs = append(fullArgs, args...)
	cmd := exec.CommandContext(ctx, r.tmuxPath, fullArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", &Error{
			Op:     args[0],
			Args:   fullArgs,
			Stderr: strings.TrimSpace(stderr.String()),
			Err:    err,
		}
	}

	return stdout.String(), nil
}

// SocketPath returns the socket path used by this runner.
func (r *Runner) SocketPath() string {
	return r.socketPath
}

// TmuxPath returns the path to the tmux binary.
func (r *Runner) TmuxPath() string {
	return r.tmuxPath
}

// Error represents a tmux command failure.
type Error struct {
	Op     string
	Args   []string
	Stderr string
	Err    error
}

func (e *Error) Error() string {
	msg := fmt.Sprintf("tmux %s failed: %v", e.Op, e.Err)
	if e.Stderr != "" {
		msg += "\nstderr: " + e.Stderr
	}
	return msg
}

func (e *Error) Unwrap() error {
	return e.Err
}

// Version runs "tmux -V" and returns the version string (e.g. "3.4").
func Version(tmuxPath string) (string, error) {
	cmd := exec.Command(tmuxPath, "-V")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tmux -V failed: %v (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	// Output is like "tmux 3.4" or "tmux next-3.5"
	output := strings.TrimSpace(stdout.String())
	version := strings.TrimPrefix(output, "tmux ")
	return version, nil
}

// WaitForSession polls until the tmux session is ready or the timeout expires.
func (r *Runner) WaitForSession(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		_, err := r.Run("list-panes", "-F", "#{pane_id}")
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("tmux session not ready after %v: %w", timeout, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
