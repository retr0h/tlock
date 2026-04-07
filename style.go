package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	purple = lipgloss.Color("99")
	teal   = lipgloss.Color("#06ffa5")
	gray   = lipgloss.Color("245")
	red    = lipgloss.Color("196")

	lockTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(purple)
	subtitleStyle  = lipgloss.NewStyle().Foreground(gray)
	promptStyle    = lipgloss.NewStyle().Foreground(teal)
	errorStyle     = lipgloss.NewStyle().Bold(true).Foreground(red)
)

var glitchBorder = lipgloss.Border{
	Top:         "\u2591\u2592\u2593\u2588\u2593\u2592\u2591",
	Bottom:      "\u2591\u2592\u2593\u2588\u2593\u2592\u2591",
	Left:        "\u2593",
	Right:       "\u2593",
	TopLeft:     "\u2588",
	TopRight:    "\u2588",
	BottomLeft:  "\u2588",
	BottomRight: "\u2588",
}

var msgBoxStyle = lipgloss.NewStyle().
	Border(glitchBorder).
	BorderForeground(teal).
	Padding(1, 4).
	Width(50).
	Foreground(teal).
	Bold(true)

var errBoxStyle = lipgloss.NewStyle().
	Border(glitchBorder).
	BorderForeground(red).
	Padding(1, 4).
	Width(50).
	Foreground(red).
	Bold(true)

func renderMessage(msg string, style lipgloss.Style) {
	w, h := getTermSize()
	clearScreen()

	var box string
	if style.GetForeground() == red {
		box = errBoxStyle.Render(msg)
	} else {
		box = msgBoxStyle.Render(msg)
	}

	lines := strings.Split(box, "\n")
	startRow := (h - len(lines)) / 2
	fmt.Printf("\033[%d;0H", startRow)
	fmt.Printf("%s\r\n", centerBlock(box, w))
}

// renderMessageOverlay renders a boxed message over existing content without clearing the screen
func renderMessageOverlay(msg string, style lipgloss.Style) {
	w, h := getTermSize()

	var box string
	if style.GetForeground() == red {
		box = errBoxStyle.Render(msg)
	} else {
		box = msgBoxStyle.Render(msg)
	}

	lines := strings.Split(box, "\n")
	boxWidth := 0
	for _, line := range lines {
		lw := lipgloss.Width(line)
		if lw > boxWidth {
			boxWidth = lw
		}
	}

	clearRect(boxWidth, len(lines), 3)

	startRow := (h - len(lines)) / 2
	fmt.Printf("\033[%d;0H", startRow)
	fmt.Printf("%s\r\n", centerBlock(box, w))
}
