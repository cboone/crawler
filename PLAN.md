# TUI Testing Library Architecture Plan

> **tuikit** - A Go testing library that provides Playwright-like capabilities for terminal user interfaces via tmux.

## Executive Summary

This document outlines the architecture for a Go testing library that enables automated testing of TUI applications through screen capture, input simulation, content assertions, and comprehensive session management. The library uses tmux as the underlying mechanism to spawn, interact with, and capture terminal application state.

## Design Principles

1. **Fluent API First** - Chainable methods for readable test code
2. **Type Safety** - Strongly typed interfaces preventing runtime errors
3. **Test Integration** - Seamless integration with Go's `testing` package
4. **Isolation by Default** - Each test runs in an isolated tmux session
5. **Fail Fast with Context** - Rich error messages with terminal snapshots
6. **Async-Aware** - Built-in polling and waiting mechanisms
7. **Zero External Dependencies** - Only standard library plus tmux CLI

---

## Package Structure

```
tuikit/
├── tuikit.go              # Main entry point, Session creation
├── session.go             # Session management
├── window.go              # Window operations
├── pane.go                # Pane operations and content capture
├── expect.go              # Assertion builder (fluent API)
├── matcher.go             # Content matchers interface
├── input.go               # Input simulation (keys, text)
├── screen.go              # Screen buffer and content model
├── ansi/                  # ANSI parsing subpackage
│   ├── parser.go          # State machine parser
│   ├── style.go           # Style extraction
│   ├── strip.go           # ANSI stripping utilities
│   └── cell.go            # Cell representation with attributes
├── keys/                  # Key constants and helpers
│   ├── keys.go            # Key constants (Enter, Escape, Ctrl+C, etc.)
│   └── sequence.go        # Key sequence builder
├── matchers/              # Built-in matchers
│   ├── text.go            # Text matching (contains, regex, etc.)
│   ├── cursor.go          # Cursor position matchers
│   ├── style.go           # Style/color matchers
│   └── layout.go          # Screen region matchers
├── internal/
│   ├── tmux/              # Low-level tmux CLI wrapper
│   │   ├── client.go      # Tmux command execution
│   │   ├── capture.go     # Pane capture implementation
│   │   └── parse.go       # Output parsing utilities
│   ├── pool/              # Session pooling (optional)
│   │   └── pool.go        # Reusable session pool
│   └── debug/             # Debugging utilities
│       ├── recorder.go    # Session recording
│       └── snapshot.go    # Screenshot capture
├── testutil/              # Test integration helpers
│   ├── testing.go         # *testing.T integration
│   └── cleanup.go         # Automatic cleanup
└── examples/              # Example tests (documentation)
    ├── basic_test.go
    ├── multi_pane_test.go
    └── async_test.go
```

---

## Core Interfaces and Types

### 1. Session Management

```go
// tuikit.go

// Config holds configuration for creating new sessions
type Config struct {
    // TmuxBinary is the path to the tmux binary (default: "tmux")
    TmuxBinary string

    // SocketPath allows using a custom tmux socket for isolation
    // If empty, a unique socket is generated per session
    SocketPath string

    // DefaultTimeout is the default timeout for all wait operations
    DefaultTimeout time.Duration

    // DefaultPollInterval is the polling interval for wait operations
    DefaultPollInterval time.Duration

    // Size specifies the terminal dimensions
    Size TerminalSize

    // Environment variables to set in the session
    Env map[string]string

    // WorkingDir sets the starting directory
    WorkingDir string

    // Debug enables verbose logging and recording
    Debug bool

    // DebugDir is the directory to write debug artifacts
    DebugDir string
}

// TerminalSize represents terminal dimensions
type TerminalSize struct {
    Cols int // Width in columns (default: 80)
    Rows int // Height in rows (default: 24)
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config

// New creates a new isolated tmux session for testing
// The session is automatically cleaned up when the test ends
func New(t testing.TB, opts ...Option) *Session

// Option is a functional option for configuring sessions
type Option func(*Config)

// WithSize sets the terminal size
func WithSize(cols, rows int) Option

// WithTimeout sets the default timeout
func WithTimeout(d time.Duration) Option

// WithEnv sets environment variables
func WithEnv(env map[string]string) Option

// WithDebug enables debug mode
func WithDebug(dir string) Option
```

### 2. Session Type

