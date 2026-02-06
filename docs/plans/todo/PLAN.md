# crawler — Playwright for TUIs via tmux

A Go testing library for black-box testing of terminal user interfaces.
Tests run real binaries inside tmux sessions, send keystrokes, capture screen
output, and assert against it — all through the standard `testing.TB` interface.

---

## Goals

1. **Framework-agnostic**: Test any TUI binary (bubbletea, tview, tcell,
   Python curses, Rust ratatui, raw ANSI programs — anything).
2. **Go-native API**: First-class integration with `*testing.T`, subtests,
   table-driven tests, `t.Helper()`, `t.Cleanup()`. No DSLs.
3. **Reliable**: Deterministic waits instead of `time.Sleep`. Automatic retry
   with timeouts, like Playwright's auto-waiting locators.
4. **Snapshot testing**: Golden-file screen captures with `CRAWLER_UPDATE=1`.
5. **Simple internals**: Shell out to the `tmux` CLI. No cgo, no terminfo
   parsing, no terminal emulator reimplementation.

## Non-goals

- Replacing unit-test tools like teatest or tcell's SimulationScreen.
  crawler is for integration/end-to-end testing of compiled binaries.
- Windows support. tmux is Unix-only; that is an accepted constraint.
- Parsing or understanding ANSI escape sequences or styled output.
  Screen content is plain text as returned by `tmux capture-pane -p`.

## Constraints

- **Minimum Go version**: 1.21+ (for `testing.TB` improvements and `slices` if
  needed from stdlib).
- **Minimum tmux version**: 3.0+ (released November 2019). Covers all needed
  features including `capture-pane -p`, `resize-window`, and `list-panes`
  format strings. Checked at runtime in `Open`.
- **Supported OS**: Linux, macOS. Any Unix-like system where tmux runs.

---

## Architecture

```
  Go test process
  ┌──────────────────────────────────┐
  │  func TestFoo(t *testing.T) {    │
  │      term := crawler.Open(t, …)  │──── tmux -S <socket-path> new-session -d …
  │      term.WaitFor(Text("hello")) │──── tmux -S <socket-path> capture-pane -p
  │      term.SendKeys("world")      │──── tmux -S <socket-path> send-keys …
  │      term.Screen().Match(…)      │──── tmux -S <socket-path> capture-pane -p
  │  }                               │
  └──────────────────────────────────┘
                    │
                    ▼
  tmux server (per-test isolated socket path)
  ┌──────────────────────────────────┐
  │  session: "test-<random>"        │
  │  window 0, pane %0               │
  │  ┌────────────────────────────┐  │
  │  │  $ ./my-tui-binary --flag  │  │
  │  │  ┌──────────────────────┐  │  │
  │  │  │  TUI rendering here  │  │  │
  │  │  └──────────────────────┘  │  │
  │  └────────────────────────────┘  │
  └──────────────────────────────────┘
```

Each test (or subtest) gets its own tmux server via a unique `-S` socket path
under `os.TempDir()`.
This provides complete isolation — tests cannot interfere with each other or
with the user's tmux sessions.

### Why tmux (not a PTY or terminal emulator)?

| Approach | Pros | Cons |
|----------|------|------|
| Raw PTY + VT parser | No external deps | Must reimplement terminal emulation; fragile |
| tcell SimScreen | In-process, fast | Only works with tcell-based programs |
| xterm.js (à la tui-test) | Full emulation | Requires Node.js |
| **tmux** | Battle-tested rendering; plain-text capture; works with any binary; already handles resize, scrollback, alternate screen | Requires tmux installed; Unix-only |

tmux has been solving "what does this terminal look like right now?" for
decades. We get that for free with `capture-pane`.

---

## Package structure

```
crawler/
├── go.mod                  # module github.com/cboone/crawler
├── go.sum
├── crawler.go              # Terminal type, Open(), core API
├── crawler_test.go         # Tests for the library itself
├── options.go              # Option type and functional options
├── screen.go               # Screen type (captured pane content)
├── keys.go                 # Key constants and SendKeys helpers
├── match.go                # Matchers: Text, Regexp, Line, etc.
├── snapshot.go             # Golden-file snapshot support
├── tmux.go                 # Package-level tmux session helpers (unexported); thin adapter over internal/tmuxcli
├── doc.go                  # Package documentation
├── internal/
│   └── tmuxcli/
│       ├── tmuxcli.go      # Low-level tmux command execution and socket-path management (used only by tmux.go)
│       └── tmuxcli_test.go
└── testdata/               # Golden files for the library's own tests
    └── ...
```

