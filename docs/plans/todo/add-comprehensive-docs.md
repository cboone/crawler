# Add comprehensive documentation to docs/

## Context

The crawler library has good inline documentation (README, `doc.go`, `example_test.go`) but no standalone guides for users who need more depth than the README overview. The `docs/` directory is empty except for `plans/`. Users need tutorial-style content, detailed matcher/snapshot guides, practical recipes, and troubleshooting help that go beyond what a README or godoc should cover.

## Plan

Create 6 Markdown files in `docs/` and update the root `README.md` to link to them. Each guide expands on topics the README covers briefly, without duplicating it.

### Files to create

#### 1. `docs/GETTING-STARTED.md` — First-test tutorial

Walk a new user from zero to a working test. Not a reference (that's the README) but a narrative walkthrough.

- Prerequisites: Go 1.24+, tmux 3.0+ (`tmux -V`), Linux/macOS
- Installing: `go get github.com/cboone/crawler`
- Writing your first test: full step-by-step with a real binary
- Running the test and reading the output
- Understanding failure output: the box-bordered screen captures, "waiting for" descriptions, "recent screen captures (oldest to newest)" format
- Configuring the session: `WithSize`, `WithTimeout`, `WithPollInterval`, `WithEnv`, `WithArgs`, `WithDir`, with defaults table (80x24, 5s timeout, 50ms poll interval)
- Next steps: links to the other guides

#### 2. `docs/MATCHERS.md` — Matchers in depth

Cover the matcher system, all built-ins, composition, and custom matchers.

- How matchers work: the `Matcher` type signature `func(s *Screen) (ok bool, description string)`, what the description string is for (appears in `WaitFor` failure output)
- Content matchers: `Text` (substring), `Regexp` (compiled once, panics on invalid pattern)
- Line matchers: `Line` (exact after trailing-space trim, 0-indexed), `LineContains` (substring, 0-indexed). Note: `Line`/`LineContains` return false for out-of-range indices; `Screen.Line(n)` panics
- Position: `Cursor(row, col)` — 0-indexed, (row, col) convention, shows actual position on mismatch
- State: `Empty()` — checks `strings.TrimSpace` against empty string
- Composition: `Not`, `All`, `Any` — how descriptions compose (e.g., `"NOT(screen to contain \"X\")"`, `"all of: X, Y"`)
- Writing custom matchers: 3 practical examples (region checker, occurrence counter, multi-line table assertion). Since `Matcher` is a public `func` type, users just write a function
- Matcher descriptions and error readability

#### 3. `docs/SNAPSHOTS.md` — Snapshot testing guide

Deep dive into golden-file testing.

- Concept: what snapshot testing is, why it's useful for TUI output
- Taking a snapshot: `term.MatchSnapshot("name")` and `screen.MatchSnapshot(t, "name")`
- File paths: `testdata/<sanitized-test-name>-<hash>/<sanitized-name>.txt`
  - Hash: first 4 bytes of SHA-256 of `t.Name()`, hex-encoded (8 chars)
  - Name sanitization: `[A-Za-z0-9.-]` kept, everything else becomes `_`, truncated to 60 chars
- Content normalization: trailing spaces trimmed per line, trailing blank lines removed, single trailing newline added
- The update workflow: `CRAWLER_UPDATE=1 go test ./...` (truthy values are exact lowercase matches: `1`, `true`, `yes`), reviewing changes with `git diff testdata/`
- `Terminal.MatchSnapshot` vs `Screen.MatchSnapshot`: the former captures then snapshots, the latter snapshots an already-captured screen (e.g., from `WaitForScreen`)
- Mismatch output format: golden file path, golden content, actual content, rerun instructions
- Missing golden file output: path, actual screen, rerun instructions
- Organizing snapshots: naming conventions, checking into version control
- CI considerations: never run with `CRAWLER_UPDATE=1` in CI

#### 4. `docs/PATTERNS.md` — Recipes and testing patterns

Cookbook of common scenarios with complete examples.

- Basic interaction: Type/Press/WaitFor lifecycle
- Form navigation: Tab between fields, type, submit, verify
- Menu/list selection: Arrow keys, Enter, verify
- Graceful shutdown: `Ctrl('c')` + `WaitExit` to check exit code
- Process exit: `WaitExit` for clean exit vs non-zero exit codes
- Terminal resize: `Resize(w, h)` + wait for SIGWINCH response
- Scrollback capture: `Scrollback()` for content that scrolled off screen, `WithHistoryLimit`
- Table-driven TUI tests: `t.Run` loop with `t.Parallel()` and table structs
- Parallel test safety: each `Open` gets an isolated tmux server
- Environment variables: `WithEnv("NO_COLOR=1")` and similar
- Working directory: `WithDir` for apps that read relative paths
- `WaitForScreen` for follow-up assertions: capture the matching screen, then inspect it
- `SendKeys` as an escape hatch: when `Type`/`Press` aren't sufficient, send raw tmux key sequences

#### 5. `docs/TROUBLESHOOTING.md` — Debugging and CI setup

Help users diagnose problems and set up CI.

- tmux not found: what happens (`t.Skip`), how to install (Ubuntu: `apt-get install tmux`, macOS: `brew install tmux`)
- tmux version too old: minimum 3.0, `t.Skip` for auto-detected vs `t.Fatal` for explicitly configured (`WithTmuxPath` or `CRAWLER_TMUX`)
- Configuring the tmux path: resolution order (WithTmuxPath > CRAWLER_TMUX > PATH)
- WaitFor timeout failures: reading the failure output, common causes, strategies (increase timeout with `WithinTimeout`, add intermediate `WaitFor` steps, check screen content manually)
- Process exited unexpectedly: TUI crashed before matcher succeeded, how to debug
- Flaky tests: common causes (rendering race, SIGWINCH timing), mitigations (always use `WaitFor` instead of `Screen` + assert)
- Socket path length: Unix 104/108 char limit, crawler truncates sanitized name to 60 chars
- CI with GitHub Actions: complete workflow YAML based on the project's own `ci.yml`
- CI with other providers: general guidance (just need tmux 3.0+ available)
- Debugging tips: `go test -run TestName -v`, `CRAWLER_TMUX` for specific tmux builds

#### 6. `docs/ARCHITECTURE.md` — How it works

For contributors and users who want to understand internals.

- Why tmux over alternatives (PTY + VT parser, tcell SimScreen, etc.)
- Session isolation: one tmux server per test via unique socket path under `os.TempDir()`
- The adapter layer: `tmux.go` bridges the public API and `internal/tmuxcli`
- Config file approach: `remain-on-exit`, `status off`, `history-limit` set via `-f` config file before session start (not `set-option` after), because fast-exiting processes could die first
- Environment variable passthrough: wraps binary in `/usr/bin/env` to set env before execution
- Screen capture: `capture-pane -p` for visible content, cursor via `display-message`, `Screen` is immutable
- The polling model: poll + sleep + matcher, 10ms floor, 3-capture failure history for diagnostics
- Socket path generation: sanitized test name + random suffix, truncated for Unix limits, collision retry (up to 10 attempts)
- Error philosophy: `t.Fatal` on errors (no error returns), `t.Skip` for missing/old tmux
- Limitations: plain text only (no colors/styles), no mouse, no multi-pane, no Windows

### Cross-references

Each doc links to related guides where relevant and points back to the README for the API overview table and `go doc` for detailed signatures.

### Root README updates

Add a `Documentation` section to the root `README.md` with links to all 6 new guides in `docs/` so they are discoverable from the repository front page.

### Source files to reference during implementation

- `crawler.go` — Open, WaitFor, WaitExit, Resize, Scrollback, diagnostics formatting
- `options.go` — all Option/WaitOption constructors, default values
- `match.go` — Matcher type, all 9 built-in matchers
- `snapshot.go` — snapshot paths, normalization, update logic
- `tmux.go` — socket paths, sanitization, config, session lifecycle
- `keys.go` — Key type, constants, Ctrl/Alt
- `screen.go` — Screen type, immutability, Line panics
- `crawler_test.go` — real usage patterns to adapt as examples
- `.github/workflows/ci.yml` — CI setup to reference in troubleshooting

## Verification

- All 6 files render correctly as GitHub-flavored Markdown
- Root `README.md` includes working links to all 6 docs guides
- Code examples are syntactically valid Go
- Cross-links between docs resolve correctly
- Facts match the source code (defaults, error messages, path formats, behavior)
- No duplication of README content — docs expand and deepen, not repeat