```go
// session.go

// Session represents an isolated tmux session for testing
type Session struct {
    t          testing.TB
    config     Config
    id         string
    socketPath string
    client     *internal.TmuxClient
    windows    []*Window
    recorder   *debug.Recorder
    closed     bool
    mu         sync.Mutex
}

// Run starts a command in the session and returns when ready
// This is the primary way to launch a TUI application
func (s *Session) Run(command string, args ...string) *Pane

// RunWithOptions starts a command with additional options
func (s *Session) RunWithOptions(opts RunOptions) *Pane

// RunOptions configures how a command is started
type RunOptions struct {
    Command      string
    Args         []string
    Env          map[string]string
    WorkingDir   string
    WaitReady    bool   // Wait for the pane to be ready before returning
    ReadyPattern string // Regex pattern indicating the app is ready
}

// NewWindow creates a new window in the session
func (s *Session) NewWindow(name string) *Window

// Windows returns all windows in the session
func (s *Session) Windows() []*Window

// ActiveWindow returns the currently active window
func (s *Session) ActiveWindow() *Window

// SelectWindow switches to a window by index or name
func (s *Session) SelectWindow(identifier interface{}) *Window

// Kill terminates the session immediately
func (s *Session) Kill() error

// Close gracefully closes the session (called automatically via t.Cleanup)
func (s *Session) Close() error

// Snapshot captures the current state of all panes for debugging
func (s *Session) Snapshot() *Snapshot

// Record starts recording all session activity
func (s *Session) Record() *Recorder

// ID returns the unique session identifier
func (s *Session) ID() string
```

### 3. Window Type

```go
// window.go

// Window represents a tmux window containing one or more panes
type Window struct {
    session *Session
    id      string
    name    string
    index   int
}

// Pane returns the primary pane (pane 0) of this window
func (w *Window) Pane() *Pane

// Panes returns all panes in this window
func (w *Window) Panes() []*Pane

// PaneByIndex returns a specific pane by index
func (w *Window) PaneByIndex(index int) *Pane

// SplitHorizontal creates a new pane by splitting horizontally
func (w *Window) SplitHorizontal() *Pane

// SplitVertical creates a new pane by splitting vertically
func (w *Window) SplitVertical() *Pane

// SplitWithOptions creates a new pane with specific options
func (w *Window) SplitWithOptions(opts SplitOptions) *Pane

// SplitOptions configures pane splitting
type SplitOptions struct {
    Vertical   bool
    Percentage int    // Size as percentage of parent
    Size       int    // Size in rows/columns
    Command    string
    Args       []string
}

// Select makes this window the active window
func (w *Window) Select() *Window

// Rename changes the window name
func (w *Window) Rename(name string) *Window

// Close closes this window
func (w *Window) Close() error

// SetLayout applies a tmux layout preset
func (w *Window) SetLayout(layout Layout) *Window

// Layout represents a tmux layout preset
type Layout string

const (
    LayoutEvenHorizontal Layout = "even-horizontal"
    LayoutEvenVertical   Layout = "even-vertical"
    LayoutMainHorizontal Layout = "main-horizontal"
    LayoutMainVertical   Layout = "main-vertical"
    LayoutTiled          Layout = "tiled"
)
```

### 4. Pane Type (Core Interface)

```go
// pane.go

// Pane represents a tmux pane - the primary interface for TUI testing
type Pane struct {
    window  *Window
    id      string
    index   int
}

// --- Input Methods (Fluent API) ---

// Type sends text input to the pane
func (p *Pane) Type(text string) *Pane

// TypeSlowly sends text with a delay between characters
func (p *Pane) TypeSlowly(text string, delay time.Duration) *Pane

// Press sends a key press (use keys.* constants)
func (p *Pane) Press(key keys.Key) *Pane

// PressRepeat sends a key press multiple times
func (p *Pane) PressRepeat(key keys.Key, count int) *Pane

// SendKeys sends raw tmux key sequences
func (p *Pane) SendKeys(keys ...string) *Pane

// Ctrl sends a control key combination (e.g., Ctrl+C)
func (p *Pane) Ctrl(char rune) *Pane

// Alt sends an alt key combination
func (p *Pane) Alt(char rune) *Pane

// --- Screen Capture Methods ---

// Screen captures the current screen content
func (p *Pane) Screen() *Screen

// ScreenWithHistory captures content including scrollback history
func (p *Pane) ScreenWithHistory(lines int) *Screen

// Text returns the plain text content (ANSI stripped)
func (p *Pane) Text() string

// RawContent returns content with ANSI escape codes preserved
func (p *Pane) RawContent() string

// --- Assertion Methods (Fluent API) ---

// Expect returns an expectation builder for fluent assertions
func (p *Pane) Expect() *Expect

// --- Wait Methods ---

// Wait pauses for a duration
func (p *Pane) Wait(d time.Duration) *Pane

// WaitForText waits until text appears on screen
func (p *Pane) WaitForText(text string) *Pane

// WaitForPattern waits until a regex pattern matches
func (p *Pane) WaitForPattern(pattern string) *Pane

// WaitUntil waits until a custom condition is met
func (p *Pane) WaitUntil(condition func(*Screen) bool) *Pane

// WaitForStable waits until screen content stops changing
func (p *Pane) WaitForStable(duration time.Duration) *Pane

// --- Pane Control ---

// Resize changes the pane size
func (p *Pane) Resize(cols, rows int) *Pane

// Select makes this pane active
func (p *Pane) Select() *Pane

// Close closes this pane
func (p *Pane) Close() error

// ScrollUp scrolls the pane up
func (p *Pane) ScrollUp(lines int) *Pane

// ScrollDown scrolls the pane down
func (p *Pane) ScrollDown(lines int) *Pane

// EnterCopyMode enters tmux copy mode
func (p *Pane) EnterCopyMode() *Pane

// ExitCopyMode exits tmux copy mode
func (p *Pane) ExitCopyMode() *Pane

// --- Debugging ---

// Screenshot saves a screenshot to the debug directory
func (p *Pane) Screenshot(name string) *Pane

// Log logs a message with the current screen content
func (p *Pane) Log(message string) *Pane
```