Single package: `crawler`. Users import one thing:

```go
import "github.com/cboone/crawler"
```

---

## Core types

### Terminal

The primary handle to a running TUI under test.

```go
// Terminal is a handle to a TUI program running inside a tmux session.
// It is created with Open and cleaned up automatically via t.Cleanup.
type Terminal struct {
    t      testing.TB
    socketPath string // tmux -S socket path
    pane   string   // tmux pane ID (e.g. "%0")
    opts   options
}
```

### Screen

An immutable snapshot of the terminal's visible content at a point in time.

```go
// Screen is an immutable capture of terminal content.
type Screen struct {
    lines []string  // one entry per row
    raw   string    // full capture joined by newlines
    width int
    height int
}
```

### Matcher

A function type for matching screen content. Composable.

```go
// A Matcher reports whether a Screen satisfies a condition.
// The string return is a human-readable description for error messages.
type Matcher func(s *Screen) (ok bool, description string)
```

### Options

Configuration passed to Open via functional options.

```go
type options struct {
    args         []string      // arguments to the binary
    width        int           // terminal columns (default 80)
    height       int           // terminal rows (default 24)
    env          []string      // additional environment variables (appended to current env)
    dir          string        // working directory for the process
    timeout      time.Duration // default timeout for WaitFor (default 5s)
    pollInterval time.Duration // poll interval for WaitFor (default 50ms)
    tmuxPath     string        // path to tmux binary (default: "tmux", resolved via $PATH)
}
```

---

## API design

### Opening and closing

```go
// Open starts the binary in a new tmux session.
// Cleanup is automatic via t.Cleanup — no defer needed.
func Open(t testing.TB, binary string, opts ...Option) *Terminal
```

Basic usage:

```go
func TestMyApp(t *testing.T) {
    term := crawler.Open(t, "path/to/binary",
        crawler.WithArgs("arg1", "arg2"),
    )
}
```

With more options:

```go
term := crawler.Open(t, "./my-app",
    crawler.WithArgs("--verbose"),
    crawler.WithSize(120, 40),
    crawler.WithEnv("NO_COLOR=1", "TERM=xterm-256color"),
    crawler.WithDir("/tmp/workdir"),
    crawler.WithTimeout(10 * time.Second),
    crawler.WithTmuxPath("/opt/homebrew/bin/tmux"),
)
```

`WithEnv` values are **appended** to the current process environment.
They do not replace it. This matches `exec.Cmd.Env` semantics when
combined with `os.Environ()`.

`WithTmuxPath` allows specifying a non-standard tmux binary location.
Defaults to `"tmux"` (resolved via `$PATH`). The `CRAWLER_TMUX`
environment variable can also be used as a fallback before the default.

**Implementation**: `Open` generates a unique socket path (for example,
`<tmp>/crawler-<test>-<random>.sock`) and calls
`tmux -S <socket-path> new-session -d -x <w> -y <h> -- <binary> <args...>`,
waits for the session to be ready, and registers a `t.Cleanup` that calls
`tmux -S <socket-path> kill-server`.

Socket files are explicitly placed under `os.TempDir()`.
`Open` removes any stale socket file before starting tmux if the path exists.

### Sending input

```go
// Type sends a string as sequential keypresses.
func (term *Terminal) Type(s string)

// Press sends one or more special keys.
func (term *Terminal) Press(keys ...Key)

// SendKeys sends raw tmux key sequences. Escape hatch for advanced use.
func (term *Terminal) SendKeys(keys ...string)
```

Usage:

```go
term := crawler.Open(t, "./my-app")
term.Type("hello world")
term.Press(crawler.Enter)
term.Press(crawler.Ctrl('c'))
term.Press(crawler.Tab, crawler.Tab, crawler.Enter)
term.Press(crawler.Up, crawler.Up, crawler.Down)
```

**Key constants** follow Go naming conventions:

