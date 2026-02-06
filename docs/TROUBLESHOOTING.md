# Troubleshooting

Debugging test failures, fixing common problems, and setting up CI.

For API overview and usage examples, see the [README](../README.md). For
detailed function signatures, see `go doc github.com/cboone/crawler`.

## tmux not found

If tmux is not installed, tests **skip** automatically (they don't fail):

```
--- SKIP: TestMyApp (0.00s)
    crawler: open: tmux not found
```

Install tmux:

```sh
# Ubuntu / Debian
sudo apt-get update && sudo apt-get install -y tmux

# macOS
brew install tmux
```

## tmux version too old

crawler requires tmux 3.0+. Check your version:

```sh
tmux -V
```

If the system tmux is auto-detected via PATH and the version is too old, the
test **skips**:

```
--- SKIP: TestMyApp (0.00s)
    crawler: open: tmux version 2.9 is below minimum 3.0
```

If the tmux path is explicitly configured via `WithTmuxPath` or the
`CRAWLER_TMUX` environment variable, the test **fails** instead of skipping:

```
--- FAIL: TestMyApp (0.00s)
    crawler: open: tmux version 2.9 is below minimum 3.0
```

The distinction: auto-detected tmux is treated as optional (skip), but
explicitly configured tmux is treated as a requirement (fail).

## Configuring the tmux path

The tmux binary is resolved in this order:

1. `WithTmuxPath("/path/to/tmux")` -- highest priority
2. `CRAWLER_TMUX` environment variable
3. PATH lookup (standard `exec.LookPath`)

This is useful when you have multiple tmux versions installed or need to test
against a specific build:

```go
term := crawler.Open(t, "./my-app",
    crawler.WithTmuxPath("/usr/local/bin/tmux-3.4"),
)
```

```sh
CRAWLER_TMUX=/opt/tmux/bin/tmux go test ./...
```

## WaitFor timeout failures

A timeout looks like this:

```
app_test.go:15: crawler: wait-for: timed out after 5s
    waiting for: screen to contain "Welcome"
    recent screen captures (oldest to newest):
    capture 1/3:
    ┌────────────────────────────────────────────────────────────────────────────────┐
    │$                                                                              │
    │                                                                               │
    └────────────────────────────────────────────────────────────────────────────────┘
```

### Reading the output

- **waiting for**: the matcher description. This tells you what condition was
  not met.
- **recent screen captures**: the last 3 screen captures before the timeout,
  shown oldest to newest. This shows what the terminal actually displayed.

### Common causes

- **Binary not producing expected output**: the program might be crashing,
  blocking on something, or writing to stderr instead of stdout.
- **Wrong matcher**: the text you are looking for doesn't match what the
  program actually renders (typo, different capitalization, extra whitespace).
- **Timing**: the program needs longer than the default 5s timeout.

### Strategies

1. **Increase the timeout** for a specific call:

   ```go
   term.WaitFor(crawler.Text("Done"), crawler.WithinTimeout(30*time.Second))
   ```

2. **Add intermediate WaitFor steps** to narrow down where the failure
   happens:

   ```go
   term.WaitFor(crawler.Text("Loading..."))    // does this pass?
   term.WaitFor(crawler.Text("Processing...")) // what about this?
   term.WaitFor(crawler.Text("Done"))          // fails here?
   ```

3. **Check the screen manually** to see what the program is actually showing:

   ```go
   screen := term.Screen()
   t.Logf("current screen:\n%s", screen.String())
   ```

4. **Use Regexp** for flexible matching when exact text varies:

   ```go
   term.WaitFor(crawler.Regexp(`(?i)welcome`))
   ```

## Process exited unexpectedly

This error means the TUI process terminated before the matcher succeeded:

```
crawler: wait-for: process exited unexpectedly (status 1)
    waiting for: screen to contain "Welcome"
    recent screen captures (oldest to newest):
    ...
```

The process crashed or exited before rendering the expected content. Check:

- Does the binary run correctly when launched manually?
- Are required environment variables set? Use `WithEnv`.
- Is the working directory correct? Use `WithDir`.
- Does the binary need arguments? Use `WithArgs`.

## Flaky tests

### Common causes

- **Reading Screen() without WaitFor**: `Screen()` captures the terminal at
  one instant. If you call it immediately after sending keys, the program may
  not have rendered yet. Always use `WaitFor` or `WaitForScreen` instead.

  ```go
  // BAD: race condition
  term.Type("hello")
  term.Press(crawler.Enter)
  screen := term.Screen()
  if !screen.Contains("echo: hello") { // might fail intermittently
      t.Fatal("missing echo")
  }

  // GOOD: deterministic
  term.Type("hello")
  term.Press(crawler.Enter)
  term.WaitFor(crawler.Text("echo: hello"))
  ```

- **SIGWINCH timing after Resize**: after calling `Resize`, the program needs
  time to receive SIGWINCH and re-render. Always `WaitFor` the expected
  post-resize content.

### Mitigations

- Always use `WaitFor` / `WaitForScreen` instead of `Screen()` + assert.
- If you need to assert on a specific captured screen, use `WaitForScreen` to
  get the matching screen, then assert on that.

## Socket path length

Unix domain sockets have a path length limit (104 bytes on macOS, 108 on
Linux). crawler handles this by:

- Sanitizing the test name: only `[A-Za-z0-9.-]` are kept, everything else
  becomes `_`.
- Truncating the sanitized name to 60 characters.
- Placing the socket in `os.TempDir()`.

If you see socket-related errors, check whether `os.TempDir()` itself has a
long path. On most systems this is `/tmp` and won't be a problem.

## CI with GitHub Actions

Here is a complete workflow based on the project's own CI configuration:

```yaml
name: CI

on:
  push:
    branches:
      - main
      - "feature/**"
  pull_request:

jobs:
  test:
    name: Test (${{ matrix.os }})
    runs-on: ${{ matrix.os }}
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, macos-latest]

    steps:
      - name: Check out repository
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Install tmux on Linux
        if: runner.os == 'Linux'
        run: |
          sudo apt-get update
          sudo apt-get install -y tmux

      - name: Install tmux on macOS
        if: runner.os == 'macOS'
        run: |
          if ! command -v tmux >/dev/null 2>&1; then
            brew install tmux
          fi

      - name: Run tests
        run: go test ./...
```

Key points:

- tmux is installed as a separate step before running tests.
- macOS runners sometimes have tmux pre-installed, so the step checks first.
- Do **not** set `CRAWLER_UPDATE=1` in CI.

## CI with other providers

The general requirements are:

1. Install tmux 3.0+ on the CI runner.
2. Run `go test ./...`.

tmux is available in most Linux package managers (`apt`, `yum`, `dnf`, `apk`)
and on macOS via Homebrew. If tmux is not available, tests skip automatically,
so a missing tmux won't break your build -- but it won't test your TUI either.

## Debugging tips

### Verbose test output

```sh
go test -run TestMyApp -v
```

### Run a single test

```sh
go test -run ^TestSpecificCase$ -v
```

### Test with a specific tmux binary

```sh
CRAWLER_TMUX=/path/to/tmux go test -run TestMyApp -v
```

### Inspect screen content during development

Add temporary logging to see what the screen contains:

```go
screen := term.Screen()
t.Logf("screen content:\n%s", screen.String())
t.Logf("screen lines: %d", len(screen.Lines()))
w, h := screen.Size()
t.Logf("screen size: %dx%d", w, h)
```

## See also

- [Getting started](GETTING-STARTED.md) -- first-test tutorial
- [Matchers in depth](MATCHERS.md) -- matchers and custom matchers
- [Recipes and patterns](PATTERNS.md) -- common testing scenarios
- [Architecture](ARCHITECTURE.md) -- how crawler works internally
