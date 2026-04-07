package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type pipe struct {
	x, y  int
	dir   int
	color lipgloss.Color
	alive bool
}

type pipesScreensaver struct{}

func (p *pipesScreensaver) run(stopCh <-chan struct{}) bool {
	return runPipesDemo(stopCh)
}

func runPipesDemo(stopCh <-chan struct{}) bool {
	tw, th := getTermSize()
	clearScreen()

	gridW := tw / cellW
	gridH := th / cellH

	grid := make([][]gridCell, gridH)
	for y := range grid {
		grid[y] = make([]gridCell, gridW)
	}

	// Reserve lock icon area (top-left 3x3 grid cells)
	lockPad := 2
	for ly := 0; ly <= lockPad; ly++ {
		for lx := 0; lx <= lockPad; lx++ {
			if ly < gridH && lx < gridW {
				grid[ly][lx] = gridCell{state: cellLock}
			}
		}
	}

	gridArea := gridW * gridH

	spawnCount := func() int {
		switch {
		case gridArea < 800:
			return 4
		case gridArea < 1500:
			return 6
		default:
			return 8
		}
	}

	// Shuffle colors for variety
	shuffledColors := func() []lipgloss.Color {
		shuffled := make([]lipgloss.Color, len(wormColors))
		copy(shuffled, wormColors)
		rand.Shuffle(
			len(shuffled),
			func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] },
		)
		return shuffled
	}

	colorPool := shuffledColors()
	colorIdx := 0
	nextColor := func() lipgloss.Color {
		c := colorPool[colorIdx%len(colorPool)]
		colorIdx++
		return c
	}

	spawnPipe := func() pipe {
		// Find a random empty cell for spawn
		for attempts := 0; attempts < 200; attempts++ {
			x := rand.Intn(gridW)
			y := rand.Intn(gridH)
			if grid[y][x].state == cellEmpty {
				return pipe{
					x:     x,
					y:     y,
					dir:   rand.Intn(4),
					color: nextColor(),
					alive: true,
				}
			}
		}
		// Fallback: spawn anywhere (will be blocked immediately and handled)
		return pipe{
			x:     rand.Intn(gridW),
			y:     rand.Intn(gridH),
			dir:   rand.Intn(4),
			color: nextColor(),
			alive: true,
		}
	}

	n := spawnCount()
	pipes := make([]pipe, n)
	for i := range pipes {
		pipes[i] = spawnPipe()
	}

	// Count non-lock cells for fill ratio tracking
	nonLockCells := 0
	for gy := 0; gy < gridH; gy++ {
		for gx := 0; gx < gridW; gx++ {
			if grid[gy][gx].state != cellLock {
				nonLockCells++
			}
		}
	}

	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	keyCh := make(chan byte, 1)
	startKeyReader := func() {
		go func() {
			buf := make([]byte, 1)
			if _, err := os.Stdin.Read(buf); err == nil {
				keyCh <- buf[0]
			}
		}()
	}
	startKeyReader()

	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH, syscall.SIGCONT)
	defer signal.Stop(sigwinch)

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
		drawLockIcon()
	}

	// Count filled (body+trail) cells to detect ~75% full
	filledCount := func() int {
		count := 0
		for gy := 0; gy < gridH; gy++ {
			for gx := 0; gx < gridW; gx++ {
				s := grid[gy][gx].state
				if s == cellBody || s == cellTrail {
					count++
				}
			}
		}
		return count
	}

	// Fade out all body/trail cells through trailBlocks stages, then clear
	fadeOutAll := func() {
		// Convert all body cells to trail at age 0
		for gy := 0; gy < gridH; gy++ {
			for gx := 0; gx < gridW; gx++ {
				cell := &grid[gy][gx]
				if cell.state == cellBody {
					cell.state = cellTrail
					cell.trailAge = 0
				}
			}
		}
		// Step through trail ages until all empty
		for age := 0; age < len(trailBlocks); age++ {
			for gy := 0; gy < gridH; gy++ {
				for gx := 0; gx < gridW; gx++ {
					cell := &grid[gy][gx]
					if cell.state == cellTrail {
						if cell.trailAge < len(trailBlocks) {
							drawBlock(gx, gy, trailBlocks[cell.trailAge], cell.color)
						}
						cell.trailAge++
						if cell.trailAge >= len(trailBlocks) {
							cell.state = cellEmpty
							eraseBlock(gx, gy)
						}
					}
				}
			}
			drawLockIcon()
			time.Sleep(120 * time.Millisecond)
		}
	}

	// Initial draw of the lock icon
	drawLockIcon()

	for {
		select {
		case <-stopCh:
			return false
		case key := <-keyCh:
			if key < 32 && key != 13 && key != 10 {
				startKeyReader()
				continue
			}
			pw := readPasswordOverlay(true)
			if handleAuth(pw) {
				return true
			}
			fullRedraw()
			startKeyReader()
			continue

		case <-sigwinch:
			tw, th = getTermSize()
			gridW = tw / cellW
			gridH = th / cellH

			// Rebuild grid to new size
			newGrid := make([][]gridCell, gridH)
			for y := range newGrid {
				newGrid[y] = make([]gridCell, gridW)
			}
			// Copy existing grid state within new bounds
			for gy := 0; gy < gridH; gy++ {
				for gx := 0; gx < gridW; gx++ {
					if gy < len(grid) && gx < len(grid[gy]) {
						newGrid[gy][gx] = grid[gy][gx]
					}
				}
			}
			// Re-reserve lock icon area
			for ly := 0; ly <= lockPad; ly++ {
				for lx := 0; lx <= lockPad; lx++ {
					if ly < gridH && lx < gridW {
						newGrid[ly][lx] = gridCell{state: cellLock}
					}
				}
			}
			grid = newGrid

			// Recalculate non-lock cells
			nonLockCells = 0
			for gy := 0; gy < gridH; gy++ {
				for gx := 0; gx < gridW; gx++ {
					if grid[gy][gx].state != cellLock {
						nonLockCells++
					}
				}
			}

			// Kill pipes that are now out of bounds
			for i := range pipes {
				if pipes[i].x >= gridW || pipes[i].y >= gridH {
					pipes[i].alive = false
				}
			}

			fullRedraw()
			continue

		case <-ticker.C:
		}

		// Check ~75% fill — fade out and respawn
		if nonLockCells > 0 && filledCount()*100/nonLockCells >= 75 {
			fadeOutAll()
			// Reset grid (preserve lock area)
			for gy := 0; gy < gridH; gy++ {
				for gx := 0; gx < gridW; gx++ {
					if grid[gy][gx].state != cellLock {
						grid[gy][gx] = gridCell{}
					}
				}
			}
			colorPool = shuffledColors()
			colorIdx = 0
			n = spawnCount()
			pipes = make([]pipe, n)
			for i := range pipes {
				pipes[i] = spawnPipe()
			}
			drawLockIcon()
			continue
		}

		// Move each pipe
		for i := range pipes {
			p := &pipes[i]
			if !p.alive {
				// Respawn dead pipe
				*p = spawnPipe()
				// Draw spawn cell
				grid[p.y][p.x] = gridCell{state: cellBody, color: p.color}
				drawBlock(p.x, p.y, "\u2588", p.color)
				continue
			}

			// ~12% chance of a random 90° turn each tick
			if rand.Intn(100) < 12 {
				turn := 1
				if rand.Intn(2) == 0 {
					turn = -1
				}
				p.dir = (p.dir + turn + 4) % 4
			}

			// Next position (wrapping)
			nx := ((p.x + dx[p.dir]) + gridW) % gridW
			ny := ((p.y + dy[p.dir]) + gridH) % gridH

			// Check if next cell is available (empty only — pipes are solid)
			canMove := func(gx, gy int) bool {
				return grid[gy][gx].state == cellEmpty
			}

			if !canMove(nx, ny) {
				// Try turning left or right
				moved := false
				for _, turn := range []int{1, -1} {
					tryDir := (p.dir + turn + 4) % 4
					tx := ((p.x + dx[tryDir]) + gridW) % gridW
					ty := ((p.y + dy[tryDir]) + gridH) % gridH
					if canMove(tx, ty) {
						p.dir = tryDir
						nx = tx
						ny = ty
						moved = true
						break
					}
				}
				if !moved {
					// Blocked on all sides — kill and respawn next tick
					p.alive = false
					continue
				}
			}

			// Pipe cells stay solid — no trail conversion
			p.x = nx
			p.y = ny
			grid[ny][nx] = gridCell{state: cellBody, color: p.color}
			drawBlock(nx, ny, "\u2588", p.color)
		}

		drawLockIcon()
	}
}