```go
const (
    Enter     Key = "Enter"
    Escape    Key = "Escape"
    Tab       Key = "Tab"
    Backspace Key = "BSpace"
    Up        Key = "Up"
    Down      Key = "Down"
    Left      Key = "Left"
    Right     Key = "Right"
    Home      Key = "Home"
    End       Key = "End"
    PageUp    Key = "PageUp"
    PageDown  Key = "PageDown"
    Space     Key = "Space"
    F1        Key = "F1"
    // ... through F12
)

// Ctrl returns the key sequence for Ctrl+<char>.
func Ctrl(c byte) Key

// Alt returns the key sequence for Alt+<char>.
func Alt(c byte) Key
```

### Capturing the screen

```go
// Screen captures the current terminal content and returns it.
func (term *Terminal) Screen() *Screen
```

This calls `tmux capture-pane -p -t <pane>` and parses the output.

Capture rules are explicit to keep behavior deterministic:

- `Screen()` captures the visible pane only, one string line per terminal row
  (wrapped rows are **not** joined).
- Line endings are normalized to `\n` and the single terminal newline emitted
  by `capture-pane` is removed from the in-memory representation.
- Leading and interior spaces are preserved exactly.
- Snapshot serialization applies additional normalization for stable diffs:
  trailing spaces on each line are trimmed, trailing blank lines are removed,
  and files are written with a single final newline.

### Screen inspection

```go
// String returns the full screen content as a string.
func (s *Screen) String() string

// Lines returns a copy of the screen content as a slice of strings, one per row.
// The returned slice is a shallow copy; callers may modify it without affecting
// the Screen.
func (s *Screen) Lines() []string

// Line returns the content of a single row (0-indexed).
func (s *Screen) Line(n int) string

// Contains reports whether the screen contains the substring.
func (s *Screen) Contains(substr string) bool

// Size returns the width and height.
func (s *Screen) Size() (width, height int)
```

These are pure accessors for use in manual assertions:

```go
screen := term.Screen()
if !screen.Contains("Welcome") {
    t.Errorf("expected Welcome on screen, got:\n%s", screen)
}
```

### Waiting (the core reliability mechanism)

```go
// WaitFor polls the screen until the matcher succeeds or the timeout expires.
// On timeout it calls t.Fatal with a description of what was expected
// and the last screen content.
func (term *Terminal) WaitFor(m Matcher, opts ...WaitOption)

// WaitForScreen has the same timeout behavior as WaitFor: it polls until the
// matcher succeeds or the timeout expires, calling t.Fatal on timeout. On
// success it returns the matching Screen.
func (term *Terminal) WaitForScreen(m Matcher, opts ...WaitOption) *Screen
```

`WaitOption` is shared by `WaitFor`, `WaitForScreen`, and `WaitExit`:

```go
// WithinTimeout overrides the call timeout.
func WithinTimeout(d time.Duration) WaitOption

// WithPollInterval overrides the polling interval for this call.
func WithPollInterval(d time.Duration) WaitOption
```

Wait option semantics:

- Defaults come from terminal options (`WithTimeout`, `WithPollInterval`),
  or library defaults (5s timeout, 50ms poll interval).
- `WithinTimeout(0)` and `WithPollInterval(0)` mean "use defaults".
- Negative values are invalid and cause an immediate `t.Fatal`.
- Positive poll intervals under 10ms are clamped to 10ms to prevent busy-loop
  polling.

Usage:

```go
// Wait for text to appear
term.WaitFor(crawler.Text("Loading complete"))

// Wait with a longer timeout
term.WaitFor(crawler.Text("Done"), crawler.WithinTimeout(30*time.Second))

// Wait for a regex
term.WaitFor(crawler.Regexp(`\d+ items loaded`))

// Wait for text on a specific line
term.WaitFor(crawler.LineContains(0, "My App v1.0"))

// Wait for text to disappear
term.WaitFor(crawler.Not(crawler.Text("Loading...")))

// Capture the matching screen
screen := term.WaitForScreen(crawler.Text("Results"))
```

**Failure output** is designed to be immediately useful:

```
terminal_test.go:42: WaitFor timed out after 5s
    waiting for: screen to contain "Loading complete"
    last screen capture:
    ┌────────────────────────────────────────────────────────────────────────────────┐
    │ My Application v1.0                                                           │
    │                                                                               │
    │ Loading...                                                                    │
    │                                                                               │
    └────────────────────────────────────────────────────────────────────────────────┘
```

