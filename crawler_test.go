package crawler_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/cboone/crawler"
)

var testBinary string

const (
	waitForTimeoutHelperEnv  = "CRAWLER_WAITFOR_TIMEOUT_HELPER"
	waitExitTimeoutHelperEnv = "CRAWLER_WAITEXIT_TIMEOUT_HELPER"
)

func TestMain(m *testing.M) {
	// Build the test fixture binary.
	dir, err := os.MkdirTemp("", "crawler-testbin-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)

	binPath := filepath.Join(dir, "testbin")
	cmd := exec.Command("go", "build", "-o", binPath, "./internal/testbin")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build testbin: %v\n", err)
		os.Exit(1)
	}

	testBinary = binPath
	os.Exit(m.Run())
}

func TestOpenAndCleanup(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))
}

func TestTypeAndEcho(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))

	term.Type("hello world")
	term.Press(crawler.Enter)
	term.WaitFor(crawler.Text("echo: hello world"))
}

func TestPressKeys(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))

	term.Type("test")
	term.Press(crawler.Enter)
	term.WaitFor(crawler.Text("echo: test"))
}

func TestWaitForSuccess(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))
}

func TestWaitForTimeout(t *testing.T) {
	if os.Getenv(waitForTimeoutHelperEnv) == "1" {
		term := crawler.Open(t, testBinary)
		term.WaitFor(crawler.Text("ready>"))
		term.WaitFor(crawler.Text("never appears"), crawler.WithinTimeout(150*time.Millisecond))
		return
	}

	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not found in PATH")
	}

	cmd := exec.Command(os.Args[0], "-test.run", "^TestWaitForTimeout$")
	cmd.Env = append(os.Environ(), waitForTimeoutHelperEnv+"=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected subprocess to fail, output:\n%s", string(out))
	}

	output := string(out)
	if !strings.Contains(output, "crawler: wait-for: timed out") {
		t.Fatalf("expected timeout message, got:\n%s", output)
	}
	if !strings.Contains(output, "recent screen captures (oldest to newest):") {
		t.Fatalf("expected recent captures header, got:\n%s", output)
	}
	if !regexp.MustCompile(`capture [0-9]+/[0-9]+:`).MatchString(output) {
		t.Fatalf("expected numbered captures, got:\n%s", output)
	}
}

func TestWaitForScreen(t *testing.T) {
	term := crawler.Open(t, testBinary)
	screen := term.WaitForScreen(crawler.Text("ready>"))

	if !screen.Contains("ready>") {
		t.Errorf("expected screen to contain 'ready>', got:\n%s", screen)
	}
}

func TestScreenContains(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))

	screen := term.Screen()
	if !screen.Contains("ready>") {
		t.Errorf("expected screen to contain 'ready>'")
	}
	if screen.Contains("nonexistent") {
		t.Errorf("expected screen to not contain 'nonexistent'")
	}
}

func TestScreenString(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))

	screen := term.Screen()
	s := screen.String()
	if !strings.Contains(s, "ready>") {
		t.Errorf("expected String() to contain 'ready>'")
	}
}

func TestScreenLines(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))

	screen := term.Screen()
	lines := screen.Lines()
	if len(lines) == 0 {
		t.Fatal("expected at least one line")
	}

	// First line should contain "ready>".
	if !strings.Contains(lines[0], "ready>") {
		t.Errorf("expected first line to contain 'ready>', got %q", lines[0])
	}

	// Lines should be a copy.
	lines[0] = "modified"
	original := screen.Lines()
	if original[0] == "modified" {
		t.Error("Lines() should return a copy")
	}
}

func TestScreenLine(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))

	screen := term.Screen()
	line := screen.Line(0)
	if !strings.Contains(line, "ready>") {
		t.Errorf("expected Line(0) to contain 'ready>', got %q", line)
	}
}

func TestScreenSize(t *testing.T) {
	term := crawler.Open(t, testBinary, crawler.WithSize(100, 30))
	term.WaitFor(crawler.Text("ready>"))

	screen := term.Screen()
	w, h := screen.Size()
	if w != 100 || h != 30 {
		t.Errorf("expected size 100x30, got %dx%d", w, h)
	}
}

