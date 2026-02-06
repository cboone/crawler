# Architecture

How crawler works internally. This guide is for contributors and users who want
to understand what happens behind the API.

For API overview and usage examples, see the [README](../README.md). For
detailed function signatures, see `go doc github.com/cboone/crawler`.

## Why tmux

Testing TUIs requires a real terminal environment: the program under test reads
from a PTY, writes escape sequences, and responds to signals like SIGWINCH.
There are several approaches to providing this:

- **PTY + VT parser**: allocate a pseudo-terminal and parse the VT100/xterm
  output stream yourself. This requires reimplementing a terminal emulator
  (cursor movement, scrolling, alternate screen, etc.) and is a large surface
  area to get right.
- **tcell SimScreen**: works only with programs built on tcell. Not
  framework-agnostic.
- **Embedded terminal emulator**: ship a Go terminal emulator library. Adds a
  significant dependency and still may not match real terminal behavior.
- **tmux**: already a complete, battle-tested terminal multiplexer with a CLI
  for programmatic control. Available on all Unix-like systems.

crawler uses tmux because it provides everything needed -- PTY management,
screen capture, key injection, resize handling -- through a stable CLI. No
terminal emulation code to maintain, no framework lock-in, and users already
have tmux or can install it easily.

The tradeoff: tmux is a runtime dependency, and tests skip if it's not
available.

## Session isolation

Each call to `Open` creates a dedicated tmux server by using a unique socket
path. The socket path is generated under `os.TempDir()`:

```
/tmp/crawler-TestMyApp-a1b2c3d4.sock
```

The format is `crawler-<sanitized-test-name>-<random-suffix>.sock`. Because
each test gets its own tmux server (not just its own session within a shared
server), there is complete isolation:

- No shared state between tests.
- `t.Parallel()` works without coordination.
- Cleanup kills the entire server, not just a session.

The server is killed during `t.Cleanup`, along with the temporary config file.

## The adapter layer

`tmux.go` is the bridge between the public API (`crawler.go`) and the
low-level tmux command runner (`internal/tmuxcli`). All tmux-specific details
are contained in `tmux.go`:

- Socket path generation and sanitization
- Config file creation
- Session startup
- Pane content capture (`capture-pane -p`)
- Cursor position queries (`display-message`)
- Key sending (`send-keys`)
- Window resizing (`resize-window`)
- Pane state queries (alive/dead, exit status)

The public API in `crawler.go` calls functions in `tmux.go` and never
interacts with `tmuxcli` directly. This keeps the boundary clean: if the tmux
interaction needs to change, only `tmux.go` is affected.

## Config file approach

tmux is configured via a temporary config file passed with `-f`, rather than
`set-option` commands after session start. The config sets:

```
set-option -g history-limit 10000
set-option -g remain-on-exit on
set-option -g status off
```

- **remain-on-exit on**: keeps the pane open after the process exits, so
  crawler can still read the exit status and final screen content. Without
  this, a fast-exiting process would disappear before crawler can query it.
- **status off**: disables the tmux status bar so the terminal dimensions
  match the requested size exactly. Without this, the status bar would consume
  one row.
- **history-limit**: controls scrollback buffer size for `Scrollback()`.

The config file is used instead of `set-option` after session start because a
process that exits immediately (before `set-option` runs) would not have
`remain-on-exit` set, causing its exit status to be lost.

## Environment variable passthrough

When `WithEnv` is used, the binary is wrapped with `/usr/bin/env`:

```
/usr/bin/env KEY1=VAL1 KEY2=VAL2 /path/to/binary --flag
```

This sets environment variables before the binary executes, within the tmux
session. The env wrapper is transparent to the running program.

## Screen capture

### Visible content

`capture-pane -p` captures the visible pane content as plain text. Each line
corresponds to a terminal row. The output is parsed into a `Screen` struct
with normalized line endings.

### Cursor position

The cursor position is queried separately via:

```
display-message -p -t <pane> "#{cursor_x} #{cursor_y}"
```

This returns `x y` coordinates (note: tmux uses x for column, y for row).
crawler swaps these to `(row, col)` for the `Cursor` matcher's `(row, col)`
convention.

### Scrollback

`capture-pane -p -S - -E -` captures the full scrollback buffer from the
earliest line (`-S -`) to the latest (`-E -`). The resulting `Screen` has
height and line count reflecting the total captured lines, not the visible pane
size.

### Immutability

A `Screen` is immutable after creation. `Lines()` returns a copy of the
internal slice. `String()` returns the raw content. There are no mutating
methods.

## The polling model

`WaitFor` and `WaitForScreen` use a poll-sleep loop:

1. Check if the pane is dead (process exited). If so, fail immediately with
   exit status.
2. Capture the screen (`capture-pane -p` + cursor query).
3. Run the matcher against the captured screen.
4. If the matcher succeeds, return (for `WaitForScreen`, return the screen).
5. If the deadline has passed, call `t.Fatal` with diagnostics.
6. Sleep for the poll interval.
7. Go to step 1.

### Poll interval

- Default: 50ms
- Minimum floor: 10ms (values below this are clamped)
- Configurable per-terminal with `WithPollInterval`
- Configurable per-call with `WithWaitPollInterval`

### Failure diagnostics

On timeout, crawler keeps the last 3 screen captures and includes them in the
failure message. This shows how the screen evolved during the wait, making it
easier to diagnose what the program was doing.

`WaitExit` uses the same polling model but checks pane state (alive/dead)
instead of running a matcher.

## Socket path generation

Socket paths must stay within Unix domain socket limits (104 bytes on macOS,
108 on Linux). crawler handles this with:

1. **Sanitize** the test name: keep `[A-Za-z0-9.-]`, replace everything else
   with `_`.
2. **Truncate** to 60 characters.
3. **Append** a random suffix (4 random bytes, hex-encoded = 8 characters).
4. **Format**: `crawler-<sanitized>-<suffix>.sock`
5. Place in `os.TempDir()` (typically `/tmp`).

If the path already exists (collision), regenerate the random suffix. Up to 10
attempts are made before failing.

## Error philosophy

crawler uses `t.Fatal` for errors and `t.Skip` for missing prerequisites:

- **t.Fatal**: tmux command failures, timeout, unexpected process exit,
  explicitly-configured tmux that's too old.
- **t.Skip**: tmux not found in PATH, auto-detected tmux version too old.

No crawler method returns an `error`. This keeps test code clean -- users never
write `if err != nil` for crawler calls. Errors format as
`crawler: <operation>: <reason>`.

## Limitations

- **Plain text only**: crawler captures text content, not colors, styles, or
  other ANSI attributes. Tests cannot assert on foreground/background colors
  or bold/underline.
- **No mouse**: tmux `send-keys` does not support mouse events. Mouse-driven
  TUIs cannot be tested with crawler.
- **No multi-pane**: each test uses a single pane. Testing multi-pane layouts
  is not supported.
- **No Windows**: tmux does not run on Windows. Tests are not supported on Windows and may fail to build.

## See also

- [Getting started](GETTING-STARTED.md) -- first-test tutorial
- [Matchers in depth](MATCHERS.md) -- the matcher system
- [Troubleshooting](TROUBLESHOOTING.md) -- debugging and CI setup