### Built-in matchers

```go
// Text matches if the screen contains the given substring anywhere.
func Text(s string) Matcher

// Regexp matches if the screen content matches the regular expression.
func Regexp(pattern string) Matcher

// Line matches if the given line (0-indexed) equals s after trimming
// trailing spaces from the screen line.
func Line(n int, s string) Matcher

// LineContains matches if the given line contains the substring.
func LineContains(n int, substr string) Matcher

// Not inverts a matcher.
func Not(m Matcher) Matcher

// All matches when every provided matcher matches.
func All(matchers ...Matcher) Matcher

// Any matches when at least one provided matcher matches.
func Any(matchers ...Matcher) Matcher

// Empty matches when the screen has no visible content.
func Empty() Matcher

// Cursor matches if the cursor is at the given position.
// Uses tmux display-message -p -t <pane> '#{cursor_x} #{cursor_y}'.
// Note: tmux reports cursor position as (x, y) = (col, row), but this
// matcher takes (row, col) to follow the usual row-then-column convention.
func Cursor(row, col int) Matcher
```

### Snapshot testing

```go
// MatchSnapshot compares the current screen against a golden file
// stored in testdata/<sanitized-test-name>/<sanitized-name>.txt.
//
// Set CRAWLER_UPDATE=1 to create or update golden files.
func (term *Terminal) MatchSnapshot(name string)

// MatchSnapshot on Screen allows snapshotting a previously captured screen.
func (s *Screen) MatchSnapshot(t testing.TB, name string)
```

Usage:

```go
func TestWelcomeScreen(t *testing.T) {
    term := crawler.Open(t, "./my-app")
    term.WaitFor(crawler.Text("Welcome"))
    term.MatchSnapshot("welcome")
    // Compares against testdata/TestWelcomeScreen-1a2b3c4d/welcome.txt
}
```

Golden files are plain text — easy to review in diffs.

Snapshot paths are sanitized to avoid collisions and invalid paths:

- Base directory uses full `t.Name()` plus a short stable hash of the full
  test name (ensures path uniqueness while remaining stable across runs).
- `/` in subtest names is replaced to keep one directory level.
- Whitespace becomes `_`; characters outside `[A-Za-z0-9._-]` become `_`.
- Snapshot `name` is sanitized with the same character rules.

**Updating golden files**: Set the `CRAWLER_UPDATE` environment variable:

```sh
CRAWLER_UPDATE=1 go test ./...
```

This avoids requiring users to write a `TestMain` just to register a flag
(which conflicts when `TestMain` is already defined for other reasons).

### Convenience: multi-step interactions

For longer interaction sequences, a step-based helper avoids repetition:

```go
func TestFormFilling(t *testing.T) {
    term := crawler.Open(t, "./my-app")

    term.WaitFor(crawler.Text("Name:"))
    term.Type("Alice")
    term.Press(crawler.Tab)

    term.WaitFor(crawler.Text("Email:"))
    term.Type("alice@example.com")
    term.Press(crawler.Tab)

    term.WaitFor(crawler.Text("Submit"))
    term.Press(crawler.Enter)

    term.WaitFor(crawler.Text("Success"))
    term.MatchSnapshot("form-submitted")
}
```

### Subtests and table-driven tests

The API composes naturally with Go subtests:

```go
func TestNavigation(t *testing.T) {
    tests := []struct {
        name string
        key  crawler.Key
        want string
    }{
        {"down moves to second item", crawler.Down, "> Item 2"},
        {"up moves to first item", crawler.Up, "> Item 1"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            term := crawler.Open(t, "./my-list-app")
            term.WaitFor(crawler.Text("> Item 1"))
            term.Press(tt.key)
            term.WaitFor(crawler.Text(tt.want))
        })
    }
}
```

Each subtest gets its own tmux session, so they are fully independent.
`t.Parallel()` works — unique socket paths prevent collisions.

---

## Detailed feature design

### tmux session lifecycle

1. **Open**: Generate a unique socket path
   (`<tmp>/crawler-<test>-<random>.sock`).
   Run `tmux -S <socket-path> new-session -d -x <w> -y <h> -- <binary> <args>`.
   Poll `tmux -S <socket-path> list-panes` until the session is ready
   (typically near-instant). Record the pane ID.

