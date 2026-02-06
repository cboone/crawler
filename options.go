package crawler

import "time"

type options struct {
	args         []string
	width        int
	height       int
	env          []string
	dir          string
	timeout      time.Duration
	pollInterval time.Duration
	tmuxPath     string
	historyLimit int
}

// Option configures a Terminal created by Open.
type Option func(*options)

// WithArgs sets the arguments passed to the binary.
func WithArgs(args ...string) Option {
	return func(o *options) {
		o.args = args
	}
}

// WithSize sets the terminal dimensions (columns x rows).
func WithSize(width, height int) Option {
	return func(o *options) {
		o.width = width
		o.height = height
	}
}

// WithEnv appends environment variables to the process environment.
// Each entry should be in "KEY=VALUE" format.
func WithEnv(env ...string) Option {
	return func(o *options) {
		o.env = append(o.env, env...)
	}
}

// WithDir sets the working directory for the binary.
func WithDir(dir string) Option {
	return func(o *options) {
		o.dir = dir
	}
}

// WithTimeout sets the default timeout for WaitFor and WaitForScreen.
func WithTimeout(d time.Duration) Option {
	return func(o *options) {
		o.timeout = d
	}
}

// WithPollInterval sets the default polling interval for WaitFor and WaitForScreen.
func WithPollInterval(d time.Duration) Option {
	return func(o *options) {
		o.pollInterval = d
	}
}

// WithTmuxPath sets the path to the tmux binary. Defaults to "tmux"
// (resolved via $PATH). The CRAWLER_TMUX environment variable can also
// be used as a fallback before the default.
func WithTmuxPath(path string) Option {
	return func(o *options) {
		o.tmuxPath = path
	}
}

// WithHistoryLimit sets the tmux scrollback history limit for the test session.
// A value of 0 uses the default set by Open (10000).
func WithHistoryLimit(limit int) Option {
	return func(o *options) {
		o.historyLimit = limit
	}
}

// WaitOption configures a single WaitFor, WaitForScreen, or WaitExit call.
type WaitOption func(*waitOptions)

type waitOptions struct {
	timeout      time.Duration
	pollInterval time.Duration
}

// WithinTimeout overrides the call timeout for a single wait call.
// A value of 0 means "use defaults". Negative values cause t.Fatal.
func WithinTimeout(d time.Duration) WaitOption {
	return func(o *waitOptions) {
		o.timeout = d
	}
}

// WithWaitPollInterval overrides the polling interval for a single wait call.
// A value of 0 means "use defaults". Negative values cause t.Fatal.
// Positive values under 10ms are clamped to 10ms.
func WithWaitPollInterval(d time.Duration) WaitOption {
	return func(o *waitOptions) {
		o.pollInterval = d
	}
}

const (
	defaultWidth        = 80
	defaultHeight       = 24
	defaultTimeout      = 5 * time.Second
	defaultPollInterval = 50 * time.Millisecond
	defaultHistoryLimit = 10000
	minPollInterval     = 10 * time.Millisecond
)

func defaultOptions() options {
	return options{
		width:        defaultWidth,
		height:       defaultHeight,
		timeout:      defaultTimeout,
		pollInterval: defaultPollInterval,
		historyLimit: defaultHistoryLimit,
	}
}