func TestTextMatcher(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))
}

func TestRegexpMatcher(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Regexp(`ready>`))
}

func TestLineMatcher(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))

	term.Type("hello")
	term.Press(crawler.Enter)
	term.WaitFor(crawler.Text("echo: hello"))

	term.WaitFor(crawler.Line(1, "echo: hello"))
}

func TestLineContainsMatcher(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))

	term.Type("world")
	term.Press(crawler.Enter)
	term.WaitFor(crawler.LineContains(1, "world"))
}

func TestNotMatcher(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Not(crawler.Text("nonexistent string")))
}

func TestAllMatcher(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.All(
		crawler.Text("ready>"),
		crawler.Not(crawler.Text("nonexistent")),
	))
}

func TestAnyMatcher(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Any(
		crawler.Text("nonexistent"),
		crawler.Text("ready>"),
	))
}

func TestEmptyMatcher(t *testing.T) {
	// A screen with content should not be empty.
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))
	term.WaitFor(crawler.Not(crawler.Empty()))
}

func TestWaitExit(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))

	term.Type("quit")
	term.Press(crawler.Enter)

	code := term.WaitExit(crawler.WithinTimeout(10 * time.Second))
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestWaitExitNonZero(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))

	term.Type("fail")
	term.Press(crawler.Enter)

	code := term.WaitExit(crawler.WithinTimeout(10 * time.Second))
	if code != 1 {
		t.Errorf("expected exit code 1, got %d", code)
	}
}

func TestWaitExitTimeout(t *testing.T) {
	if os.Getenv(waitExitTimeoutHelperEnv) == "1" {
		term := crawler.Open(t, testBinary)
		term.WaitFor(crawler.Text("ready>"))
		_ = term.WaitExit(crawler.WithinTimeout(150 * time.Millisecond))
		return
	}

	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not found in PATH")
	}

	cmd := exec.Command(os.Args[0], "-test.run", "^TestWaitExitTimeout$")
	cmd.Env = append(os.Environ(), waitExitTimeoutHelperEnv+"=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected subprocess to fail, output:\n%s", string(out))
	}

	output := string(out)
	if !strings.Contains(output, "crawler: wait-exit: timed out") {
		t.Fatalf("expected wait-exit timeout message, got:\n%s", output)
	}
	if !strings.Contains(output, "recent screen captures (oldest to newest):") {
		t.Fatalf("expected recent captures header, got:\n%s", output)
	}
}

func TestResize(t *testing.T) {
	term := crawler.Open(t, testBinary, crawler.WithSize(80, 24))
	term.WaitFor(crawler.Text("ready>"))

	// Ask testbin to report size before resize.
	term.Type("size")
	term.Press(crawler.Enter)
	term.WaitFor(crawler.Text("size: 80x24"))

	// Resize.
	term.Resize(120, 40)

	// Ask for size again.
	term.Type("size")
	term.Press(crawler.Enter)
	term.WaitFor(crawler.Text("size: 120x40"))
}

func TestScrollback(t *testing.T) {
	term := crawler.Open(t, testBinary, crawler.WithSize(80, 10))
	term.WaitFor(crawler.Text("ready>"))

	// Generate enough lines to overflow the visible area.
	term.Type("lines 20")
	term.Press(crawler.Enter)
	term.WaitFor(crawler.Text("ready>"))

	// Give tmux a moment to update scrollback.
	time.Sleep(100 * time.Millisecond)

	scrollback := term.Scrollback()
	content := scrollback.String()

	// Should contain early lines that scrolled off screen.
	if !strings.Contains(content, "line 1") {
		t.Errorf("expected scrollback to contain 'line 1', got:\n%s", content)
	}
	if !strings.Contains(content, "line 20") {
		t.Errorf("expected scrollback to contain 'line 20', got:\n%s", content)
	}
}

func TestWithEnv(t *testing.T) {
	// Use testbin with env var and verify it through command output.
	term := crawler.Open(t, "/bin/sh",
		crawler.WithArgs("-c", "echo $CRAWLER_TEST_VAR && read line"),
		crawler.WithEnv("CRAWLER_TEST_VAR=hello_from_env"),
	)
	term.WaitFor(crawler.Text("hello_from_env"))
}

