// Package crawler provides black-box testing for terminal user interfaces.
//
// crawler runs a real binary inside an isolated tmux server, sends keystrokes,
// captures screen output, and performs assertions through the standard
// [testing.TB] interface. It is framework-agnostic and works with any program
// that renders in a terminal.
//
// # Quick Start
//
//	func TestMyApp(t *testing.T) {
//		term := crawler.Open(t, "./my-app")
//		term.WaitFor(crawler.Text("Welcome"))
//		term.Type("hello")
//		term.Press(crawler.Enter)
//		term.WaitFor(crawler.Text("hello"))
//	}
//
// Cleanup is automatic through t.Cleanup; there is no Close method.
//
// # Session Lifecycle
//
// [Open] creates a dedicated tmux server for each test, using a unique socket
// path under os.TempDir. This gives subtests and parallel tests full isolation.
//
// Internally, crawler starts tmux with a temporary config file that enables:
//
//   - remain-on-exit on
//   - status off
//   - deterministic history-limit
//
// The tmux server is torn down with kill-server during cleanup.
//
// # Waiting and Matchers
//
// [Terminal.WaitFor] and [Terminal.WaitForScreen] poll until a [Matcher]
// succeeds or a timeout expires. This is the core reliability mechanism and
// avoids ad hoc sleeps in tests.
//
// Wait behavior:
//
//   - Defaults: 5s timeout, 50ms poll interval
//   - Per-terminal overrides: [WithTimeout], [WithPollInterval]
//   - Per-call overrides: [WithinTimeout], [WithWaitPollInterval]
//   - Poll intervals under 10ms are clamped to 10ms
//   - Negative timeout or poll values fail the test immediately
//   - If the process exits early, waits fail immediately with diagnostics
//
// Built-in matchers include [Text], [Regexp], [Line], [LineContains], [Not],
// [All], [Any], [Empty], and [Cursor].
//
// # Screen Capture
//
// [Terminal.Screen] captures the visible pane. [Terminal.Scrollback] captures
// full scrollback history. A [Screen] is immutable and provides helpers such as
// [Screen.String], [Screen.Lines], [Screen.Line], [Screen.Contains], and
// [Screen.Size].
//
// # Snapshots
//
// [Terminal.MatchSnapshot] and [Screen.MatchSnapshot] compare screen content to
// golden files under testdata. Set CRAWLER_UPDATE=1 to create or update golden
// files.
//
// Snapshot content is normalized for stable diffs by trimming trailing spaces,
// trimming trailing blank lines, and writing a single trailing newline.
//
// # Diagnostics
//
// On wait failures, crawler reports:
//
//   - expected matcher description
//   - timeout or exit details
//   - multiple recent screen captures (oldest to newest)
//
// This keeps failures actionable without extra debug tooling.
//
// # Requirements
//
//   - Go 1.24+
//   - tmux 3.0+
//   - Linux or macOS
//
// tmux is resolved in this order:
//
//   - [WithTmuxPath]
//   - CRAWLER_TMUX
//   - PATH lookup for tmux
package crawler