### 5. Screen and Cell Types

```go
// screen.go

// Screen represents the captured terminal screen content
type Screen struct {
    cells    [][]Cell
    width    int
    height   int
    cursorX  int
    cursorY  int
    captured time.Time
}

// Cell represents a single terminal cell with content and style
type Cell struct {
    Rune  rune
    Style Style
    Width int // Character width (1 or 2 for wide chars)
}

// Style represents text styling attributes
type Style struct {
    FG            Color
    BG            Color
    Bold          bool
    Italic        bool
    Underline     bool
    Strikethrough bool
    Dim           bool
    Blink         bool
    Reverse       bool
}

// Color represents a terminal color
type Color interface {
    IsDefault() bool
    RGB() (r, g, b uint8, ok bool)
    Index() (index int, ok bool)
}

// Screen Methods

// Text returns the plain text content of the screen
func (s *Screen) Text() string

// Line returns the text content of a specific line (0-indexed)
func (s *Screen) Line(row int) string

// Lines returns all lines as a slice
func (s *Screen) Lines() []string

// Cell returns the cell at a specific position
func (s *Screen) Cell(col, row int) Cell

// Region returns a rectangular region of the screen
func (s *Screen) Region(x, y, width, height int) *Screen

// Row returns a horizontal slice
func (s *Screen) Row(row int) []Cell

// Column returns a vertical slice
func (s *Screen) Column(col int) []Cell

// CursorPosition returns the cursor position
func (s *Screen) CursorPosition() (col, row int)

// Size returns the screen dimensions
func (s *Screen) Size() (cols, rows int)

// Contains checks if text exists anywhere on screen
func (s *Screen) Contains(text string) bool

// ContainsPattern checks if a regex pattern matches
func (s *Screen) ContainsPattern(pattern string) bool

// Find returns positions where text is found
func (s *Screen) Find(text string) []Position

// FindPattern returns positions matching a regex pattern
func (s *Screen) FindPattern(pattern string) []Match

// Position represents a screen position
type Position struct {
    Col int
    Row int
}

// Match represents a regex match with position
type Match struct {
    Position
    Text   string
    Groups []string
}

// StyledText returns text with style markers
func (s *Screen) StyledText() string

// Diff returns differences from another screen
func (s *Screen) Diff(other *Screen) *ScreenDiff

// ScreenDiff represents changes between two screens
type ScreenDiff struct {
    Added   []Position
    Removed []Position
    Changed []Position
}
```

### 6. Expect (Assertion Builder)

```go
// expect.go

// Expect provides a fluent API for assertions
type Expect struct {
    pane    *Pane
    timeout time.Duration
    poll    time.Duration
    not     bool
}

// Timeout sets the maximum wait time for this expectation
func (e *Expect) Timeout(d time.Duration) *Expect

// Poll sets the polling interval for this expectation
func (e *Expect) Poll(d time.Duration) *Expect

// Not negates the following assertion
func (e *Expect) Not() *Expect

// --- Content Assertions ---

// ToContainText asserts the screen contains specific text
func (e *Expect) ToContainText(text string) *Expect

// ToContainPattern asserts the screen matches a regex pattern
func (e *Expect) ToContainPattern(pattern string) *Expect

// ToHaveText asserts the screen text equals exactly
func (e *Expect) ToHaveText(text string) *Expect

// ToMatchSnapshot compares against a saved snapshot
func (e *Expect) ToMatchSnapshot(name string) *Expect

// LineToContain asserts a specific line contains text
func (e *Expect) LineToContain(row int, text string) *Expect

// LineToEqual asserts a specific line equals text exactly
func (e *Expect) LineToEqual(row int, text string) *Expect

// LineToMatch asserts a specific line matches a pattern
func (e *Expect) LineToMatch(row int, pattern string) *Expect

// --- Cursor Assertions ---

// CursorAt asserts the cursor is at a specific position
func (e *Expect) CursorAt(col, row int) *Expect

// CursorInRegion asserts the cursor is within a region
func (e *Expect) CursorInRegion(x, y, width, height int) *Expect

// CursorOnLine asserts the cursor is on a specific line
func (e *Expect) CursorOnLine(row int) *Expect

// --- Style Assertions ---

// ToHaveStyledText asserts text exists with specific styling
func (e *Expect) ToHaveStyledText(text string, style StyleMatcher) *Expect

// RegionToHaveStyle asserts a region has specific styling
func (e *Expect) RegionToHaveStyle(x, y, w, h int, style StyleMatcher) *Expect

// --- Layout Assertions ---

// ToBeEmpty asserts the screen is empty
func (e *Expect) ToBeEmpty() *Expect

// ToBeNonEmpty asserts the screen has content
func (e *Expect) ToBeNonEmpty() *Expect

// RowCount asserts the number of non-empty rows
func (e *Expect) RowCount(count int) *Expect

// --- Custom Assertions ---

// ToPass asserts a custom matcher passes
func (e *Expect) ToPass(matcher Matcher) *Expect

// ToSatisfy asserts a custom condition
func (e *Expect) ToSatisfy(fn func(*Screen) bool, msg string) *Expect

// Matcher is the interface for custom matchers
type Matcher interface {
    Match(screen *Screen) MatchResult
    Description() string
}

// MatchResult contains the result of a matcher
type MatchResult struct {
    Passed   bool
    Message  string
    Actual   string
    Expected string
}
```

