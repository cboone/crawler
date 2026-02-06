# crawler

Playwright for TUIs via tmux.

A Go testing library for black-box testing of terminal user interfaces.
Tests run real binaries inside tmux sessions, send keystrokes, capture screen
output, and assert against it -- all through the standard `testing.TB` interface.

## Quick start

```go
import "github.com/cboone/crawler"

func TestMyApp(t *testing.T) {
    term := crawler.Open(t, "./my-app")
    term.WaitFor(crawler.Text("Welcome"))
    term.Type("hello")
    term.Press(crawler.Enter)
    term.WaitFor(crawler.Text("hello"))
}
```

No `defer`, no `Close()`. Cleanup is automatic via `t.Cleanup`.

## Install

```sh
go get github.com/cboone/crawler
```

Requires tmux 3.0+ installed on the system. No other dependencies.

## Features

**Framework-agnostic** -- tests any TUI binary: bubbletea, tview, tcell,
Python curses, Rust ratatui, raw ANSI programs, anything that runs in a
terminal.

**Go-native API** -- first-class integration with `testing.TB`, subtests,
table-driven tests, `t.Helper()`, `t.Cleanup()`. No DSLs.

**Reliable waits** -- deterministic polling with timeouts instead of
`time.Sleep`. Like Playwright's auto-waiting locators.

**Snapshot testing** -- golden-file screen captures with `CRAWLER_UPDATE=1`.

**Zero dependencies** -- standard library only. No version conflicts for users.

## API overview

### Opening a session

```go
term := crawler.Open(t, "./my-app",
    crawler.WithArgs("--verbose"),
    crawler.WithSize(120, 40),
    crawler.WithEnv("NO_COLOR=1"),
    crawler.WithDir("/tmp/workdir"),
    crawler.WithTimeout(10 * time.Second),
)
```

### Sending input

```go
term.Type("hello world")           // literal text
term.Press(crawler.Enter)           // special keys
term.Press(crawler.Ctrl('c'))       // Ctrl combinations
term.Press(crawler.Alt('x'))        // Alt combinations
term.Press(crawler.Tab, crawler.Tab, crawler.Enter)  // multiple keys
term.SendKeys("raw", "tmux", "keys")  // escape hatch
```

### Capturing the screen

```go
screen := term.Screen()
screen.String()           // full content as string
screen.Lines()            // []string, one per row
screen.Line(0)            // single row (0-indexed)
screen.Contains("hello")  // substring check
screen.Size()             // (width, height)
```

### Waiting for content

```go
term.WaitFor(crawler.Text("Loading complete"))
term.WaitFor(crawler.Regexp(`\d+ items`))
term.WaitFor(crawler.LineContains(0, "My App v1.0"))
term.WaitFor(crawler.Not(crawler.Text("Loading...")))
term.WaitFor(crawler.All(crawler.Text("Done"), crawler.Not(crawler.Text("Error"))))

// Capture the matching screen
screen := term.WaitForScreen(crawler.Text("Results"))

// Override timeout for a single call
term.WaitFor(crawler.Text("Done"), crawler.WithinTimeout(30*time.Second))

// Override poll interval for a single call
term.WaitFor(crawler.Text("Done"), crawler.WithWaitPollInterval(100*time.Millisecond))
```

On timeout, `WaitFor` calls `t.Fatal` with a diagnostic message showing what
was expected and the most recent screen captures:

```
terminal_test.go:42: crawler: wait-for: timed out after 5s
    waiting for: screen to contain "Loading complete"
    recent screen captures (oldest to newest):
    capture 1/3:
    ┌────────────────────────────────────────────────────────────────────────────────┐
    │ My Application v1.0                                                            │
    │                                                                                │
    │ Loading...                                                                     │
    └────────────────────────────────────────────────────────────────────────────────┘
    capture 2/3:
    ┌────────────────────────────────────────────────────────────────────────────────┐
    │ My Application v1.0                                                            │
    │                                                                                │
    │ Loading...                                                                     │
    └────────────────────────────────────────────────────────────────────────────────┘
    capture 3/3:
    ┌────────────────────────────────────────────────────────────────────────────────┐
    │ My Application v1.0                                                            │
    │                                                                                │
    │ Loading...                                                                     │
    └────────────────────────────────────────────────────────────────────────────────┘
```

