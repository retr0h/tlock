// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

// Package tlock holds the terminal-lock implementation. The cmd/ tree
// is the cobra surface that parses flags and calls Run; everything
// else (auth, screensavers, terminal helpers) lives here as the
// importable package the cmd/ tree depends on.
package tlock

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/term"
)

// Config is the runtime configuration assembled by the cmd/ layer
// (or any other caller). Zero values are valid:
//
//   - Screensaver "" → password prompt only, no screensaver
//   - Delay 0       → if a screensaver is set, launch it immediately
//   - Cycle 0       → no rotation; "random" picks once and sticks
//   - NumWorms 0    → auto-size based on terminal dimensions
type Config struct {
	Screensaver string // "", "worms", "pipes", "dvd", "random"
	Delay       time.Duration
	Cycle       time.Duration
	NumWorms    int
}

// Run is the locked-screen event loop. Returns nil when the user
// authenticates and we exit cleanly. Errors are returned only for
// unrecoverable terminal-setup failures (raw mode, etc.).
func Run(cfg Config) error {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("set raw mode: %w", err)
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	hideCursor()
	defer showCursor()
	defer clearScreen()

	// Ignore signals that could bypass the lock.
	signal.Ignore(syscall.SIGINT, syscall.SIGTERM, syscall.SIGTSTP)

	// Single persistent stdin reader — shared across all screensavers.
	// This prevents goroutine leaks when cycling between screensavers:
	// each cycle previously left an orphan goroutine blocked on
	// stdin.Read, stealing keypresses from the active screensaver.
	keyCh := make(chan byte, 4)
	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			if n > 0 {
				keyCh <- buf[0]
			}
		}
	}()

	isRandom := cfg.Screensaver == "random"
	activeName := cfg.Screensaver
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
			ws.numWorms = cfg.NumWorms
		}
		return ss
	}

	// runWithCycle runs screensavers in a loop, rotating every
	// cfg.Cycle. Returns true if the user authenticated.
	runWithCycle := func() bool {
		current := activeName
		for {
			stopCh := make(chan struct{})

			timer := time.NewTimer(cfg.Cycle)
			authCh := make(chan bool, 1)

			go func() {
				ss := buildScreensaver(current)
				if ss != nil {
					authCh <- ss.run(stopCh, keyCh)
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
				current = pickRandomScreensaver(current)
			case <-timer.C:
				close(stopCh)
				<-authCh
				current = pickRandomScreensaver(current)
			}
		}
	}

	neverStop := make(chan struct{}) // never closed — for non-cycling mode

	for {
		switch {
		case activeName != "" && cfg.Delay == 0:
			if isRandom && cfg.Cycle > 0 {
				if runWithCycle() {
					return nil
				}
			} else {
				ss := buildScreensaver(activeName)
				if ss != nil && ss.run(neverStop, keyCh) {
					return nil
				}
			}
		case activeName != "":
			// Show password prompt; switch to screensaver after idle timeout.
			pwCh := make(chan string, 1)
			go func() {
				pwCh <- readPasswordOverlay(false, keyCh)
			}()

			timer := time.NewTimer(cfg.Delay)
			select {
			case pw := <-pwCh:
				timer.Stop()
				if handleAuth(pw) {
					return nil
				}
			case <-timer.C:
				if isRandom && cfg.Cycle > 0 {
					if runWithCycle() {
						return nil
					}
				} else {
					ss := buildScreensaver(activeName)
					if ss != nil && ss.run(neverStop, keyCh) {
						return nil
					}
				}
			}
		default:
			// No screensaver — password prompt only.
			pw := readPasswordOverlay(false, keyCh)
			if handleAuth(pw) {
				return nil
			}
		}
	}
}