### 7. Matchers Package

```go
// matchers/text.go

// Text returns a matcher that checks for text presence
func Text(substring string) Matcher

// Pattern returns a matcher that checks for regex pattern
func Pattern(regex string) Matcher

// Exact returns a matcher for exact text match
func Exact(text string) Matcher

// All returns a matcher that requires all sub-matchers to pass
func All(matchers ...Matcher) Matcher

// Any returns a matcher that requires at least one sub-matcher to pass
func Any(matchers ...Matcher) Matcher

// Count returns a matcher that checks occurrence count
func Count(text string, count int) Matcher

// AtLine returns a matcher that checks a specific line
func AtLine(row int, matcher Matcher) Matcher

// InRegion returns a matcher scoped to a screen region
func InRegion(x, y, w, h int, matcher Matcher) Matcher
```

```go
// matchers/style.go

// StyleMatcher matches cell styles
type StyleMatcher interface {
    MatchStyle(style Style) bool
}

// Bold returns a style matcher for bold text
func Bold() StyleMatcher

// Italic returns a style matcher for italic text
func Italic() StyleMatcher

// FG returns a style matcher for foreground color
func FG(color Color) StyleMatcher

// BG returns a style matcher for background color
func BG(color Color) StyleMatcher

// Styled combines multiple style matchers
func Styled(matchers ...StyleMatcher) StyleMatcher
```

### 8. Keys Package

```go
// keys/keys.go

// Key represents a terminal key
type Key string

// Common keys
const (
    Enter     Key = "Enter"
    Tab       Key = "Tab"
    Escape    Key = "Escape"
    Backspace Key = "BSpace"
    Delete    Key = "DC"

    Up        Key = "Up"
    Down      Key = "Down"
    Left      Key = "Left"
    Right     Key = "Right"

    Home      Key = "Home"
    End       Key = "End"
    PageUp    Key = "PPage"
    PageDown  Key = "NPage"

    F1        Key = "F1"
    F2        Key = "F2"
    // ... F3-F12

    Space     Key = "Space"
)

// Ctrl returns a Ctrl+key combination
func Ctrl(char rune) Key

// Alt returns an Alt+key combination
func Alt(char rune) Key

// Shift returns a Shift+key combination
func Shift(key Key) Key

// Sequence builds a sequence of keys
func Sequence(keys ...Key) []Key
```

```go
// keys/sequence.go

// KeySequence is a builder for complex key sequences
type KeySequence struct {
    keys []string
}

// NewSequence creates a new key sequence builder
func NewSequence() *KeySequence

// Press adds a key press
func (s *KeySequence) Press(key Key) *KeySequence

// Type adds text input
func (s *KeySequence) Type(text string) *KeySequence

// Ctrl adds a Ctrl combination
func (s *KeySequence) Ctrl(char rune) *KeySequence

// Wait adds a delay
func (s *KeySequence) Wait(d time.Duration) *KeySequence

// Build returns the tmux key sequence
func (s *KeySequence) Build() []string
```

### 9. ANSI Parsing Package

```go
// ansi/parser.go

// Parser parses ANSI escape sequences from terminal output
type Parser struct {
    state   parserState
    handler Handler
    params  []int
    buffer  []byte
}

// NewParser creates a new ANSI parser
func NewParser() *Parser

// Parse processes bytes and returns parsed cells
func (p *Parser) Parse(data []byte) []Cell

// ParseString processes a string and returns parsed cells
func (p *Parser) ParseString(s string) []Cell

// Handler is called for each parsed element
type Handler interface {
    Print(r rune)
    Execute(code byte)
    SGR(params []int)
    CSI(cmd byte, params []int)
    OSC(data string)
    Error(err error)
}

// Strip removes all ANSI escape codes from text
func Strip(s string) string

// StripBytes removes ANSI codes from bytes
func StripBytes(b []byte) []byte

// ExtractStyle parses style from ANSI SGR sequence
func ExtractStyle(s string) (string, Style)
```

### 10. Internal Tmux Client

