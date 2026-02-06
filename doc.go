// Package crawler is a Go testing library for black-box testing of terminal
// user interfaces. It is framework-agnostic: it tests any TUI binary
// (bubbletea, tview, tcell, Python curses, Rust ratatui, raw ANSI programs)
// by running it inside a tmux session, sending keystrokes, capturing screen
// output, and asserting against it.
//
// # Quick start
//
// A minimal test:
//
//	func TestMyApp(t *testing.T) {
//	    term := crawler.Open(t, "./my-app")
//	    term.WaitFor(crawler.Text("Welcome"))
//	    term.Type("hello")
//	    term.Press(crawler.Enter)
//	    term.WaitFor(crawler.Text("hello"))
//	}
//
// # Key concepts
//
//   - [Open] starts a binary in a new, isolated tmux session. Cleanup is
//     automatic via t.Cleanup.
//   - [Terminal.WaitFor] polls the screen until a [Matcher] succeeds or a
//     timeout expires, providing reliable waits without time.Sleep.
//   - [Terminal.Screen] captures the current visible content as a [Screen].
//   - [Terminal.Type] and [Terminal.Press] send input to the running program.
//   - [Terminal.MatchSnapshot] compares the screen against a golden file.
//
// # Requirements
//
// tmux 3.0+ must be installed and available in $PATH (or configured via
// [WithTmuxPath] or the CRAWLER_TMUX environment variable).
// Only Unix-like systems (Linux, macOS) are supported.
package crawler
