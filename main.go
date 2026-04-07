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
	screensaverName := new(string)
	delayStr := flag.String(
		"delay",
		"",
		"Duration idle before screensaver starts, e.g. 30s, 5m (default: immediate)",
	)
	cycleStr := flag.String(
		"cycle",
		"",
		"Duration between screensaver rotation, e.g. 30s, 5m, 1h (with --random)",
	)

	// Shortcut flags — each launches its screensaver immediately
	snake := flag.Bool("snake", false, "Launch snake screensaver immediately")
	pipes := flag.Bool("pipes", false, "Launch pipes screensaver immediately")
	dvd := flag.Bool("dvd", false, "Launch DVD lock screensaver immediately")
	random := flag.Bool("random", false, "Launch a random screensaver immediately")

	snakeCount := flag.Int("snake-count", 0, "Number of worms (0 = auto based on terminal size)")
	wormCount := flag.Int("worm-count", 0, "Alias for --snake-count")

	flag.Parse()

	// Resolve shortcut flags
	switch {
	case *snake:
		*screensaverName = "snake"
	case *pipes:
		*screensaverName = "pipes"
	case *dvd:
		*screensaverName = "dvd"
	case *random:
		*screensaverName = "random"
	}

	// Parse durations
	var delay time.Duration
	if *delayStr != "" {
		var err error
		delay, err = time.ParseDuration(*delayStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --delay: %v\n", err)
			os.Exit(1)
		}
	}

	var cycleDur time.Duration
	if *cycleStr != "" {
		var err error
		cycleDur, err = time.ParseDuration(*cycleStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --cycle: %v\n", err)
			os.Exit(1)
		}
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

	isRandom := *screensaverName == "random"
	activeName := *screensaverName
	if isRandom {
		activeName = pickRandomScreensaver("")
	}

	buildScreensaver := func(name string) screensaver {
		factory, ok := screensaverFactory[name]
		if !ok {
			return nil
		}
		ss := factory()
		if ws, ok := ss.(*wormScreensaver); ok {
			ws.numWorms = numWorms
		}
		return ss
	}

	// runWithCycle runs screensavers in a loop, rotating every cycleDur.
	// Returns true if user authenticated.
	runWithCycle := func() bool {
		current := activeName
		for {
			stopCh := make(chan struct{})

			// Start cycle timer
			timer := time.NewTimer(cycleDur)
			authCh := make(chan bool, 1)

			go func() {
				ss := buildScreensaver(current)
				if ss != nil {
					authCh <- ss.run(stopCh)
				} else {
					authCh <- false
				}
			}()

			select {
			case authenticated := <-authCh:
				timer.Stop()
				if authenticated {
					return true
				}
				// Auth failed inside screensaver, pick next
				current = pickRandomScreensaver(current)
			case <-timer.C:
				// Time to rotate
				close(stopCh)
				<-authCh // wait for screensaver to exit
				current = pickRandomScreensaver(current)
			}
		}
	}

	neverStop := make(chan struct{}) // never closed — for non-cycling mode

	for {
		if activeName != "" && delay == 0 {
			if isRandom && cycleDur > 0 {
				if runWithCycle() {
					return
				}
			} else {
				ss := buildScreensaver(activeName)
				if ss != nil && ss.run(neverStop) {
					return
				}
			}
		} else if activeName != "" {
			// Show password prompt; switch to screensaver after idle timeout
			pwCh := make(chan string, 1)
			go func() {
				pwCh <- readPasswordOverlay(false)
			}()

			timer := time.NewTimer(delay)
			select {
			case pw := <-pwCh:
				timer.Stop()
				if handleAuth(pw) {
					return
				}
				continue
			case <-timer.C:
				if isRandom && cycleDur > 0 {
					if runWithCycle() {
						return
					}
				} else {
					ss := buildScreensaver(activeName)
					if ss != nil && ss.run(neverStop) {
						return
					}
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