```go
// internal/tmux/client.go

// Client wraps tmux CLI interactions
type Client struct {
    binary     string
    socketPath string
    logger     Logger
}

// NewClient creates a new tmux client
func NewClient(binary, socketPath string) *Client

// Exec executes a tmux command
func (c *Client) Exec(args ...string) (string, error)

// ExecWithTimeout executes with a timeout
func (c *Client) ExecWithTimeout(timeout time.Duration, args ...string) (string, error)

// NewSession creates a new tmux session
func (c *Client) NewSession(name string, opts SessionOptions) error

// KillSession terminates a session
func (c *Client) KillSession(name string) error

// ListSessions returns all sessions
func (c *Client) ListSessions() ([]SessionInfo, error)

// CapturePane captures pane content
func (c *Client) CapturePane(target string, opts CaptureOptions) (string, error)

// CaptureOptions configures pane capture
type CaptureOptions struct {
    StartLine   int  // -S flag, negative for history
    EndLine     int  // -E flag
    EscapeCodes bool // -e flag, include escape codes
    Quiet       bool // -q flag
}

// SendKeys sends keys to a pane
func (c *Client) SendKeys(target string, keys ...string) error

// ResizePane resizes a pane
func (c *Client) ResizePane(target string, cols, rows int) error

// SplitWindow splits a window
func (c *Client) SplitWindow(target string, opts SplitOptions) (string, error)

// GetPaneInfo returns information about a pane
func (c *Client) GetPaneInfo(target string) (*PaneInfo, error)

// PaneInfo contains tmux pane information
type PaneInfo struct {
    ID      string
    Index   int
    Width   int
    Height  int
    CursorX int
    CursorY int
    Active  bool
    Title   string
    Path    string
    Command string
}
```

### 11. Debug and Recording

```go
// internal/debug/recorder.go

// Recorder captures session activity for debugging
type Recorder struct {
    session   string
    outputDir string
    events    []Event
    recording bool
    mu        sync.Mutex
}

// NewRecorder creates a new recorder
func NewRecorder(session, outputDir string) *Recorder

// Start begins recording
func (r *Recorder) Start() error

// Stop ends recording and saves artifacts
func (r *Recorder) Stop() error

// AddEvent records an event
func (r *Recorder) AddEvent(event Event)

// Event represents a recorded action
type Event struct {
    Time   time.Time
    Type   EventType
    Data   interface{}
    Screen *Screen
}

// EventType categorizes events
type EventType string

const (
    EventKeyPress   EventType = "keypress"
    EventTextInput  EventType = "text"
    EventScreenshot EventType = "screenshot"
    EventAssertion  EventType = "assertion"
)

// SaveScreenshot saves a PNG screenshot
func (r *Recorder) SaveScreenshot(name string, screen *Screen) error

// GenerateHTML creates an HTML replay file
func (r *Recorder) GenerateHTML() error

// GenerateAsciicast creates an asciicast v2 file
func (r *Recorder) GenerateAsciicast() error
```

---

## Error Handling Strategy

### Error Types

```go
// errors.go

// Error wraps errors with context
type Error struct {
    Op      string  // Operation that failed
    Err     error   // Underlying error
    Screen  *Screen // Screen state at time of error
    Details string  // Additional context
}

func (e *Error) Error() string
func (e *Error) Unwrap() error

// Specific error types
var (
    ErrTimeout        = errors.New("operation timed out")
    ErrSessionClosed  = errors.New("session is closed")
    ErrPaneNotFound   = errors.New("pane not found")
    ErrTmuxNotFound   = errors.New("tmux binary not found")
    ErrPatternInvalid = errors.New("invalid regex pattern")
)

// AssertionError is returned when an assertion fails
type AssertionError struct {
    Message  string
    Expected string
    Actual   string
    Screen   *Screen
    Position *Position // Where mismatch occurred
}

func (e *AssertionError) Error() string

// TimeoutError includes context about what was being waited for
type TimeoutError struct {
    Waiting string        // What we were waiting for
    Timeout time.Duration
    Screen  *Screen       // Last screen state
}

func (e *TimeoutError) Error() string
```

### Error Handling Approach

1. **Fatal by Default**: Assertions call `t.Fatal()` on failure for immediate feedback
2. **Rich Context**: All errors include screen state for debugging
3. **Wrapped Errors**: Use `errors.Is()` and `errors.As()` for checking
4. **Recovery Mode**: Optional non-fatal mode for exploratory testing

```go
// Optional soft assertion mode
func (e *Expect) Soft() *Expect // Don't fail test, collect errors

// Check collected errors at end
func (s *Session) Errors() []error
func (s *Session) HasErrors() bool
```

---

## Timeout and Retry Mechanisms

### Configuration

```go
// Defaults
const (
    DefaultTimeout      = 10 * time.Second
    DefaultPollInterval = 100 * time.Millisecond
    MinPollInterval     = 10 * time.Millisecond
)

// Per-session configuration
func WithTimeout(d time.Duration) Option
func WithPollInterval(d time.Duration) Option

// Per-operation override
func (p *Pane) WithTimeout(d time.Duration) *Pane
func (e *Expect) Timeout(d time.Duration) *Expect
```