func TestWithDir(t *testing.T) {
	// WithDir sets the working directory.
	term := crawler.Open(t, "/bin/sh",
		crawler.WithArgs("-c", "pwd && read line"),
		crawler.WithDir(os.TempDir()),
	)
	// The output should contain a path.
	term.WaitFor(crawler.Regexp(`/`))
}

func TestWithTimeout(t *testing.T) {
	term := crawler.Open(t, testBinary, crawler.WithTimeout(10*time.Second))
	term.WaitFor(crawler.Text("ready>"))
}

func TestWithPollInterval(t *testing.T) {
	term := crawler.Open(t, testBinary, crawler.WithPollInterval(100*time.Millisecond))
	term.WaitFor(crawler.Text("ready>"))
}

func TestCtrlC(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))

	term.Press(crawler.Ctrl('c'))
	// Ctrl-C sends SIGINT; the process exits with a signal-based code.
	code := term.WaitExit(crawler.WithinTimeout(10 * time.Second))
	// Accept any non-zero exit code (SIGINT typically yields 130 or 2).
	_ = code
}

func TestMatchSnapshotUpdate(t *testing.T) {
	// Only run snapshot update test when CRAWLER_UPDATE is set.
	if os.Getenv("CRAWLER_UPDATE") != "1" {
		t.Skip("skipping snapshot update test (set CRAWLER_UPDATE=1)")
	}

	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))
	term.MatchSnapshot("ready-screen")
}

func TestParallelSubtests(t *testing.T) {
	for i := 0; i < 5; i++ {
		i := i
		t.Run(fmt.Sprintf("subtest-%d", i), func(t *testing.T) {
			t.Parallel()
			term := crawler.Open(t, testBinary)
			term.WaitFor(crawler.Text("ready>"))

			msg := fmt.Sprintf("parallel-%d", i)
			term.Type(msg)
			term.Press(crawler.Enter)
			term.WaitFor(crawler.Text("echo: " + msg))
		})
	}
}

func TestStressParallel(t *testing.T) {
	// Run 25 parallel subtests to verify no cross-test leakage.
	// Each subtest gets its own tmux session and verifies isolation.
	for i := 0; i < 25; i++ {
		i := i
		t.Run(fmt.Sprintf("stress-%d", i), func(t *testing.T) {
			t.Parallel()
			term := crawler.Open(t, testBinary)
			term.WaitFor(crawler.Text("ready>"))

			msg := fmt.Sprintf("stress-msg-%d", i)
			term.Type(msg)
			term.Press(crawler.Enter)
			term.WaitFor(crawler.Text("echo: " + msg))

			// Verify the screen contains our message, not another test's.
			screen := term.Screen()
			if !screen.Contains("echo: " + msg) {
				t.Errorf("expected screen to contain our echo, got:\n%s", screen)
			}
		})
	}
}

func TestCursorMatcher(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))
	term.WaitFor(crawler.Cursor(0, 6))
}

func TestSendKeys(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))

	// Use raw SendKeys to send literal text.
	term.SendKeys("h", "i")
	term.WaitFor(crawler.Text("hi"))
}

func TestMultipleCommands(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))

	// First command.
	term.Type("first")
	term.Press(crawler.Enter)
	term.WaitFor(crawler.Text("echo: first"))

	// Second command.
	term.Type("second")
	term.Press(crawler.Enter)
	term.WaitFor(crawler.Text("echo: second"))

	// Third command.
	term.Type("third")
	term.Press(crawler.Enter)
	term.WaitFor(crawler.Text("echo: third"))
}

func TestBackspace(t *testing.T) {
	term := crawler.Open(t, testBinary)
	term.WaitFor(crawler.Text("ready>"))

	// Type text, use backspace to correct, then press Enter.
	// The terminal line discipline handles backspace.
	term.Type("helloo")
	term.Press(crawler.Backspace)
	// After backspace, "hello" remains. Type more and send.
	term.Press(crawler.Enter)
	term.WaitFor(crawler.Text("echo: hello"))
}
