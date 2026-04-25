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

// Package cmd holds the cobra command tree for tlock. The lock
// implementation lives in internal/tlock — this package is the
// thin CLI surface that parses flags and delegates to tlock.Run.
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/retr0h/tlock/internal/tlock"
)

var (
	delayFlag      string
	cycleFlag      string
	wormsFlag      bool
	snakeFlag      bool
	pipesFlag      bool
	dvdFlag        bool
	randomFlag     bool
	wormCountFlag  int
	snakeCountFlag int
)

var rootCmd = &cobra.Command{
	Use:   "tlock",
	Short: "Terminal lock screen for macOS with Touch ID and password authentication.",
	Long: "tlock locks the current terminal until you authenticate via Touch ID\n" +
		"or password. Optional screensavers (worms, pipes, dvd) run while locked,\n" +
		"with --random + --cycle rotating through them.",
	SilenceUsage: true,
	RunE: func(_ *cobra.Command, _ []string) error {
		// Resolve shortcut flags into the canonical screensaver name
		// the lock loop understands. --worm-count / --snake-count are
		// aliases; --worm-count wins when both are non-zero.
		var screensaver string
		switch {
		case wormsFlag || snakeFlag:
			screensaver = "worms"
		case pipesFlag:
			screensaver = "pipes"
		case dvdFlag:
			screensaver = "dvd"
		case randomFlag:
			screensaver = "random"
		}

		var delay time.Duration
		if delayFlag != "" {
			d, err := time.ParseDuration(delayFlag)
			if err != nil {
				return fmt.Errorf("invalid --delay: %w", err)
			}
			delay = d
		}

		var cycle time.Duration
		if cycleFlag != "" {
			d, err := time.ParseDuration(cycleFlag)
			if err != nil {
				return fmt.Errorf("invalid --cycle: %w", err)
			}
			cycle = d
		}

		numWorms := wormCountFlag
		if snakeCountFlag > 0 && numWorms == 0 {
			numWorms = snakeCountFlag
		}

		return tlock.Run(tlock.Config{
			Screensaver: screensaver,
			Delay:       delay,
			Cycle:       cycle,
			NumWorms:    numWorms,
		})
	},
}

func init() {
	f := rootCmd.Flags()
	f.StringVar(
		&delayFlag, "delay", "",
		"Duration idle before screensaver starts, e.g. 30s, 5m (default: immediate)",
	)
	f.StringVar(
		&cycleFlag, "cycle", "",
		"Duration between screensaver rotation, e.g. 30s, 5m, 1h (with --random)",
	)
	f.BoolVar(&wormsFlag, "worms", false, "Launch worms screensaver immediately")
	f.BoolVar(&snakeFlag, "snake", false, "Alias for --worms")
	f.BoolVar(&pipesFlag, "pipes", false, "Launch pipes screensaver immediately")
	f.BoolVar(&dvdFlag, "dvd", false, "Launch DVD lock screensaver immediately")
	f.BoolVar(&randomFlag, "random", false, "Launch a random screensaver immediately")
	f.IntVar(
		&wormCountFlag, "worm-count", 0,
		"Number of worms (0 = auto based on terminal size)",
	)
	f.IntVar(&snakeCountFlag, "snake-count", 0, "Alias for --worm-count")
}

// Execute runs the cobra command tree. main.go is just a shell that
// calls this — non-zero exit propagates the error up to the OS.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
