// Command testbin is a minimal TUI fixture program for testing the crawler
// library. It reads stdin line by line and responds to commands.
//
// Behavior:
//   - On startup, prints "ready>" prompt
//   - On Enter, processes the current line:
//   - "quit": exits with status 0
//   - "fail": exits with status 1
//   - "lines N": prints N numbered lines (for scrollback testing)
//   - "size": prints the terminal size
//   - Anything else: prints "echo: <line>" and a new "ready>" prompt
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

func main() {
	// Track terminal size via SIGWINCH.
	var (
		mu         sync.Mutex
		cols, rows int
	)

	// Get initial size.
	if c, r, err := getTermSize(os.Stdout.Fd()); err == nil {
		mu.Lock()
		cols, rows = c, r
		mu.Unlock()
	}

	// Listen for SIGWINCH.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	go func() {
		for range sigCh {
			if c, r, err := getTermSize(os.Stdout.Fd()); err == nil {
				mu.Lock()
				cols, rows = c, r
				mu.Unlock()
			}
		}
	}()

	fmt.Print("ready>")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := scanner.Text()

		switch {
		case input == "quit":
			os.Exit(0)

		case input == "fail":
			os.Exit(1)

		case strings.HasPrefix(input, "lines "):
			countStr := strings.TrimPrefix(input, "lines ")
			count, parseErr := strconv.Atoi(countStr)
			if parseErr != nil {
				fmt.Printf("error: invalid count %q\n", countStr)
			} else {
				for i := 1; i <= count; i++ {
					fmt.Printf("line %d\n", i)
				}
			}
			fmt.Print("ready>")

		case input == "size":
			mu.Lock()
			fmt.Printf("size: %dx%d\n", cols, rows)
			mu.Unlock()
			fmt.Print("ready>")

		default:
			fmt.Printf("echo: %s\n", input)
			fmt.Print("ready>")
		}
	}
}

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func getTermSize(fd uintptr) (cols, rows int, err error) {
	var ws winsize
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd,
		uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&ws)))
	if errno != 0 {
		return 0, 0, errno
	}
	return int(ws.Col), int(ws.Row), nil
}
