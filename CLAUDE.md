# crawler

A Go testing library for black-box testing of TUI programs via tmux.

## Project overview

- **Module**: `github.com/cboone/crawler`
- **Package**: Single public package `crawler`
- **Go version**: 1.24+
- **Dependencies**: Zero third-party Go dependencies (stdlib only)
- **Runtime requirement**: tmux 3.0+ on Linux or macOS

## Architecture

Each test gets an isolated tmux server via a unique socket path. All terminal
interaction goes through the `tmux` CLI (`capture-pane`, `send-keys`,
`resize-window`, `list-panes`, `display-message`). A temporary config file
sets `remain-on-exit`, `history-limit`, and `status off` before the session
starts.

### File layout

```
crawler.go          Terminal type, Open(), core methods (Type, Press, WaitFor, etc.)
options.go          Option/WaitOption types and functional option constructors
screen.go           Screen type (immutable capture of terminal content)
keys.go             Key type, constants (Enter, Tab, arrows, F1-F12), Ctrl/Alt helpers
match.go            Matcher type and built-in matchers (Text, Regexp, Line, Not, All, etc.)
snapshot.go         MatchSnapshot, golden file management, CRAWLER_UPDATE support
tmux.go             tmux adapter layer: session lifecycle, version check, socket paths,
                    pane state queries, cursor position, sanitizeName
doc.go              Package-level godoc documentation

internal/
  tmuxcli/          Low-level tmux command runner (Runner, Error, Version, WaitForSession)
  testbin/          Minimal line-based TUI fixture used by integration tests

crawler_test.go     Integration tests (35 tests including 25-subtest parallel stress test)
testdata/           Golden files for snapshot tests (created by CRAWLER_UPDATE=1)
```

### Key design decisions

- `tmux.go` is the adapter between the public API and `internal/tmuxcli`. All
  tmux details are contained there.
- `remain-on-exit` is set via config file (`-f`) rather than `set-option` after
  session start, so fast-exiting processes still report exit codes.
- `status off` disables the tmux status bar so terminal dimensions match the
  requested size exactly.
- Screen captures include cursor position on a best-effort basis for the
  `Cursor` matcher. If `display-message` fails, cursor fields use sentinel
  values (-1) and the `Cursor` matcher reports "cursor position unavailable."
- Socket paths include a sanitized test name and random suffix, truncated to
  stay within Unix socket path limits.

## Development

### Running tests

```sh
go test ./...
```

Tests require tmux in `$PATH`. If tmux is not found, tests skip automatically.

### Updating snapshots

```sh
CRAWLER_UPDATE=1 go test ./...
```

### Key environment variables

- `CRAWLER_UPDATE` -- set to `1` to create/update golden files
- `CRAWLER_TMUX` -- override the tmux binary path

## Conventions

- All public methods that interact with tmux call `t.Fatal` on error; users
  never check `err` returns.
- Error messages follow the format: `crawler: <operation>: <reason>`.
- `WaitFor` and `WaitForScreen` fail immediately if the pane dies before the
  matcher succeeds.
- `WaitExit` is the expected API for tests that intentionally terminate the
  process.
- Matchers return `(ok bool, description string)` where description is
  human-readable for error messages.
