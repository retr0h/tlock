package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// lockW and lockH are the dimensions of the bouncing padlock icon.
const (
	lockW = 5
	lockH = 5
)

// buildLockLines constructs a padlock rendered in a single tint color.
func buildLockLines(color lipgloss.Color) []string {
	bright := lipgloss.NewStyle().Foreground(color)
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("236"))
	return []string{
		" " + bright.Render("\u2593\u2593\u2593") + " ",
		bright.Render("\u2593") + "   " + bright.Render("\u2593"),
		bright.Render("\u2588\u2588\u2588\u2588\u2588"),
		bright.Render("\u2588") + dim.Render(" \u2588 ") + bright.Render("\u2588"),
		bright.Render("\u2588\u2588\u2588\u2588\u2588"),
	}
}

// eraseLock overwrites the 5×5 lock area at terminal position (col, row) with spaces.
// col and row are 1-based terminal coordinates.
func eraseLock(col, row int) {
	blank := "     " // lockW spaces
	for r := 0; r < lockH; r++ {
		fmt.Printf("\033[%d;%dH%s", row+r, col, blank)
	}
}

// drawLockAt renders lock lines at terminal position (col, row), 1-based.
func drawLockAt(col, row int, lines []string) {
	for r, line := range lines {
		fmt.Printf("\033[%d;%dH%s", row+r, col, line)
	}
}

type dvdScreensaver struct{}

func (d *dvdScreensaver) run() bool {
	return runDVDDemo()
}

func runDVDDemo() bool {
	tw, th := getTermSize()
	clearScreen()
	hideCursor()

	// Pick a random starting color.
	color := wormColors[rand.Intn(len(wormColors))]
	lines := buildLockLines(color)

	// Compute movement bounds: position is the terminal column/row of the top-left
	// corner of the lock (1-based). The lock must stay within [1, tw-lockW+1] cols
	// and [1, th-lockH+1] rows.
	maxCol := tw - lockW + 1
	maxRow := th - lockH + 1
	if maxCol < 1 {
		maxCol = 1
	}
	if maxRow < 1 {
		maxRow = 1
	}

	col := rand.Intn(maxCol) + 1
	row := rand.Intn(maxRow) + 1
	vx := 1
	vy := 1

	// Draw static lock icon in top-left corner.
	drawLockIcon()

	// Initial draw of bouncing lock.
	drawLockAt(col, row, lines)

	ticker := time.NewTicker(100 * time.Millisecond)
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
		drawLockIcon()
		// Clamp position to new bounds.
		maxCol = tw - lockW + 1
		maxRow = th - lockH + 1
		if maxCol < 1 {
			maxCol = 1
		}
		if maxRow < 1 {
			maxRow = 1
		}
		if col > maxCol {
			col = maxCol
		}
		if row > maxRow {
			row = maxRow
		}
		drawLockAt(col, row, lines)
	}

	for {
		select {
		case key := <-keyCh:
			// Ignore non-printable keys (tmux focus events), except Enter.
			if key < 32 && key != 13 && key != 10 {
				startKeyReader()
				continue
			}
			pw := readPasswordOverlay(true)
			if handleAuth(pw) {
				showCursor()
				return true
			}
			fullRedraw()
			startKeyReader()
			continue

		case <-sigwinch:
			tw, th = getTermSize()
			fullRedraw()
			continue

		case <-ticker.C:
		}

		// Erase current position.
		eraseLock(col, row)

		// Move.
		col += vx
		row += vy

		// Bounce off walls, pick new color on each bounce.
		bounced := false
		if col < 1 {
			col = 1
			vx = 1
			bounced = true
		} else if col > maxCol {
			col = maxCol
			vx = -1
			bounced = true
		}
		if row < 1 {
			row = 1
			vy = 1
			bounced = true
		} else if row > maxRow {
			row = maxRow
			vy = -1
			bounced = true
		}

		if bounced {
			color = wormColors[rand.Intn(len(wormColors))]
			lines = buildLockLines(color)
		}

		// Draw at new position.
		drawLockAt(col, row, lines)

		// Redraw static lock icon (bouncing lock may have overlapped it).
		drawLockIcon()
	}
}