2. **During test**: All operations target `-S <socket-path> -t <pane>`.

3. **Cleanup** (via `t.Cleanup`): Run `tmux -S <socket-path> kill-server`.
   This kills the tmux server and all processes within it. Socket files
   are cleaned up automatically by tmux.

### Error handling

All methods that interact with tmux check for errors and call `t.Fatal`
with a clear message. The user never has to check `err` returns — this
follows the pattern of `httptest.NewServer` and similar test helpers.

Error format is standardized across the package:

```text
crawler: <operation>: <reason>
command: tmux <args...>
stderr: <tmux stderr, if any>
```

`<operation>` is one of `open`, `send-keys`, `capture`, `wait-for`,
`wait-exit`, `resize`, or `snapshot`.

Process lifecycle semantics are explicit:

- `Screen`, `Type`, `Press`, `SendKeys`, and `Resize` fail immediately if the
  pane is dead: `crawler: <operation>: process exited unexpectedly (status N)`.
- `WaitFor` and `WaitForScreen` fail immediately if the pane dies before the
  matcher succeeds.
- `WaitExit` returns immediately if the process is already dead when called.
- Calling `WaitExit` is the expected API for tests that intentionally terminate
  the process; this is treated as a normal flow.
- `WaitExit` timeout is fatal and includes timeout, pane state, and last screen
  capture for diagnostics.

### Resize support

```go
// Resize changes the terminal dimensions.
// This sends a SIGWINCH to the running program.
func (term *Terminal) Resize(width, height int)
```

Implementation: `tmux -S <socket-path> resize-window -x <w> -y <h>`.

### Reading process exit

```go
// WaitExit waits for the TUI process to exit and returns its exit code.
// Useful for testing that a program terminates cleanly.
func (term *Terminal) WaitExit(opts ...WaitOption) int
```

Named `WaitExit` (not `Wait`) to clearly distinguish from `WaitFor`,
which waits for screen content.

Implementation: Poll
`tmux -S <socket-path> list-panes -F '#{pane_dead} #{pane_dead_status}'`
until the pane is marked dead, then return the status.

### Scrollback access

```go
// Scrollback captures the full scrollback buffer, not just the visible screen.
func (term *Terminal) Scrollback() *Screen
```

Implementation: `tmux capture-pane -p -S - -E -` (capture from start to end
of history).

Scrollback is bounded by tmux `history-limit`; it is not infinite.
To reduce environment variance, `Open` sets `history-limit` for the test
session to a deterministic default value (for example `10000`), which can be
overridden via a `WithHistoryLimit(limit int)` option if needed.

---

## Implementation phases

### Phase 1: Core

Minimum viable library. Enough to write real tests.

- [ ] `go.mod` initialization
- [ ] `internal/tmuxcli` — execute tmux commands, manage socket paths
- [ ] `Terminal` type with `Open` and `t.Cleanup` teardown
- [ ] `SendKeys`, `Type`, `Press` with key constants
- [ ] `Screen` type with `capture-pane` integration
- [ ] `Screen.Contains`, `Screen.String`, `Screen.Lines`, `Screen.Line`
- [ ] `WaitFor` with polling, timeout, and clear failure messages
- [ ] `Text` and `Regexp` matchers
- [ ] Basic integration tests (test the library against a small TUI program)

**Phase 1 acceptance criteria**: A user can write a test that opens a real
binary, sends keystrokes, waits for screen content, and asserts against it.
All of the following pass:
- Test that `Open` starts a tmux session and `t.Cleanup` kills it.
- Test that `Type` and `Press` produce the expected screen output.
- Test that `WaitFor` succeeds when content appears.
- Test that `WaitFor` calls `t.Fatal` with a useful message on timeout.
- Test that two parallel subtests do not interfere with each other.
- Stress test with at least 20 `t.Parallel()` subtests shows no cross-test
  leakage, and waiting does not poll faster than the configured interval.

### Phase 2: Matchers and snapshots

- [ ] `Line`, `LineContains`, `Not`, `All`, `Any`, `Empty` matchers
- [ ] `MatchSnapshot` with golden file creation and `CRAWLER_UPDATE` env var
- [ ] `Resize`
- [ ] `WaitExit` (process exit)
- [ ] `Scrollback`
- [ ] More key constants (function keys, Alt combos)

