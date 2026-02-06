# Snapshot testing

Snapshot testing (also called golden-file testing) lets you assert that a
screen looks exactly like a saved reference. Instead of writing individual
assertions for each piece of text, you capture the entire screen and compare it
against a committed file.

For API overview and usage examples, see the [README](../README.md). For
detailed function signatures, see `go doc github.com/cboone/crawler`.

## When to use snapshots

Snapshots work well when:

- You care about the **exact layout** of a screen, not just individual strings.
- You want to detect **unintentional changes** to TUI output.
- You have a stable screen that doesn't change between runs (no timestamps,
  random IDs, or other dynamic content).

For screens with dynamic content, use [matchers](MATCHERS.md) instead.

## Taking a snapshot

There are two ways to snapshot a screen:

### Terminal.MatchSnapshot

Captures the current screen and compares it to the golden file in one step:

```go
term.WaitFor(crawler.Text("Dashboard"))
term.MatchSnapshot("dashboard")
```

### Screen.MatchSnapshot

Snapshots an already-captured screen. This is useful when you get a screen back
from `WaitForScreen` and want to snapshot it:

```go
screen := term.WaitForScreen(crawler.Text("Results"))
// Do some assertions on the screen...
screen.MatchSnapshot(t, "results-page")
```

The difference: `Terminal.MatchSnapshot` calls `Screen()` internally then
snapshots. `Screen.MatchSnapshot` snapshots a screen you already have -- for
instance one returned by `WaitForScreen`.

## File paths

Golden files are stored at:

```
testdata/<sanitized-test-name>-<hash>/<sanitized-name>.txt
```

- **sanitized-test-name**: the `t.Name()` value with unsafe characters
  replaced. Characters in `[A-Za-z0-9.-]` are kept; everything else becomes
  `_`. Truncated to 60 characters.
- **hash**: first 4 bytes of the SHA-256 of `t.Name()`, hex-encoded (8
  characters). This ensures uniqueness even if truncation makes two test names
  identical.
- **sanitized-name**: the snapshot name you pass to `MatchSnapshot`, sanitized
  using the same rules.

Example: for a test named `TestDashboard/admin_view` with snapshot name
`"main-screen"`, the path would be something like:

```
testdata/TestDashboard_admin_view-a1b2c3d4/main-screen.txt
```

## Content normalization

Before writing or comparing, crawler normalizes the screen content:

1. Trailing spaces are trimmed from each line.
2. Trailing blank lines are removed.
3. A single trailing newline is added.

This produces stable diffs that aren't affected by terminal padding.

## The update workflow

Golden files don't exist until you create them. On the first run,
`MatchSnapshot` fails with a message telling you to create the file:

```
crawler: snapshot: golden file not found: testdata/TestFoo-a1b2c3d4/my-screen.txt
Run with CRAWLER_UPDATE=1 to create it.

Actual screen:
<screen content>
```

To create or update golden files:

```sh
CRAWLER_UPDATE=1 go test ./...
```

The `CRAWLER_UPDATE` variable is recognized as truthy when its exact lowercase
value is `1`, `true`, or `yes`. Any other value (including empty) is treated as
false.

After updating, review the changes:

```sh
git diff testdata/
```

Then commit the golden files alongside your test code.

## Mismatch output

When the screen doesn't match the golden file:

```
crawler: snapshot: mismatch for "dashboard"
Golden file: testdata/TestDashboard-a1b2c3d4/dashboard.txt
Run with CRAWLER_UPDATE=1 to update.

--- golden ---
Dashboard v1.0
Items: 42
Status: OK

--- actual ---
Dashboard v1.0
Items: 43
Status: OK
```

## Organizing snapshots

### Naming conventions

Use descriptive, stable names:

```go
term.MatchSnapshot("empty-state")
term.MatchSnapshot("after-login")
term.MatchSnapshot("error-dialog")
```

Snapshot names are sanitized to be filesystem-safe, so you can use hyphens and
dots but special characters will become underscores.

### Version control

Golden files in `testdata/` should be committed to the repository. They are
part of your test suite. When reviewing pull requests, changes to golden files
show exactly what changed in the TUI output.

## CI considerations

Never run tests with `CRAWLER_UPDATE=1` in CI. If you do, golden files will be
silently created or overwritten and the test will always pass, defeating the
purpose.

A typical CI setup runs tests normally:

```sh
go test ./...
```

If a snapshot is out of date, the test fails and the developer updates locally:

```sh
CRAWLER_UPDATE=1 go test ./...
git diff testdata/
git add testdata/
git commit -m "update golden files"
```

## See also

- [Getting started](GETTING-STARTED.md) -- first-test tutorial
- [Matchers in depth](MATCHERS.md) -- assertion matchers for dynamic content
- [Recipes and patterns](PATTERNS.md) -- common testing patterns
- [Troubleshooting](TROUBLESHOOTING.md) -- debugging and CI setup
