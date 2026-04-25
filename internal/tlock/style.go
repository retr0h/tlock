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

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	teal = lipgloss.Color("#06ffa5")
	gray = lipgloss.Color("245")
	red  = lipgloss.Color("196")

	subtitleStyle = lipgloss.NewStyle().Foreground(gray)
	errorStyle    = lipgloss.NewStyle().Bold(true).Foreground(red)
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
