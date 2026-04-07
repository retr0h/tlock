package main

import "math/rand"

type screensaver interface {
	run() bool // returns true if user authenticated
}

var screensaverNames = []string{"snake", "pipes", "dvd"}

var screensaverFactory = map[string]func() screensaver{
	"snake": func() screensaver { return &wormScreensaver{} },
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

type pipesScreensaver struct{}

func (p *pipesScreensaver) run() bool { return false }

type dvdScreensaver struct{}

func (d *dvdScreensaver) run() bool { return false }
