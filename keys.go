package crawler

import "fmt"

// Key represents a tmux key sequence.
type Key string

// Special key constants for use with Press.
const (
	Enter     Key = "Enter"
	Escape    Key = "Escape"
	Tab       Key = "Tab"
	Backspace Key = "BSpace"
	Up        Key = "Up"
	Down      Key = "Down"
	Left      Key = "Left"
	Right     Key = "Right"
	Home      Key = "Home"
	End       Key = "End"
	PageUp    Key = "PageUp"
	PageDown  Key = "PageDown"
	Space     Key = "Space"
	Delete    Key = "DC"

	F1  Key = "F1"
	F2  Key = "F2"
	F3  Key = "F3"
	F4  Key = "F4"
	F5  Key = "F5"
	F6  Key = "F6"
	F7  Key = "F7"
	F8  Key = "F8"
	F9  Key = "F9"
	F10 Key = "F10"
	F11 Key = "F11"
	F12 Key = "F12"
)

// Ctrl returns the key sequence for Ctrl+<char>.
func Ctrl(c byte) Key {
	return Key(fmt.Sprintf("C-%c", c))
}

// Alt returns the key sequence for Alt+<char>.
func Alt(c byte) Key {
	return Key(fmt.Sprintf("M-%c", c))
}
