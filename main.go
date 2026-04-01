package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework LocalAuthentication -framework Foundation -lpam

#include <stdlib.h>
#include <LocalAuthentication/LocalAuthentication.h>
#include <security/pam_appl.h>
#include <pwd.h>
#include <unistd.h>

// Check if Touch ID is available
int touchid_available() {
    LAContext *context = [[LAContext alloc] init];
    NSError *error = nil;
    BOOL available = [context canEvaluatePolicy:LAPolicyDeviceOwnerAuthenticationWithBiometrics error:&error];
    return available ? 1 : 0;
}

// Touch ID authentication
int authenticate_touchid() {
    __block int result = 0;
    dispatch_semaphore_t sema = dispatch_semaphore_create(0);

    LAContext *context = [[LAContext alloc] init];
    NSError *error = nil;

    if ([context canEvaluatePolicy:LAPolicyDeviceOwnerAuthenticationWithBiometrics error:&error]) {
        [context evaluatePolicy:LAPolicyDeviceOwnerAuthenticationWithBiometrics
                localizedReason:@"tlock: unlock terminal"
                          reply:^(BOOL success, NSError *evalError) {
            if (success) {
                result = 1;
            }
            dispatch_semaphore_signal(sema);
        }];
        dispatch_semaphore_wait(sema, DISPATCH_TIME_FOREVER);
    }

    return result;
}

// PAM conversation function
static const char *pam_password = NULL;

int pam_conv_func(int num_msg, const struct pam_message **msg,
                  struct pam_response **resp, void *appdata_ptr) {
    struct pam_response *reply = calloc(num_msg, sizeof(struct pam_response));
    if (reply == NULL) return PAM_BUF_ERR;

    for (int i = 0; i < num_msg; i++) {
        if (msg[i]->msg_style == PAM_PROMPT_ECHO_OFF ||
            msg[i]->msg_style == PAM_PROMPT_ECHO_ON) {
            reply[i].resp = strdup(pam_password);
            reply[i].resp_retcode = 0;
        }
    }
    *resp = reply;
    return PAM_SUCCESS;
}

// Password authentication via PAM
int authenticate_password(const char *pw) {
    pam_password = pw;
    struct passwd *pwd = getpwuid(getuid());
    if (pwd == NULL) return 0;

    struct pam_conv conv = { pam_conv_func, NULL };
    pam_handle_t *pamh = NULL;

    int ret = pam_start("login", pwd->pw_name, &conv, &pamh);
    if (ret != PAM_SUCCESS) return 0;

    ret = pam_authenticate(pamh, 0);
    pam_end(pamh, ret);

    return (ret == PAM_SUCCESS) ? 1 : 0;
}
*/
import "C"

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
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
	var result []string
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

func renderLockScreen() {
	clearScreen()
}

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

// clearRect blanks out a padded rectangle in the center of the screen
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

func readPasswordOverlay(overlay bool) string {
	if !overlay {
		clearScreen()
	}

	prefix := subtitleStyle.Render("ENTER PASSPHRASE") + "\r\n"
	var pw []byte

	// Blinking cursor state
	cursorVisible := true
	stopBlink := make(chan struct{})
	blinkBlock := lipgloss.NewStyle().Foreground(teal).Render("\u2588")
	dimBlock := lipgloss.NewStyle().Foreground(gray).Render("\u2591")

	hint := subtitleStyle.Render("ESC: Touch ID")

	redrawPrompt := func() {
		w, h := getTermSize()

		content := prefix
		stars := ""
		for range pw {
			stars += dimBlock
		}
		var cursor string
		if cursorVisible {
			cursor = blinkBlock
		} else {
			cursor = " "
		}
		content += stars + cursor
		box := msgBoxStyle.Render(content)
		lines := strings.Split(box, "\n")

		// Calculate box width for clearing
		boxWidth := 0
		for _, line := range lines {
			lw := lipgloss.Width(line)
			if lw > boxWidth {
				boxWidth = lw
			}
		}

		// Clear just the area around the prompt (+ hint line below)
		clearRect(boxWidth, len(lines)+3, 3)

		startRow := (h - len(lines)) / 2
		fmt.Printf("\033[%d;0H", startRow)
		fmt.Printf("%s\r\n", centerBlock(box, w))
		fmt.Print("\r\n")
		fmt.Printf("%s", centerText(hint, w))
		drawLockIcon()
	}

	// Handle resize
	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)
	defer signal.Stop(sigwinch)

	// Start blinking and resize handling
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopBlink:
				return
			case <-sigwinch:
				redrawPrompt()
			case <-ticker.C:
				cursorVisible = !cursorVisible
				redrawPrompt()
			}
		}
	}()

	redrawPrompt()

	buf := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			continue
		}
		b := buf[0]
		switch {
		case b == 27: // Esc — switch to Touch ID
			close(stopBlink)
			return "\x1b"
		case b == 13 || b == 10: // Enter
			close(stopBlink)
			return string(pw)
		case b == 127 || b == 8: // Backspace
			if len(pw) > 0 {
				pw = pw[:len(pw)-1]
				cursorVisible = true
				redrawPrompt()
			}
		case b == 3: // Ctrl+C — ignore
			continue
		case b >= 32: // Printable
			pw = append(pw, b)
			cursorVisible = true
			redrawPrompt()
		}
	}
}

