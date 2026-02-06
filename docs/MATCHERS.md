# Matchers in depth

Matchers are the core assertion mechanism in crawler. Every call to `WaitFor`
or `WaitForScreen` takes a matcher that is polled against screen captures until
it succeeds or the timeout expires.

For the API overview table, see the [README](../README.md). For function
signatures, see `go doc github.com/cboone/crawler`.

## How matchers work

A `Matcher` is a function type:

```go
type Matcher func(s *Screen) (ok bool, description string)
```

- `ok` reports whether the screen satisfies the condition.
- `description` is a human-readable string used in failure messages. When
  `WaitFor` times out, it prints `waiting for: <description>` so you can see
  exactly what condition was not met.

Because `Matcher` is a public `func` type, you can write custom matchers by
writing a function with this signature -- no interfaces to implement.

## Content matchers

### Text

Matches if the screen contains the given substring anywhere.

```go
term.WaitFor(crawler.Text("Welcome"))
```

Description: `screen to contain "Welcome"`

### Regexp

Matches if the full screen content matches the regular expression. The pattern
is compiled once when `Regexp` is called. An invalid pattern causes a panic.

```go
term.WaitFor(crawler.Regexp(`\d+ items loaded`))
term.WaitFor(crawler.Regexp(`(?i)error`))
```

Description: `screen to match regexp "\\d+ items loaded"`

## Line matchers

### Line

Matches if the given line (0-indexed) exactly equals the string after trimming
trailing spaces from the screen line.

```go
term.WaitFor(crawler.Line(0, "My Application v1.0"))
```

Description: `line 0 to equal "My Application v1.0"`

Returns `false` (does not panic) if the line index is out of range.

### LineContains

Matches if the given line (0-indexed) contains the substring.

```go
term.WaitFor(crawler.LineContains(2, "Status: OK"))
```

Description: `line 2 to contain "Status: OK"`

Returns `false` (does not panic) if the line index is out of range.

**Note:** The `Line` and `LineContains` matchers safely return `false` for
out-of-range indices. This is different from `Screen.Line(n)`, which panics on
out-of-range access.

## Position matcher

### Cursor

Matches if the cursor is at the given row and column. Both are 0-indexed, and
the convention is `(row, col)`.

```go
term.WaitFor(crawler.Cursor(0, 6))
```

Description: `cursor at row=0, col=6`

On mismatch, the description includes the actual position:
`cursor at row=0, col=6 (actual: row=0, col=0)`

## State matcher

### Empty

Matches when the screen has no visible content (`strings.TrimSpace` returns an
empty string).

```go
term.WaitFor(crawler.Empty())
term.WaitFor(crawler.Not(crawler.Empty()))
```

Description: `screen to be empty`

## Composition

### Not

Inverts a matcher.

```go
term.WaitFor(crawler.Not(crawler.Text("Loading...")))
```

Description: `NOT(screen to contain "Loading...")`

### All

Matches when every provided matcher matches. Short-circuits on the first
failure.

```go
term.WaitFor(crawler.All(
    crawler.Text("Status: OK"),
    crawler.Not(crawler.Text("Error")),
    crawler.LineContains(0, "Dashboard"),
))
```

Description: `all of: screen to contain "Status: OK", NOT(screen to contain "Error"), line 0 to contain "Dashboard"`

### Any

Matches when at least one provided matcher matches. Short-circuits on the first
success.

```go
term.WaitFor(crawler.Any(
    crawler.Text("Success"),
    crawler.Text("Already exists"),
))
```

Description: `any of: screen to contain "Success", screen to contain "Already exists"`

## Writing custom matchers

Since `Matcher` is a `func` type, custom matchers are just functions. Here are
three practical examples.

### Region checker

Check whether a rectangular region of the screen contains specific text:

```go
func Region(startRow, startCol, endRow, endCol int, want string) crawler.Matcher {
    return func(s *crawler.Screen) (bool, string) {
        desc := fmt.Sprintf("region [%d:%d]-[%d:%d] to contain %q",
            startRow, startCol, endRow, endCol, want)
        lines := s.Lines()
        var region strings.Builder
        for r := startRow; r <= endRow && r < len(lines); r++ {
            line := lines[r]
            from := startCol
            to := endCol
            if from >= len(line) {
                continue
            }
            if to > len(line) {
                to = len(line)
            }
            region.WriteString(line[from:to])
            region.WriteByte('\n')
        }
        return strings.Contains(region.String(), want), desc
    }
}
```

### Occurrence counter

Assert that a substring appears at least N times:

```go
func AtLeast(n int, substr string) crawler.Matcher {
    return func(s *crawler.Screen) (bool, string) {
        count := strings.Count(s.String(), substr)
        desc := fmt.Sprintf("screen to contain %q at least %d times (found %d)",
            substr, n, count)
        return count >= n, desc
    }
}
```

### Multi-line table assertion

Verify that a table has a specific number of data rows (lines matching a
pattern):

```go
func TableRows(pattern string, minRows int) crawler.Matcher {
    re := regexp.MustCompile(pattern)
    return func(s *crawler.Screen) (bool, string) {
        count := 0
        for _, line := range s.Lines() {
            if re.MatchString(line) {
                count++
            }
        }
        desc := fmt.Sprintf("at least %d rows matching %q (found %d)",
            minRows, pattern, count)
        return count >= minRows, desc
    }
}
```

Usage:

```go
// Wait for a table with at least 5 rows matching "| <data> |"
term.WaitFor(TableRows(`\|.*\|`, 5))
```

## Descriptions and error readability

Good descriptions make failures easy to diagnose. When writing custom matchers:

- Describe what the matcher **expects**, not what it found.
- Include actual values in the description when the match fails (like `Cursor`
  does).
- Keep descriptions concise -- they appear inline in test output.

The description is the string that appears after `waiting for:` in timeout
messages, so write it as something that completes the sentence "timed out
waiting for \_\_\_".

## See also

- [Getting started](GETTING-STARTED.md) -- first-test tutorial
- [Recipes and patterns](PATTERNS.md) -- common scenarios using matchers
- [Snapshot testing](SNAPSHOTS.md) -- golden-file assertions
