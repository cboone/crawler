package crawler

import (
	"fmt"
	"regexp"
	"strings"
)

// A Matcher reports whether a Screen satisfies a condition.
// The string return is a human-readable description for error messages.
type Matcher func(s *Screen) (ok bool, description string)

// Text matches if the screen contains the given substring anywhere.
func Text(s string) Matcher {
	return func(scr *Screen) (bool, string) {
		return scr.Contains(s), fmt.Sprintf("screen to contain %q", s)
	}
}

// Regexp matches if the screen content matches the regular expression.
// The pattern is compiled once; an invalid pattern causes a panic.
func Regexp(pattern string) Matcher {
	re := regexp.MustCompile(pattern)
	return func(scr *Screen) (bool, string) {
		return re.MatchString(scr.String()), fmt.Sprintf("screen to match regexp %q", pattern)
	}
}

// Line matches if the given line (0-indexed) equals s after trimming
// trailing spaces from the screen line.
func Line(n int, s string) Matcher {
	return func(scr *Screen) (bool, string) {
		desc := fmt.Sprintf("line %d to equal %q", n, s)
		lines := scr.Lines()
		if n < 0 || n >= len(lines) {
			return false, desc
		}
		return strings.TrimRight(lines[n], " ") == s, desc
	}
}

// LineContains matches if the given line (0-indexed) contains the substring.
func LineContains(n int, substr string) Matcher {
	return func(scr *Screen) (bool, string) {
		desc := fmt.Sprintf("line %d to contain %q", n, substr)
		lines := scr.Lines()
		if n < 0 || n >= len(lines) {
			return false, desc
		}
		return strings.Contains(lines[n], substr), desc
	}
}

// Not inverts a matcher.
func Not(m Matcher) Matcher {
	return func(scr *Screen) (bool, string) {
		ok, desc := m(scr)
		return !ok, "NOT(" + desc + ")"
	}
}

// All matches when every provided matcher matches.
func All(matchers ...Matcher) Matcher {
	return func(scr *Screen) (bool, string) {
		descs := make([]string, 0, len(matchers))
		for _, m := range matchers {
			ok, desc := m(scr)
			descs = append(descs, desc)
			if !ok {
				return false, "all of: " + strings.Join(descs, ", ")
			}
		}
		return true, "all of: " + strings.Join(descs, ", ")
	}
}

// Any matches when at least one provided matcher matches.
func Any(matchers ...Matcher) Matcher {
	return func(scr *Screen) (bool, string) {
		descs := make([]string, 0, len(matchers))
		for _, m := range matchers {
			ok, desc := m(scr)
			descs = append(descs, desc)
			if ok {
				return true, "any of: " + strings.Join(descs, ", ")
			}
		}
		return false, "any of: " + strings.Join(descs, ", ")
	}
}

// Empty matches when the screen has no visible content.
func Empty() Matcher {
	return func(scr *Screen) (bool, string) {
		return strings.TrimSpace(scr.String()) == "", "screen to be empty"
	}
}

// Cursor matches if the cursor is at the given position.
// Uses tmux display-message to query cursor position.
// Note: row and col are 0-indexed. This matcher takes (row, col)
// to follow the usual row-then-column convention.
func Cursor(row, col int) Matcher {
	return func(scr *Screen) (bool, string) {
		desc := fmt.Sprintf("cursor at row=%d, col=%d", row, col)
		if scr.cursorRow == row && scr.cursorCol == col {
			return true, desc
		}
		return false, desc + fmt.Sprintf(" (actual: row=%d, col=%d)", scr.cursorRow, scr.cursorCol)
	}
}