### Retry Logic

```go
// internal/retry/retry.go

// Retry executes a function with retries
type Retry struct {
    Timeout  time.Duration
    Interval time.Duration
    OnRetry  func(attempt int, err error)
}

// Do retries until success or timeout
func (r *Retry) Do(ctx context.Context, fn func() (bool, error)) error

// Example internal usage
func (p *Pane) WaitForText(text string) *Pane {
    retry := &Retry{
        Timeout:  p.timeout,
        Interval: p.pollInterval,
    }

    err := retry.Do(p.ctx, func() (bool, error) {
        screen := p.Screen()
        return screen.Contains(text), nil
    })

    if err != nil {
        p.t.Fatalf("timeout waiting for text %q:\n%s", text, p.Text())
    }
    return p
}
```

### Exponential Backoff (Optional)

```go
func WithExponentialBackoff(initial, max time.Duration) Option
```

---

## Example Use Cases

### Basic TUI Testing

```go
func TestBasicTUI(t *testing.T) {
    // Create isolated session (auto-cleaned up)
    session := tuikit.New(t, tuikit.WithSize(80, 24))

    // Launch TUI application
    pane := session.Run("./my-tui-app")

    // Wait for app to be ready
    pane.WaitForText("Welcome to My App")

    // Interact with the app
    pane.Type("hello").Press(keys.Enter)

    // Assert expected state
    pane.Expect().
        ToContainText("You typed: hello").
        CursorOnLine(2)
}
```

### Multi-Pane Testing

```go
func TestSplitPane(t *testing.T) {
    session := tuikit.New(t)

    // Create split layout
    window := session.ActiveWindow()
    leftPane := window.Pane()
    rightPane := window.SplitHorizontal()

    // Run different commands
    leftPane.Run("htop")
    rightPane.Run("./my-server")

    // Test interaction between panes
    rightPane.WaitForText("Server started on :8080")

    leftPane.Select().
        Type("q") // Quit htop

    leftPane.Expect().Not().ToContainText("htop")
}
```

### Async Waiting

```go
func TestAsyncUpdates(t *testing.T) {
    session := tuikit.New(t, tuikit.WithTimeout(30*time.Second))
    pane := session.Run("./slow-loading-app")

    // Wait for loading to complete
    pane.WaitForPattern(`Loading.*100%`)

    // Or wait until stable
    pane.WaitForStable(500 * time.Millisecond)

    // Custom wait condition
    pane.WaitUntil(func(s *Screen) bool {
        return !s.Contains("Loading") && s.Contains("Ready")
    })
}
```

### Style Assertions

```go
func TestTextStyles(t *testing.T) {
    session := tuikit.New(t)
    pane := session.Run("./styled-app")

    pane.Expect().
        ToHaveStyledText("Error:", matchers.Bold()).
        ToHaveStyledText("Error:", matchers.FG(matchers.Red)).
        RegionToHaveStyle(0, 0, 10, 1, matchers.Styled(
            matchers.Bold(),
            matchers.BG(matchers.Blue),
        ))
}
```

### Snapshot Testing

```go
func TestUISnapshot(t *testing.T) {
    session := tuikit.New(t)
    pane := session.Run("./my-app", "--demo-mode")

    pane.WaitForStable(200 * time.Millisecond)

    // Compare against saved snapshot
    // First run creates snapshot, subsequent runs compare
    pane.Expect().ToMatchSnapshot("demo-screen")
}
```

### Debug and Recording

```go
func TestWithRecording(t *testing.T) {
    session := tuikit.New(t,
        tuikit.WithDebug("./debug-output"),
    )

    pane := session.Run("./my-app")

    // Take manual screenshots
    pane.Screenshot("initial-state")

    pane.Type("test input").Press(keys.Enter)

    pane.Screenshot("after-input")

    // Recording is saved to debug-output/
    // - screenshots as PNG
    // - session.html for replay
    // - session.cast for asciicast
}
```

### Custom Matchers

```go
// Custom matcher for table content
type TableMatcher struct {
    headers []string
    rows    [][]string
}

func (m *TableMatcher) Match(screen *Screen) matchers.MatchResult {
    text := screen.Text()
    // Parse table from text...
    return matchers.MatchResult{
        Passed:  matches,
        Message: "table content matches",
    }
}

func (m *TableMatcher) Description() string {
    return "table with specified content"
}

// Usage
func TestTable(t *testing.T) {
    pane := tuikit.New(t).Run("./table-app")

    pane.Expect().ToPass(&TableMatcher{
        headers: []string{"Name", "Age"},
        rows: [][]string{
            {"Alice", "30"},
            {"Bob", "25"},
        },
    })
}
```

---

## Implementation Phases

### Phase 1: Core Foundation