### Phase 3: Polish

- [ ] `Cursor` matcher (cursor position via `tmux display-message`)
- [ ] `WaitForScreen` (return the matching screen)
- [ ] Diagnostic output: on failure, dump last N screen captures
- [ ] Parallel test documentation and testing
- [ ] CI setup (GitHub Actions with tmux installed)
- [ ] `tmux` version detection and minimum version check
- [ ] Comprehensive `go doc` documentation
- [ ] Example tests in `example_test.go` (shown by `go doc`)
- [ ] README with usage guide

---

## Testing the library itself

The library needs a test TUI binary. Use a raw ANSI fixture program to keep
the test harness dependency-free and deterministic:

```
internal/testbin/
├── main.go          # Raw ANSI + stdin loop fixture program
└── testbin_test.go  # Build in TestMain, run integration tests against it
```

Tests for the library verify:
- `Open` starts a session and `t.Cleanup` tears it down.
- `Type`/`Press` sends the correct keystrokes (verified via screen capture).
- `WaitFor` succeeds when content appears and fails with a timeout when it doesn't.
- `MatchSnapshot` creates golden files and detects mismatches.
- `Resize` changes the terminal dimensions.
- Parallel subtests don't interfere with each other.

---

## Dependencies

**Runtime**: `tmux` 3.0+ must be installed on the system (not a Go dependency).
The tmux binary is located by checking, in order: `WithTmuxPath` option,
`CRAWLER_TMUX` environment variable, then `$PATH` lookup.

Dependency policy:

- If no explicit tmux path is configured and tmux is not in `$PATH`, call
  `t.Skip("crawler: open: tmux not found")`.
- If `WithTmuxPath` or `CRAWLER_TMUX` is set but invalid or not executable,
  call `t.Fatal` (explicit configuration error).
- If tmux is found via `$PATH` but version is below 3.0, call `t.Skip` with
  the detected version and required minimum (environment limitation).
- If tmux is found via `WithTmuxPath` or `CRAWLER_TMUX` but version is below
  3.0, call `t.Fatal` (explicit configuration error: unsupported tmux).

**Go module dependencies**: Ideally zero. The standard library provides
everything needed:
- `os/exec` — run tmux commands
- `strings`, `regexp` — screen content matching
- `testing` — test integration
- `time` — polling and timeouts
- `crypto/rand` or `math/rand` — unique socket path suffixes
- `path/filepath`, `os` — golden file management, `CRAWLER_UPDATE` env var

No third-party dependencies means no version conflicts for users.

---

## Prior art and differentiation

| Tool | Language | Mechanism | Scope |
|------|----------|-----------|-------|
| `teatest` | Go | In-process tea.Model | Bubble Tea only |
| `tcell.SimulationScreen` | Go | In-process fake screen | tcell only |
| `tui-test` (Microsoft) | TypeScript | xterm.js emulator | Any binary, but JS |
| `VHS` (Charm) | Go | Tape DSL → recording | Demo/docs, not testing |
| `go-expect` / `goexpect` | Go | PTY + expect patterns | Line-oriented, not TUI |
| **crawler** | **Go** | **tmux** | **Any binary, native Go tests** |

crawler's niche: **the only Go-native, framework-agnostic, TUI-aware testing
library**. PTY-based expect libraries work for line-oriented CLI programs but
can't reliably capture full-screen TUI state. Framework-specific tools only
work with one framework. crawler works with anything that runs in a terminal.

---

## Decisions (resolved from earlier open questions)

1. **Color/style assertions**: Deferred. Plain text via `capture-pane -p` is
   the starting point. Style matching via `capture-pane -e` can be added as
   an opt-in feature later if demand arises.

2. **Mouse support**: Deferred. Not included in any initial phase. When added,
   a `term.Click(row, col)` API would be the natural surface.

3. **Multi-pane testing**: Out of scope. Single pane per test is the right
   constraint. Multi-pane adds substantial complexity with minimal initial
   value.

4. **tmux minimum version**: 3.0+ (released November 2019). Covers all needed
   features. Checked at runtime in `Open` via `tmux -V`.

5. **Module path**: `github.com/cboone/crawler`. The name is fine — the
   package doc and README provide context. A vanity import path can be added
   later if desired.
