// Package main implements tlock, a terminal lock screen for macOS with
// Touch ID and password authentication.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/term"
)

func main() {
	snake := flag.Bool(
		"snake",
		false,
		"Screensaver on immediately (shortcut for --screensaver --screensaver-delay 0)",
	)
	snakeCount := flag.Int("snake-count", 0, "Number of worms (0 = auto based on terminal size)")
	screensaver := flag.Bool("screensaver", false, "Enable xlock-style worm screensaver")
	screensaverDelay := flag.Int(
		"screensaver-delay",
		30,
		"Seconds idle before screensaver starts (0 = immediate)",
	)
	flag.Parse()

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set raw mode: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	hideCursor()
	defer showCursor()
	defer clearScreen()

	// Ignore signals that could bypass the lock
	signal.Ignore(syscall.SIGINT, syscall.SIGTERM, syscall.SIGTSTP)

	// --snake is a shortcut: screensaver on, delay 0
	if *snake {
		*screensaver = true
		*screensaverDelay = 0
	}

	for {
		if *screensaver && *screensaverDelay == 0 {
			// Screensaver immediately — worms run, keypress triggers auth
			if runWormDemo(*snakeCount) {
				return
			}
		} else if *screensaver {
			// Show password prompt, start screensaver after delay if idle
			// Run password prompt with a timeout
			pwCh := make(chan string, 1)
			go func() {
				pwCh <- readPasswordOverlay(false)
			}()

			timer := time.NewTimer(time.Duration(*screensaverDelay) * time.Second)
			select {
			case pw := <-pwCh:
				timer.Stop()
				if handleAuth(pw) {
					return
				}
				continue
			case <-timer.C:
				// Timeout — switch to screensaver
				if runWormDemo(*snakeCount) {
					return
				}
				continue
			}
		} else {
			// No screensaver — just password prompt
			pw := readPasswordOverlay(false)
			if handleAuth(pw) {
				return
			}
		}
	}
}
