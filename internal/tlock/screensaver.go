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

package tlock

import "math/rand"

type screensaver interface {
	// run starts the screensaver animation loop.
	// stopCh signals the screensaver to exit (for cycling).
	// keyCh delivers keypresses from the shared stdin reader.
	// Returns true if the user authenticated, false otherwise.
	run(stopCh <-chan struct{}, keyCh <-chan byte) bool
}

var screensaverNames = []string{"worms", "pipes", "dvd"}

var screensaverFactory = map[string]func() screensaver{
	"worms": func() screensaver { return &wormScreensaver{} },
	"pipes": func() screensaver { return &pipesScreensaver{} },
	"dvd":   func() screensaver { return &dvdScreensaver{} },
}

func pickRandomScreensaver(exclude string) string {
	for {
		name := screensaverNames[rand.Intn(len(screensaverNames))]
		if name != exclude {
			return name
		}
	}
}