**Deliverables:**
1. Basic session management (create, kill, cleanup)
2. Tmux client wrapper (exec, capture, send-keys)
3. Simple text content capture (no ANSI parsing)
4. Basic `Press()` and `Type()` input methods
5. Integration with `testing.T` for cleanup
6. Basic `WaitForText()` implementation

**Key Files:**
- `tuikit.go` - Entry point with `New()`
- `session.go` - Session type
- `pane.go` - Basic pane operations
- `internal/tmux/client.go` - Tmux CLI wrapper

**Milestone Test:**
```go
func TestPhase1(t *testing.T) {
    session := tuikit.New(t)
    pane := session.Run("echo", "hello world")
    pane.WaitForText("hello world")
}
```

### Phase 2: Input and Control

**Deliverables:**
1. Complete key constants (`keys/keys.go`)
2. Key sequence builder
3. Control key combinations (Ctrl, Alt, Shift)
4. Window and pane management (split, resize)
5. Layout presets
6. Pane selection and focus

**Key Files:**
- `keys/keys.go` - Key constants
- `keys/sequence.go` - Sequence builder
- `window.go` - Window operations
- `input.go` - Input helpers

**Milestone Test:**
```go
func TestPhase2(t *testing.T) {
    session := tuikit.New(t)
    pane := session.Run("vim")
    pane.Press(keys.Escape).
        Type(":q!").
        Press(keys.Enter)
}
```

### Phase 3: Screen Model and ANSI Parsing

**Deliverables:**
1. ANSI escape sequence parser
2. Cell and Style types
3. Screen buffer with full content model
4. Color extraction (16, 256, truecolor)
5. Style attribute extraction
6. ANSI stripping utilities

**Key Files:**
- `ansi/parser.go` - State machine parser
- `ansi/style.go` - Style extraction
- `ansi/cell.go` - Cell representation
- `screen.go` - Screen buffer

**Milestone Test:**
```go
func TestPhase3(t *testing.T) {
    session := tuikit.New(t)
    pane := session.Run("./colored-output")

    screen := pane.Screen()
    cell := screen.Cell(0, 0)
    if !cell.Style.Bold {
        t.Error("expected bold text")
    }
}
```

### Phase 4: Assertions and Matchers

**Deliverables:**
1. Expect fluent API builder
2. Text matchers (contains, pattern, exact)
3. Cursor position matchers
4. Style matchers
5. Region matchers
6. Custom matcher interface
7. Rich assertion error messages

**Key Files:**
- `expect.go` - Fluent assertion builder
- `matcher.go` - Matcher interface
- `matchers/text.go` - Text matchers
- `matchers/style.go` - Style matchers
- `matchers/cursor.go` - Cursor matchers

**Milestone Test:**
```go
func TestPhase4(t *testing.T) {
    session := tuikit.New(t)
    pane := session.Run("./my-app")

    pane.Expect().
        ToContainText("Welcome").
        ToContainPattern(`Version \d+\.\d+`).
        CursorAt(0, 0).
        Not().ToContainText("Error")
}
```

### Phase 5: Advanced Features

**Deliverables:**
1. Snapshot testing
2. Screen diff
3. Debug recording
4. Screenshot capture (text and optionally PNG via external tool)
5. HTML replay generation
6. Asciicast export
7. Scrollback history capture

**Key Files:**
- `internal/debug/recorder.go` - Recording
- `internal/debug/snapshot.go` - Screenshots
- `snapshot.go` - Snapshot testing

**Milestone Test:**
```go
func TestPhase5(t *testing.T) {
    session := tuikit.New(t, tuikit.WithDebug("./debug"))
    pane := session.Run("./my-app")

    pane.Expect().ToMatchSnapshot("initial")

    pane.Type("input").Press(keys.Enter)
    pane.Screenshot("after-input")

    pane.Expect().ToMatchSnapshot("final")
}
```

### Phase 6: Polish and Performance

**Deliverables:**
1. Session pooling for faster tests
2. Parallel test support
3. Comprehensive documentation
4. Example tests for common TUI frameworks (Bubble Tea, etc.)
5. Performance benchmarks
6. CI/CD integration examples

**Key Files:**
- `internal/pool/pool.go` - Session pooling
- `examples/*.go` - Example tests
- `doc.go` - Package documentation

---

## Testing Strategy for the Library

### Unit Tests

```go
// ansi/parser_test.go
func TestParserSGR(t *testing.T) {
    p := NewParser()
    cells := p.ParseString("\x1b[1;31mHello\x1b[0m")

    if !cells[0].Style.Bold {
        t.Error("expected bold")
    }
    if cells[0].Style.FG != ColorRed {
        t.Error("expected red foreground")
    }
}

// internal/tmux/client_test.go
func TestClientExec(t *testing.T) {
    // Test command execution with mock
}
```

### Integration Tests

```go
// integration_test.go
func TestRealTmuxSession(t *testing.T) {
    if _, err := exec.LookPath("tmux"); err != nil {
        t.Skip("tmux not available")
    }

    session := tuikit.New(t)
    pane := session.Run("bash", "-c", "echo test")
    pane.WaitForText("test")
}
```