### Built-in matchers

| Matcher | Description |
|---------|-------------|
| `Text(s)` | Screen contains substring |
| `Regexp(pattern)` | Screen matches regex |
| `Line(n, s)` | Row n equals s (trailing spaces trimmed) |
| `LineContains(n, s)` | Row n contains substring |
| `Not(m)` | Inverts a matcher |
| `All(m...)` | All matchers must match |
| `Any(m...)` | At least one matcher must match |
| `Empty()` | Screen has no visible content |
| `Cursor(row, col)` | Cursor is at position |

### Snapshot testing

```go
term.WaitFor(crawler.Text("Welcome"))
term.MatchSnapshot("welcome-screen")
```

Golden files are stored in `testdata/<test-name>-<hash>/<name>.txt`.
Update them with:

```sh
CRAWLER_UPDATE=1 go test ./...
```

### Other operations

```go
// Resize the terminal (sends SIGWINCH)
term.Resize(120, 40)

// Wait for the process to exit
code := term.WaitExit()

// Capture full scrollback history
scrollback := term.Scrollback()
```

## Subtests and parallel tests

Each call to `Open` starts a dedicated tmux server with its own socket path and creates a new session within it.
Subtests and `t.Parallel()` work naturally:

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
            t.Parallel()
            term := crawler.Open(t, "./my-list-app")
            term.WaitFor(crawler.Text("> Item 1"))
            term.Press(tt.key)
            term.WaitFor(crawler.Text(tt.want))
        })
    }
}
```

## Documentation

- [Getting started](docs/GETTING-STARTED.md) -- first-test tutorial
- [Matchers in depth](docs/MATCHERS.md) -- all built-in matchers, composition, and custom matchers
- [Snapshot testing](docs/SNAPSHOTS.md) -- golden-file testing guide
- [Recipes and patterns](docs/PATTERNS.md) -- common testing scenarios with complete examples
- [Troubleshooting](docs/TROUBLESHOOTING.md) -- debugging failures and CI setup
- [Architecture](docs/ARCHITECTURE.md) -- how crawler works under the hood

## Requirements

- **Go** 1.24+
- **tmux** 3.0+ (checked at runtime; tests skip if tmux is not found)
- **OS**: Linux, macOS, or any Unix-like system where tmux runs

The tmux binary is located by checking, in order:
1. `WithTmuxPath` option
2. `CRAWLER_TMUX` environment variable
3. `$PATH` lookup

## How it works

Each test gets its own tmux server via a unique socket path under `os.TempDir()`.
All operations (`capture-pane`, `send-keys`, `resize-window`) go through the
`tmux` CLI. No cgo, no terminfo parsing, no terminal emulator reimplementation.

```
Go test process
+-------------------------------------------------+
|  func TestFoo(t *testing.T) {                   |
|      term := crawler.Open(t, ...)               |---- tmux new-session -d ...
|      term.WaitFor(crawler.Text("hello"))        |---- tmux capture-pane -p
|      term.Type("world")                         |---- tmux send-keys -l ...
|  }                                              |
+-------------------------------------------------+
                  |
                  v
tmux server (per-test isolated socket)
+----------------------------------+
|  session: default                |
|  +----------------------------+  |
|  |  $ ./my-tui-binary --flag  |  |
|  |  +----------------------+  |  |
|  |  |  TUI rendering here |  |  |
|  |  +----------------------+  |  |
|  +----------------------------+  |
+----------------------------------+
```
