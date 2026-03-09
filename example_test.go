package strider_test

import (
	"testing"
	"time"

	"github.com/cboone/strider"
)

func ExampleOpen() {
	_ = func(t *testing.T) {
		term := strider.Open(t, "./my-app",
			strider.WithArgs("--verbose"),
			strider.WithSize(120, 40),
			strider.WithTimeout(10*time.Second),
		)
		term.WaitFor(strider.Text("Welcome"))
	}
}

func ExampleTerminal_WaitFor() {
	_ = func(t *testing.T) {
		term := strider.Open(t, "./my-app")
		term.WaitFor(strider.Text("Name:"))
		term.Type("Alice")
		term.Press(strider.Enter)
		term.WaitFor(strider.All(
			strider.Text("Saved"),
			strider.Not(strider.Text("Error")),
		))
	}
}

func ExampleTerminal_MatchSnapshot() {
	_ = func(t *testing.T) {
		term := strider.Open(t, "./my-app")
		term.WaitFor(strider.Text("Dashboard"))
		term.MatchSnapshot("dashboard")
	}
}