### End-to-End Tests

```go
// e2e/bubbletea_test.go
func TestBubbleTeaApp(t *testing.T) {
    session := tuikit.New(t)
    pane := session.Run("./testdata/bubbletea-example")

    pane.WaitForText("What's your name?")
    pane.Type("Claude").Press(keys.Enter)
    pane.Expect().ToContainText("Hello, Claude!")
}
```

### Test Fixtures

```
testdata/
├── simple-echo/          # Simple shell echo script
├── bubbletea-counter/    # Bubble Tea counter example
├── vim-like/             # Modal editor simulation
├── split-pane-app/       # Multi-pane application
└── slow-loader/          # Async loading simulation
```

### Benchmarks

```go
func BenchmarkScreenCapture(b *testing.B) {
    session := tuikit.New(b)
    pane := session.Run("./large-output-app")

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = pane.Screen()
    }
}

func BenchmarkANSIParsing(b *testing.B) {
    data := loadLargeANSIOutput()
    parser := ansi.NewParser()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        parser.Parse(data)
    }
}
```

---

## Performance Considerations

### Efficient Screen Capture

```go
// Use tmux capture-pane with specific options for performance
func (c *Client) CapturePane(target string, opts CaptureOptions) (string, error) {
    args := []string{"capture-pane", "-t", target, "-p"}

    // Only include escape codes when needed
    if opts.EscapeCodes {
        args = append(args, "-e")
    }

    // Limit capture range for large scrollback
    if opts.StartLine != 0 {
        args = append(args, "-S", strconv.Itoa(opts.StartLine))
    }
    if opts.EndLine != 0 {
        args = append(args, "-E", strconv.Itoa(opts.EndLine))
    }

    return c.Exec(args...)
}
```

### Parser Optimization

1. **Object Pooling**: Reuse parser instances
2. **Lazy Parsing**: Only parse ANSI codes when styles are accessed
3. **Streaming**: Process large outputs in chunks

```go
// Parser pooling
var parserPool = sync.Pool{
    New: func() interface{} {
        return NewParser()
    },
}

func GetParser() *Parser {
    return parserPool.Get().(*Parser)
}

func PutParser(p *Parser) {
    p.Reset()
    parserPool.Put(p)
}
```

### Session Pooling (Optional)

```go
// For test suites that create many sessions
type SessionPool struct {
    sessions chan *Session
    size     int
}

func NewPool(t testing.TB, size int) *SessionPool

func (p *SessionPool) Get() *Session
func (p *SessionPool) Put(s *Session)
```

---

## Debugging Features

### Verbose Mode

```go
session := tuikit.New(t,
    tuikit.WithDebug("./debug"),
    tuikit.WithVerbose(), // Log all tmux commands
)
```

### On-Failure Snapshots

```go
// Automatically capture screen on test failure
func (s *Session) cleanup() {
    if s.t.Failed() && s.config.Debug {
        s.Snapshot().SaveTo(s.config.DebugDir)
    }
    s.Kill()
}
```

### Interactive Debug Mode

```go
// Pause test and allow manual inspection
func (p *Pane) DebugPause() *Pane {
    fmt.Printf("Test paused. Session: %s\n", p.session.ID())
    fmt.Printf("Attach with: tmux -S %s attach\n", p.session.socketPath)
    fmt.Print("Press Enter to continue...")
    bufio.NewReader(os.Stdin).ReadBytes('\n')
    return p
}
```

---

## Dependencies

### Required
- Go 1.21+ (for generics and improved testing)
- tmux 3.0+ (for consistent behavior)

### Optional External Tools
- `asciinema` - For recording (if asciicast export is used)
- `ansitoimg` - For PNG screenshot generation

### No Third-Party Go Dependencies
The library uses only the standard library to minimize dependency conflicts and maintenance burden.

---

## Critical Implementation Files

These files form the foundation of the library and should be implemented first:

| File | Purpose |
|------|---------|
| `tuikit.go` | Entry point with `New()` function and `Config` type |
| `pane.go` | Core `Pane` type with input simulation and screen capture |
| `internal/tmux/client.go` | Low-level tmux CLI wrapper |
| `ansi/parser.go` | ANSI escape sequence parser |
| `expect.go` | Fluent assertion builder |

---

## References

- [gotmux - Go library for tmux](https://github.com/GianlucaP106/gotmux)
- [Charmbracelet x/ansi - ANSI parsing package](https://pkg.go.dev/github.com/charmbracelet/x/ansi)
- [Playwright Fluent API pattern](https://dev.to/10-minutes-qa-story/fluent-api-pattern-implementation-with-playwright-and-javascripttypescript-2lk1)
- [Gomega matcher library](https://pkg.go.dev/github.com/onsi/gomega)
- [Using tmux to test console applications](https://www.drmaciver.com/2015/05/using-tmux-to-test-your-console-applications/)
