# Recipes and testing patterns

A cookbook of common TUI testing scenarios with complete examples. Each recipe
is self-contained.

For API details, see the [README](../README.md) and
the [package documentation on pkg.go.dev](https://pkg.go.dev/github.com/cboone/crawler)
or run `go doc github.com/cboone/crawler`.

## Basic interaction: Type / Press / WaitFor

The fundamental lifecycle is: wait for the screen to be ready, send input, wait
for the result.

```go
func TestBasicInteraction(t *testing.T) {
    term := crawler.Open(t, "./my-app")
    term.WaitFor(crawler.Text("Enter name:"))

    term.Type("Alice")
    term.Press(crawler.Enter)

    term.WaitFor(crawler.Text("Hello, Alice!"))
}
```

Always `WaitFor` before and after input. Never assume the screen is ready
immediately after `Open` or after sending keys.

## Form navigation

Tab between fields, type values, and submit:

```go
func TestFormSubmit(t *testing.T) {
    term := crawler.Open(t, "./my-form-app")
    term.WaitFor(crawler.Text("Name:"))

    term.Type("Alice")
    term.Press(crawler.Tab)

    term.WaitFor(crawler.Text("Email:"))
    term.Type("alice@example.com")
    term.Press(crawler.Tab)

    term.WaitFor(crawler.Text("Submit"))
    term.Press(crawler.Enter)

    term.WaitFor(crawler.Text("Saved successfully"))
}
```

## Menu / list selection

Navigate with arrow keys, select with Enter:

```go
func TestMenuSelection(t *testing.T) {
    term := crawler.Open(t, "./my-menu-app")
    term.WaitFor(crawler.Text("> Option 1"))

    term.Press(crawler.Down)
    term.WaitFor(crawler.Text("> Option 2"))

    term.Press(crawler.Down)
    term.WaitFor(crawler.Text("> Option 3"))

    term.Press(crawler.Enter)
    term.WaitFor(crawler.Text("Selected: Option 3"))
}
```

## Graceful shutdown with Ctrl+C

Send `Ctrl('c')` and verify the process exits cleanly:

```go
func TestGracefulShutdown(t *testing.T) {
    term := crawler.Open(t, "./my-server")
    term.WaitFor(crawler.Text("Server running"))

    term.Press(crawler.Ctrl('c'))

    code := term.WaitExit()
    if code != 0 {
        t.Fatalf("expected clean exit, got code %d", code)
    }
}
```

## Process exit

`WaitExit` waits for the process to terminate and returns its exit code. Use it
for tests where the binary is expected to finish:

```go
func TestExitZero(t *testing.T) {
    term := crawler.Open(t, "./my-cli", crawler.WithArgs("--version"))
    term.WaitFor(crawler.Text("v1.0.0"))

    code := term.WaitExit()
    if code != 0 {
        t.Fatalf("expected exit 0, got %d", code)
    }
}

func TestExitNonZero(t *testing.T) {
    term := crawler.Open(t, "./my-cli", crawler.WithArgs("--bad-flag"))
    term.WaitFor(crawler.Text("unknown flag"))

    code := term.WaitExit()
    if code == 0 {
        t.Fatal("expected non-zero exit code")
    }
}
```

## Terminal resize

`Resize` changes the terminal dimensions and sends SIGWINCH to the process:

```go
func TestResize(t *testing.T) {
    term := crawler.Open(t, "./my-app", crawler.WithSize(80, 24))
    term.WaitFor(crawler.Text("Dashboard"))

    term.Resize(120, 40)

    // Wait for the app to re-render at the new size.
    term.WaitFor(crawler.Text("Dashboard"))
}
```

After calling `Resize`, always `WaitFor` something to give the program time to
handle SIGWINCH and re-render.

## Scrollback capture

`Scrollback()` captures the full scrollback buffer, including lines that have
scrolled off the visible screen:

```go
func TestScrollback(t *testing.T) {
    term := crawler.Open(t, "./my-logger",
        crawler.WithSize(80, 10),
        crawler.WithHistoryLimit(50000),
    )
    term.WaitFor(crawler.Text("Log output complete"))

    scrollback := term.Scrollback()

    // Check for content that scrolled off screen.
    if !scrollback.Contains("First log entry") {
        t.Error("expected scrollback to contain first log entry")
    }

    // Use len(Lines()) for the total number of captured lines.
    lines := scrollback.Lines()
    t.Logf("captured %d scrollback lines", len(lines))
}
```

`WithHistoryLimit` controls how many scrollback lines tmux retains (default:
10000).

## Table-driven TUI tests

Use `t.Run` and `t.Parallel()` for table-driven tests. Each subtest gets its
own isolated tmux session:

```go
func TestNavigation(t *testing.T) {
    tests := []struct {
        name string
        keys []crawler.Key
        want string
    }{
        {"move down once", []crawler.Key{crawler.Down}, "> Item 2"},
        {"move down twice", []crawler.Key{crawler.Down, crawler.Down}, "> Item 3"},
        {"move down then up", []crawler.Key{crawler.Down, crawler.Up}, "> Item 1"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            term := crawler.Open(t, "./my-list-app")
            term.WaitFor(crawler.Text("> Item 1"))

            for _, key := range tt.keys {
                term.Press(key)
            }
            term.WaitFor(crawler.Text(tt.want))
        })
    }
}
```

## Parallel test safety

Every call to `Open` creates a fully isolated tmux server with its own socket
path. There is no shared state between tests. This means:

- `t.Parallel()` works without any extra setup.
- Subtests can run concurrently without cross-contamination.
- No mutexes, no coordination, no port allocation.

```go
func TestParallel(t *testing.T) {
    for i := 0; i < 10; i++ {
        i := i
        t.Run(fmt.Sprintf("instance-%d", i), func(t *testing.T) {
            t.Parallel()
            term := crawler.Open(t, "./my-app")
            term.WaitFor(crawler.Text("Ready"))

            msg := fmt.Sprintf("parallel-%d", i)
            term.Type(msg)
            term.Press(crawler.Enter)
            term.WaitFor(crawler.Text(msg))
        })
    }
}
```

## Environment variables

Pass environment variables with `WithEnv`:

```go
func TestNoColor(t *testing.T) {
    term := crawler.Open(t, "./my-app",
        crawler.WithEnv("NO_COLOR=1"),
    )
    term.WaitFor(crawler.Text("Ready"))
}

func TestCustomConfig(t *testing.T) {
    term := crawler.Open(t, "./my-app",
        crawler.WithEnv("APP_CONFIG=/tmp/test-config.json", "DEBUG=1"),
    )
    term.WaitFor(crawler.Text("Config loaded"))
}
```

Each entry should be in `KEY=VALUE` format. The environment is set by wrapping
the binary with `/usr/bin/env` internally.

## Working directory

`WithDir` sets the working directory for the binary:

```go
func TestWorkingDir(t *testing.T) {
    dir := t.TempDir()
    // ... set up files in dir ...

    term := crawler.Open(t, "./my-app",
        crawler.WithDir(dir),
    )
    term.WaitFor(crawler.Text("Files loaded"))
}
```

## WaitForScreen for follow-up assertions

`WaitForScreen` returns the `*Screen` that matched, so you can do additional
assertions on the same captured state:

```go
func TestWaitForScreen(t *testing.T) {
    term := crawler.Open(t, "./my-app")
    screen := term.WaitForScreen(crawler.Text("Results"))

    // The screen is guaranteed to contain "Results" at this point.
    // Do additional checks on the same capture.
    if !screen.Contains("Total: 42") {
        t.Errorf("expected total, got:\n%s", screen.String())
    }

    lines := screen.Lines()
    if len(lines) < 5 {
        t.Errorf("expected at least 5 lines, got %d", len(lines))
    }

    // You can also snapshot this exact screen.
    screen.MatchSnapshot(t, "results-page")
}
```

This avoids race conditions where `Screen()` might capture a different state
than what `WaitFor` saw.

## SendKeys as an escape hatch

`SendKeys` sends raw tmux key sequences. Use it when `Type` and `Press` don't
cover your needs:

```go
func TestSendKeys(t *testing.T) {
    term := crawler.Open(t, "./my-app")
    term.WaitFor(crawler.Text("Ready"))

    // Send raw tmux key names.
    term.SendKeys("h", "e", "l", "l", "o")
    term.WaitFor(crawler.Text("hello"))
}
```

`Type` sends text literally (via `send-keys -l`), `Press` sends named keys
(like `Enter`, `Up`), and `SendKeys` sends raw sequences with no
transformation. Prefer `Type` and `Press` unless you need a key sequence that
they don't support.

## See also

- [Getting started](GETTING-STARTED.md) -- first-test tutorial
- [Matchers in depth](MATCHERS.md) -- all matchers and custom matchers
- [Snapshot testing](SNAPSHOTS.md) -- golden-file testing
- [Troubleshooting](TROUBLESHOOTING.md) -- debugging and CI setup
