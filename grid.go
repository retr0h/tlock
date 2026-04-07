package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Grid cell size in terminal characters (width x height)
// Terminal chars are ~2:1 tall:wide, so 3x2 looks roughly square
const (
	cellW = 3
	cellH = 2
)

type point struct {
	x, y int
}

// Grid cell states
const (
	cellEmpty = iota
	cellBody
	cellTrail
	cellLock
)

type gridCell struct {
	state    int
	wormIdx  int
	trailAge int
	color    lipgloss.Color
}

// Retro phosphor CRT palette
var wormColors = []lipgloss.Color{
	lipgloss.Color("#06ffa5"), // teal phosphor
	lipgloss.Color("#00ff00"), // green phosphor (P1)
	lipgloss.Color("#33ff33"), // bright green
	lipgloss.Color("#FFB000"), // amber phosphor (P3)
	lipgloss.Color("#ff6600"), // hot amber
	lipgloss.Color("#00ffff"), // cyan
	lipgloss.Color("#ff00ff"), // magenta CRT bleed
	lipgloss.Color("99"),      // purple
	lipgloss.Color("#ff3366"), // hot pink scanline
	lipgloss.Color("#66ff66"), // soft green
	lipgloss.Color("#ffff33"), // yellow phosphor burn
	lipgloss.Color("#cc66ff"), // lavender
	lipgloss.Color("#ff9933"), // warm orange
}

// Direction deltas: up, right, down, left (cardinal only, in grid coords)
var (
	dx = []int{0, 1, 0, -1}
	dy = []int{-1, 0, 1, 0}
)

// Trail fade stages
var trailBlocks = []string{"\u2588", "\u2593", "\u2592", "\u2591"}

// drawBlock fills a grid cell with a block character at terminal position
func drawBlock(gx, gy int, ch string, color lipgloss.Color) {
	styled := lipgloss.NewStyle().Foreground(color).Render(ch)
	// Convert grid coords to terminal coords
	tx := gx*cellW + 1
	ty := gy*cellH + 1
	for row := 0; row < cellH; row++ {
		fmt.Printf("\033[%d;%dH", ty+row, tx)
		for col := 0; col < cellW; col++ {
			fmt.Print(styled)
		}
	}
}

// eraseBlock clears a grid cell
func eraseBlock(gx, gy int) {
	tx := gx*cellW + 1
	ty := gy*cellH + 1
	blank := strings.Repeat(" ", cellW)
	for row := 0; row < cellH; row++ {
		fmt.Printf("\033[%d;%dH%s", ty+row, tx, blank)
	}
}
