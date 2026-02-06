package crawler

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MatchSnapshot compares the current screen against a golden file
// stored in testdata/<sanitized-test-name>/<sanitized-name>.txt.
//
// Set CRAWLER_UPDATE=1 to create or update golden files.
func (term *Terminal) MatchSnapshot(name string) {
	term.t.Helper()
	scr := term.Screen()
	scr.MatchSnapshot(term.t, name)
}

// MatchSnapshot on Screen allows snapshotting a previously captured screen.
func (s *Screen) MatchSnapshot(t testing.TB, name string) {
	t.Helper()

	// Build snapshot path.
	dir := snapshotDir(t)
	sanitized := sanitizeName(name)
	path := filepath.Join(dir, sanitized+".txt")

	// Normalize screen content for stable diffs:
	// - Trim trailing spaces on each line
	// - Remove trailing blank lines
	// - End with a single newline
	content := normalizeForSnapshot(s.String())

	if shouldUpdate() {
		// Create/update golden file.
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("crawler: snapshot: failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("crawler: snapshot: failed to write golden file: %v", err)
		}
		return
	}

	// Read and compare.
	golden, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("crawler: snapshot: golden file not found: %s\nRun with CRAWLER_UPDATE=1 to create it.\n\nActual screen:\n%s", path, content)
		}
		t.Fatalf("crawler: snapshot: failed to read golden file: %v", err)
	}

	if string(golden) != content {
		t.Fatalf("crawler: snapshot: mismatch for %q\nGolden file: %s\nRun with CRAWLER_UPDATE=1 to update.\n\n--- golden ---\n%s\n--- actual ---\n%s",
			name, path, string(golden), content)
	}
}

// snapshotDir returns the directory for golden files for the current test.
// Uses testdata/<sanitized-test-name>-<hash>/ where hash ensures uniqueness.
func snapshotDir(t testing.TB) string {
	t.Helper()

	fullName := t.Name()
	sanitized := sanitizeName(fullName)

	// Short stable hash for uniqueness.
	h := sha256.Sum256([]byte(fullName))
	hash := hex.EncodeToString(h[:4])

	return filepath.Join("testdata", sanitized+"-"+hash)
}

// normalizeForSnapshot normalizes screen content for stable golden file diffs.
func normalizeForSnapshot(raw string) string {
	lines := strings.Split(raw, "\n")

	// Trim trailing spaces on each line.
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}

	// Remove trailing blank lines.
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// End with a single newline.
	return strings.Join(lines, "\n") + "\n"
}

// shouldUpdate returns true if CRAWLER_UPDATE is set to a truthy value.
func shouldUpdate() bool {
	v := os.Getenv("CRAWLER_UPDATE")
	return v == "1" || v == "true" || v == "yes"
}

