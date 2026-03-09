# Getting started

This guide walks you through writing and running your first strider test. For a
concise API overview, see the [README](../README.md). For detailed function
signatures, see the [package documentation on pkg.go.dev](https://pkg.go.dev/github.com/cboone/strider)
or run `go doc github.com/cboone/strider`.

## Prerequisites

- **Go 1.24+**
- **tmux 3.0+** -- check with `tmux -V`
- **Linux or macOS** (or any Unix where tmux runs)

Install tmux if you don't have it:

```sh
# Ubuntu / Debian
sudo apt-get install tmux

# macOS
brew install tmux
```

## Install strider

```sh
go get github.com/cboone/strider
```

There are no other dependencies. strider uses the Go standard library only.

## Write your first test

Create a file called `app_test.go` next to whatever binary you want to test.
The binary can be written in any language -- Go, Rust, Python, anything that
runs in a terminal.

For this example, assume you have a binary `./my-app` that prints `Hello!` on
startup and waits for input.

```go
package myapp_test

import (
    "testing"

    "github.com/cboone/strider"
)

func TestMyApp(t *testing.T) {
    // Open starts the binary in an isolated tmux session.
    // Cleanup is automatic -- no defer or Close() needed.
    term := strider.Open(t, "./my-app")

    // Wait for the greeting to appear.
    term.WaitFor(strider.Text("Hello!"))

    // Type some input and press Enter.
    term.Type("world")
    term.Press(strider.Enter)

    // Wait for the response.
    term.WaitFor(strider.Text("world"))
}
```

That's it. `Open` creates an isolated tmux server, starts your binary inside
it, and registers cleanup via `t.Cleanup`. No `defer`, no `Close()`.

## Run the test

```sh
go test -run TestMyApp -v
```

If tmux is not installed or is below version 3.0, the test will skip
automatically (not fail).

## Understanding failure output

When `WaitFor` times out, strider reports what it was waiting for and shows the
most recent screen captures in a box-bordered format:

```
app_test.go:15: strider: wait-for: timed out after 5s
    waiting for: screen to contain "Hello!"
    recent screen captures (oldest to newest):
    capture 1/3:
    ┌────────────────────────────────────────────────────────────────────────────────┐
    │$                                                                              │
    │                                                                               │
    │                                                                               │
    └────────────────────────────────────────────────────────────────────────────────┘
    capture 2/3:
    ┌────────────────────────────────────────────────────────────────────────────────┐
    │$                                                                              │
    │                                                                               │
    │                                                                               │
    └────────────────────────────────────────────────────────────────────────────────┘
```

Up to 3 recent captures are shown (oldest to newest), so you can see how the
screen evolved before the timeout.

If the process exits before the matcher succeeds, you get an immediate failure
with the exit status:

```
app_test.go:15: strider: wait-for: process exited unexpectedly (status 1)
    waiting for: screen to contain "Hello!"
    recent screen captures (oldest to newest):
    ...
```

## Configuring the session

`Open` accepts functional options to customize the session:

```go
term := strider.Open(t, "./my-app",
    strider.WithSize(120, 40),
    strider.WithTimeout(10 * time.Second),
    strider.WithPollInterval(100 * time.Millisecond),
    strider.WithEnv("NO_COLOR=1", "TERM=xterm"),
    strider.WithArgs("--verbose", "--port", "8080"),
    strider.WithDir("/tmp/workdir"),
    strider.WithHistoryLimit(50000),
    strider.WithTmuxPath("/usr/local/bin/tmux"),
)
```

### Defaults

| Option | Default | Description |
|--------|---------|-------------|
| `WithSize` | 80 x 24 | Terminal width and height in characters |
| `WithTimeout` | 5s | Default timeout for `WaitFor`, `WaitForScreen`, `WaitExit` |
| `WithPollInterval` | 50ms | How often the screen is polled during waits (10ms floor) |
| `WithEnv` | (none) | Environment variables in `KEY=VALUE` format |
| `WithArgs` | (none) | Arguments passed to the binary |
| `WithDir` | (none) | Working directory for the binary |
| `WithHistoryLimit` | 10000 | tmux scrollback history limit |
| `WithTmuxPath` | (none) | Explicit path to the tmux binary |

Individual `WaitFor` / `WaitForScreen` / `WaitExit` calls can override the
timeout and poll interval with per-call options:

```go
term.WaitFor(strider.Text("Done"), strider.WithinTimeout(30*time.Second))
term.WaitFor(strider.Text("Done"), strider.WithWaitPollInterval(200*time.Millisecond))
```

## Next steps

- [Matchers in depth](MATCHERS.md) -- all built-in matchers, composition, and
  writing custom matchers
- [Snapshot testing](SNAPSHOTS.md) -- golden-file testing for screen content
- [Recipes and patterns](PATTERNS.md) -- common testing scenarios with complete
  examples
- [Troubleshooting](TROUBLESHOOTING.md) -- debugging failures and CI setup
- [Architecture](ARCHITECTURE.md) -- how strider works under the hood