// --- xlock-style worm screensaver demo (grid-based) ---

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
	state   int
	wormIdx int
	trailAge int
	color   lipgloss.Color
}

type worm struct {
	body     []point // circular buffer of grid positions
	head     int     // index of head in body (circular buffer)
	length   int     // current drawn length (grows from 0 to cap)
	dir      int     // 0=up, 1=right, 2=down, 3=left
	color    lipgloss.Color
	turnCool int     // ticks before next turn allowed
	minRun   int     // minimum straight cells before turning
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
var dx = []int{0, 1, 0, -1}
var dy = []int{-1, 0, 1, 0}

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

func runWormDemo() bool {
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

	// Worm lengths (in grid cells) — curated mix
	wormLengths := []int{8, 12, 18, 10, 22, 14, 6, 16, 9, 20, 11, 13, 17}
	numWorms := len(wormLengths)
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
			clear := true
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
							clear = false
						}
						continue
					}
					if grid[cy][cx].state != cellEmpty {
						clear = false
						break
					}
				}
				if !clear {
					break
				}
			}
			if clear {
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
			grid[by][bx] = gridCell{state: cellBody, wormIdx: i, color: wormColors[i%len(wormColors)]}
		}

		worms[i] = worm{
			body:     body,
			head:     0,
			length:   bodyLen,
			dir:      dir,
			color:    wormColors[i%len(wormColors)],
			turnCool: rand.Intn(8) + 6,
			minRun:   rand.Intn(6) + 4,
		}
	}

	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()

	// Keypress channel
	keyCh := make(chan byte, 1)
	startKeyReader := func() {
		go func() {
			buf := make([]byte, 1)
			os.Stdin.Read(buf)
			keyCh <- buf[0]
		}()
	}
	startKeyReader()

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
		case key := <-keyCh:
			// Ignore non-printable (tmux focus events)
			if key < 32 && key != 13 && key != 10 {
				startKeyReader()
				continue
			}
			// Freeze worms — show password prompt
			pw := readPasswordOverlay(true)
			if handleAuth(pw) {
				return true // authenticated — unlock
			}
			fullRedraw()
			startKeyReader()
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

func main() {
	snake := flag.Bool("snake", false, "Screensaver on immediately (shortcut for --screensaver --screensaver-delay 0)")
	screensaver := flag.Bool("screensaver", false, "Enable xlock-style worm screensaver")
	screensaverDelay := flag.Int("screensaver-delay", 30, "Seconds idle before screensaver starts (0 = immediate)")
	flag.Parse()

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set raw mode: %v\n", err)
		os.Exit(1)
	}
	defer term.Restore(fd, oldState)

	hideCursor()
	defer showCursor()
	defer clearScreen()

	// Ignore signals that could bypass the lock
	signal.Ignore(syscall.SIGINT, syscall.SIGTERM, syscall.SIGTSTP)

	// --snake is a shortcut: screensaver on, delay 0
	if *snake {
		*screensaver = true
		*screensaverDelay = 0
	}

	for {
		if *screensaver && *screensaverDelay == 0 {
			// Screensaver immediately — worms run, keypress triggers auth
			if runWormDemo() {
				return
			}
		} else if *screensaver {
			// Show password prompt, start screensaver after delay if idle
			// Run password prompt with a timeout
			pwCh := make(chan string, 1)
			go func() {
				pwCh <- readPasswordOverlay(false)
			}()

			timer := time.NewTimer(time.Duration(*screensaverDelay) * time.Second)
			select {
			case pw := <-pwCh:
				timer.Stop()
				if handleAuth(pw) {
					return
				}
				continue
			case <-timer.C:
				// Timeout — switch to screensaver
				if runWormDemo() {
					return
				}
				continue
			}
		} else {
			// No screensaver — just password prompt
			pw := readPasswordOverlay(false)
			if handleAuth(pw) {
				return
			}
		}
	}
}

// handleAuth processes password/Touch ID input. Returns true if authenticated.
func handleAuth(pw string) bool {
	// Esc pressed — switch to Touch ID
	if pw == "\x1b" {
		if C.touchid_available() == 1 {
			if C.authenticate_touchid() == 1 {
				return true
			}
		}
		return false
	}

	if pw == "" {
		return false
	}

	// Verify password via PAM
	cpw := C.CString(pw)
	result := C.authenticate_password(cpw)
	C.free(unsafe.Pointer(cpw))

	if result == 1 {
		return true
	}

	renderMessage("ACCESS DENIED", errorStyle)
	time.Sleep(1 * time.Second)
	return false
}
