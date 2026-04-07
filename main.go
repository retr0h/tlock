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
	screensaverName := flag.String(
		"screensaver",
		"",
		"Screensaver to run: snake, pipes, dvd, random",
	)
	screensaverDelay := flag.Int(
		"screensaver-delay",
		30,
		"Seconds idle before screensaver starts (0 = immediate)",
	)
	_ = flag.Int(
		"screensaver-cycle",
		0,
		"Minutes between screensaver rotation when using random (0 = disabled)",
	)

	// --snake is an alias for --screensaver snake --screensaver-delay 0
	snake := flag.Bool(
		"snake",
		false,
		"Shortcut for --screensaver snake --screensaver-delay 0",
	)
	snakeCount := flag.Int("snake-count", 0, "Number of worms (0 = auto based on terminal size)")
	wormCount := flag.Int("worm-count", 0, "Alias for --snake-count")

	flag.Parse()

	// Resolve aliases
	if *snake {
		*screensaverName = "snake"
		*screensaverDelay = 0
	}

	// --worm-count wins over --snake-count when both provided; otherwise take whichever is non-zero
	numWorms := *snakeCount
	if *wormCount > 0 {
		numWorms = *wormCount
	}

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

	// Resolve "random" to a concrete name once per session
	activeName := *screensaverName
	if activeName == "random" {
		activeName = pickRandomScreensaver("")
	}

	buildScreensaver := func() screensaver {
		factory, ok := screensaverFactory[activeName]
		if !ok {
			return nil
		}
		ss := factory()
		if ws, ok := ss.(*wormScreensaver); ok {
			ws.numWorms = numWorms
		}
		return ss
	}

	for {
		if activeName != "" && *screensaverDelay == 0 {
			// Run screensaver immediately; it returns true on successful auth
			ss := buildScreensaver()
			if ss != nil && ss.run() {
				return
			}
		} else if activeName != "" {
			// Show password prompt; switch to screensaver after idle timeout
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
				ss := buildScreensaver()
				if ss != nil && ss.run() {
					return
				}
				continue
			}
		} else {
			// No screensaver — password prompt only
			pw := readPasswordOverlay(false)
			if handleAuth(pw) {
				return
			}
		}
	}
}
