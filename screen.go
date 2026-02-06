package crawler

import (
	"strings"
)

// Screen is an immutable capture of terminal content.
type Screen struct {
	lines     []string
	raw       string
	width     int
	height    int
	cursorRow int
	cursorCol int
}

// newScreen creates a Screen from raw capture-pane output.
// It normalizes line endings and trims the trailing newline emitted by
// capture-pane.
func newScreen(raw string, width, height int) *Screen {
	// Normalize line endings.
	raw = strings.ReplaceAll(raw, "\r\n", "\n")

	// Remove single trailing newline from capture-pane output.
	raw = strings.TrimSuffix(raw, "\n")

	lines := strings.Split(raw, "\n")

	return &Screen{
		lines:     lines,
		raw:       raw,
		width:     width,
		height:    height,
		cursorRow: -1,
		cursorCol: -1,
	}
}

// String returns the full screen content as a string.
func (s *Screen) String() string {
	return s.raw
}

// Lines returns a copy of the screen content as a slice of strings, one per row.
// The returned slice is a shallow copy; callers may modify it without affecting
// the Screen.
func (s *Screen) Lines() []string {
	cp := make([]string, len(s.lines))
	copy(cp, s.lines)
	return cp
}

// Line returns the content of a single row (0-indexed).
// Panics if n is out of range.
func (s *Screen) Line(n int) string {
	return s.lines[n]
}

// Contains reports whether the screen contains the substring.
func (s *Screen) Contains(substr string) bool {
	return strings.Contains(s.raw, substr)
}

// Size returns the width and height.
func (s *Screen) Size() (width, height int) {
	return s.width, s.height
}
