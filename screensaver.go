package main

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
