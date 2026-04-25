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
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type worm struct {
	body     []point // circular buffer of grid positions
	head     int     // index of head in body (circular buffer)
	length   int     // current drawn length (grows from 0 to cap)
	dir      int     // 0=up, 1=right, 2=down, 3=left
	color    lipgloss.Color
	turnCool int // ticks before next turn allowed
	minRun   int // minimum straight cells before turning
}

type wormScreensaver struct {
	numWorms int
}

func (ws *wormScreensaver) run(stopCh <-chan struct{}, keyCh <-chan byte) bool {
	return runWormDemo(ws.numWorms, stopCh, keyCh)
}

func runWormDemo(numWorms int, stopCh <-chan struct{}, keyCh <-chan byte) bool {
	tw, th := getTermSize()
	clearScreen()

	// Grid dimensions based on terminal size
	gridW := tw / cellW
	gridH := th / cellH

	// The grid — each cell knows its state
	grid := make([][]gridCell, gridH)
	for y := range grid {
		grid[y] = make([]gridCell, gridW)
	}

	// Reserve lock icon area (top-left, grid cell 0,0 + padding)
	lockPad := 2
	for ly := 0; ly <= lockPad; ly++ {
		for lx := 0; lx <= lockPad; lx++ {
			if ly < gridH && lx < gridW {
				grid[ly][lx] = gridCell{state: cellLock}
			}
		}
	}

	// Auto-scale worm count based on terminal area if not specified
	gridArea := gridW * gridH
	if numWorms <= 0 {
		switch {
		case gridArea < 800: // ~110x40 or smaller
			numWorms = 4
		case gridArea < 1500: // ~150x60
			numWorms = 7
		case gridArea < 3000: // ~200x80
			numWorms = 10
		default: // large terminals
			numWorms = 13
		}
	}

	// Generate curated worm lengths — varied but deliberate
	baseLengths := []int{8, 12, 18, 10, 22, 14, 6, 16, 9, 20, 11, 13, 17, 15, 19, 7, 24}
	wormLengths := make([]int, numWorms)
	for j := range wormLengths {
		wormLengths[j] = baseLengths[j%len(baseLengths)]
	}

	// Shuffle colors and assign — no two adjacent worms share a color
	shuffled := make([]lipgloss.Color, len(wormColors))
	copy(shuffled, wormColors)
	rand.Shuffle(
		len(shuffled),
		func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] },
	)
	assignedColors := make([]lipgloss.Color, numWorms)
	colorIdx := 0
	for j := range assignedColors {
		assignedColors[j] = shuffled[colorIdx%len(shuffled)]
		colorIdx++
		// If next worm would get same color as this one, skip ahead
		if j < numWorms-1 && colorIdx%len(shuffled) == (colorIdx-1)%len(shuffled) {
			colorIdx++
		}
	}

	worms := make([]worm, numWorms)

	// Spawn worms at non-overlapping grid positions
	for i := range worms {
		dir := rand.Intn(4)
		bodyLen := wormLengths[i]
		body := make([]point, bodyLen)

		// Find a clear spawn point
		var sx, sy int
		for attempts := 0; attempts < 500; attempts++ {
			sx = rand.Intn(gridW-4) + 2
			sy = rand.Intn(gridH-4) + 2
			open := true
			// Check the full body line + 1 cell padding around it
			for j := -1; j <= bodyLen; j++ {
				for pad := -1; pad <= 1; pad++ {
					cx := sx - dx[dir]*j
					cy := sy - dy[dir]*j
					// Add perpendicular padding
					if dir == 0 || dir == 2 {
						cx += pad
					} else {
						cy += pad
					}
					if cx < 0 || cx >= gridW || cy < 0 || cy >= gridH {
						if j >= 0 && j < bodyLen {
							open = false
						}
						continue
					}
					if grid[cy][cx].state != cellEmpty {
						open = false
						break
					}
				}
				if !open {
					break
				}
			}
			if open {
				break
			}
		}

		// Lay out body behind the head
		for j := range body {
			bx := sx - dx[dir]*j
			by := sy - dy[dir]*j
			// Wrap around grid edges
			bx = ((bx % gridW) + gridW) % gridW
			by = ((by % gridH) + gridH) % gridH
			body[j] = point{bx, by}
			grid[by][bx] = gridCell{state: cellBody, wormIdx: i, color: assignedColors[i]}
		}

		worms[i] = worm{
			body:     body,
			head:     0,
			length:   bodyLen,
			dir:      dir,
			color:    assignedColors[i],
			turnCool: rand.Intn(8) + 6,
			minRun:   rand.Intn(6) + 4,
		}
	}

	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()

	// Handle resize and pane refocus
	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH, syscall.SIGCONT)
	defer signal.Stop(sigwinch)

	drawLock := drawLockIcon

	// Full redraw from grid state
	fullRedraw := func() {
		clearScreen()
		for gy := 0; gy < gridH; gy++ {
			for gx := 0; gx < gridW; gx++ {
				cell := grid[gy][gx]
				switch cell.state {
				case cellBody:
					drawBlock(gx, gy, "\u2588", cell.color)
				case cellTrail:
					if cell.trailAge < len(trailBlocks) {
						drawBlock(gx, gy, trailBlocks[cell.trailAge], cell.color)
					}
				}
			}
		}
		drawLock()
	}

	// Initial draw
	fullRedraw()

	for {
		select {
		case <-stopCh:
			return false
		case key := <-keyCh:
			// Ignore non-printable (tmux focus events)
			if key < 32 && key != 13 && key != 10 {
				continue
			}
			// Freeze worms — show password prompt
			pw := readPasswordOverlay(true, keyCh)
			if handleAuth(pw) {
				return true // authenticated — unlock
			}
			fullRedraw()
			continue
		case <-sigwinch:
			tw, th = getTermSize()
			gridW = tw / cellW
			gridH = th / cellH
			fullRedraw()
			continue
		case <-ticker.C:
		}

		// Age trails
		for gy := 0; gy < gridH; gy++ {
			for gx := 0; gx < gridW; gx++ {
				cell := &grid[gy][gx]
				if cell.state == cellTrail {
					cell.trailAge++
					if cell.trailAge >= len(trailBlocks) {
						cell.state = cellEmpty
						eraseBlock(gx, gy)
					} else {
						drawBlock(gx, gy, trailBlocks[cell.trailAge], cell.color)
					}
				}
			}
		}

		// Move each worm
		for i := range worms {
			wm := &worms[i]

			// Turn logic — only after minimum straight run
			if wm.turnCool > 0 {
				wm.turnCool--
			} else {
				turn := 1
				if rand.Intn(2) == 0 {
					turn = -1
				}
				wm.dir = (wm.dir + turn + 4) % 4
				wm.turnCool = rand.Intn(10) + wm.minRun
			}

			// Calculate new head position (wrap around)
			head := wm.body[wm.head]
			newX := ((head.x + dx[wm.dir]) + gridW) % gridW
			newY := ((head.y + dy[wm.dir]) + gridH) % gridH

			// Check if blocked — cell + 1 block padding around it
			// A cell is clear only if it and all adjacent cells have no other worm's body
			isClear := func(gx, gy, selfIdx int) bool {
				for py := -1; py <= 1; py++ {
					for px := -1; px <= 1; px++ {
						cx := ((gx + px) + gridW) % gridW
						cy := ((gy + py) + gridH) % gridH
						cell := grid[cy][cx]
						if cell.state == cellLock {
							return false
						}
						if cell.state == cellBody && cell.wormIdx != selfIdx {
							return false
						}
						if cell.state == cellTrail {
							return false
						}
					}
				}
				return true
			}

			blocked := !isClear(newX, newY, i)

			if blocked {
				// Try left, right, reverse
				tried := false
				for _, turn := range []int{1, -1, 2} {
					tryDir := (wm.dir + turn + 4) % 4
					tryX := ((head.x + dx[tryDir]) + gridW) % gridW
					tryY := ((head.y + dy[tryDir]) + gridH) % gridH
					if isClear(tryX, tryY, i) {
						wm.dir = tryDir
						newX = tryX
						newY = tryY
						wm.turnCool = rand.Intn(6) + wm.minRun
						tried = true
						break
					}
				}
				if !tried {
					continue // stuck, skip this tick
				}
			}

			// Move tail → trail
			tailIdx := (wm.head + wm.length - 1) % len(wm.body)
			tail := wm.body[tailIdx]
			grid[tail.y][tail.x] = gridCell{state: cellTrail, color: wm.color, trailAge: 0}
			drawBlock(tail.x, tail.y, trailBlocks[0], wm.color)

			// Advance head (circular buffer — overwrite oldest)
			wm.head = (wm.head - 1 + len(wm.body)) % len(wm.body)
			wm.body[wm.head] = point{newX, newY}
			grid[newY][newX] = gridCell{state: cellBody, wormIdx: i, color: wm.color}

			// Draw worm body with fade
			for j := 0; j < wm.length; j++ {
				idx := (wm.head + j) % len(wm.body)
				p := wm.body[idx]
				fadeIdx := j * len(trailBlocks) / wm.length
				if fadeIdx >= len(trailBlocks) {
					fadeIdx = len(trailBlocks) - 1
				}
				var ch string
				if j == 0 {
					ch = "\u2588" // head always solid
				} else {
					ch = trailBlocks[fadeIdx]
				}
				drawBlock(p.x, p.y, ch, wm.color)
			}
		}

		drawLock()
	}
}
