package crawler_test

import (
	"testing"
	"time"

	"github.com/cboone/crawler"
)

func ExampleOpen() {
	_ = func(t *testing.T) {
		term := crawler.Open(t, "./my-app",
			crawler.WithArgs("--verbose"),
			crawler.WithSize(120, 40),
			crawler.WithTimeout(10*time.Second),
		)
		term.WaitFor(crawler.Text("Welcome"))
	}
}

func ExampleTerminal_WaitFor() {
	_ = func(t *testing.T) {
		term := crawler.Open(t, "./my-app")
		term.WaitFor(crawler.Text("Name:"))
		term.Type("Alice")
		term.Press(crawler.Enter)
		term.WaitFor(crawler.All(
			crawler.Text("Saved"),
			crawler.Not(crawler.Text("Error")),
		))
	}
}

func ExampleTerminal_MatchSnapshot() {
	_ = func(t *testing.T) {
		term := crawler.Open(t, "./my-app")
		term.WaitFor(crawler.Text("Dashboard"))
		term.MatchSnapshot("dashboard")
	}
}
