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
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

func hideCursor() {
	fmt.Print("\033[?25l")
}

func showCursor() {
	fmt.Print("\033[?25h")
}

func centerText(text string, width int) string {
	pad := (width - lipgloss.Width(text)) / 2
	if pad < 0 {
		pad = 0
	}
	return fmt.Sprintf("%*s%s", pad, "", text)
}

func centerBlock(block string, width int) string {
	lines := strings.Split(block, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		result = append(result, centerText(line, width))
	}
	return strings.Join(result, "\r\n")
}

func getTermSize() (int, int) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80, 24
	}
	return w, h
}

func drawLockIcon() {
	silver := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	brass := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB000"))
	dark := lipgloss.NewStyle().Foreground(lipgloss.Color("236"))

	lines := []string{
		" " + silver.Render("\u2593\u2593\u2593") + " ",
		silver.Render("\u2593") + "   " + silver.Render("\u2593"),
		brass.Render("\u2588\u2588\u2588\u2588\u2588"),
		brass.Render("\u2588") + dark.Render(" \u2588 ") + brass.Render("\u2588"),
		brass.Render("\u2588\u2588\u2588\u2588\u2588"),
	}
	for i, line := range lines {
		fmt.Printf("\033[%d;2H%s", i+1, line)
	}
}

func clearRect(boxWidth, boxHeight, padding int) {
	w, h := getTermSize()
	totalW := boxWidth + padding*2
	totalH := boxHeight + padding*2
	startCol := (w - totalW) / 2
	startRow := (h-boxHeight)/2 - padding

	if startCol < 0 {
		startCol = 0
	}
	if startRow < 1 {
		startRow = 1
	}

	blank := strings.Repeat(" ", totalW)
	for row := startRow; row < startRow+totalH && row <= h; row++ {
		fmt.Printf("\033[%d;%dH%s", row, startCol+1, blank)
	}
}
