# New Screensavers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add two new screensavers (bouncing DVD logo, pipes) and unify CLI flags under `--screensaver <name>`.

**Architecture:** Extract shared code (terminal utils, grid, auth, styles) from the monolithic `main.go` into separate files. Each screensaver becomes its own file implementing a common `screensaver` interface. Main dispatches by name.

**Tech Stack:** Go, CGo, lipgloss, golang.org/x/term

---

### File Structure

| File | Responsibility |
|------|---------------|
| `main.go` | Entry point, flag parsing, terminal setup, screensaver dispatch (trimmed from 877 → ~120 lines) |
| `terminal.go` | clearScreen, hideCursor, showCursor, centerText, centerBlock, getTermSize, drawLockIcon, clearRect |
| `auth.go` | handleAuth, readPasswordOverlay, CGo block (Touch ID + PAM) |
| `style.go` | Color vars, lipgloss styles, glitchBorder, msgBoxStyle, errBoxStyle, renderMessage, renderMessageOverlay |
| `grid.go` | cellW/cellH constants, point, gridCell, cellEmpty/cellBody/cellTrail/cellLock, drawBlock, eraseBlock, trailBlocks, wormColors, dx/dy |
| `screensaver.go` | `screensaver` interface definition, screensaver registry, random/cycle dispatch logic |
| `screensaver_worm.go` | worm struct, runWormDemo → implements screensaver interface |
| `screensaver_dvd.go` | Bouncing DVD logo screensaver |
| `screensaver_pipes.go` | Pipes screensaver |

---

### Task 1: Extract shared code from main.go

**Files:**
- Create: `terminal.go`, `auth.go`, `style.go`, `grid.go`
- Modify: `main.go`

- [ ] **Step 1: Create `terminal.go`**

```go
package main

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
```

- [ ] **Step 2: Create `style.go`**

```go
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
```

- [ ] **Step 3: Create `auth.go`**

Move the CGo block (lines 5-83), the `import "C"` line, and the `handleAuth` + `readPasswordOverlay` functions. The CGo comment block MUST be directly above `import "C"` with no blank line.

```go
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
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/charmbracelet/lipgloss"
)

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
```

- [ ] **Step 4: Create `grid.go`**

```go
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
```

- [ ] **Step 5: Trim `main.go`**

Replace `main.go` entirely with just the entry point. Remove all extracted code. Keep only `package main`, imports, `renderLockScreen`, and `main()`.

```go
// Package main implements tlock, a terminal lock screen for macOS with
// Touch ID and password authentication.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/term"
)

func renderLockScreen() {
	clearScreen()
}

func main() {
	snake := flag.Bool(
		"snake",
		false,
		"Screensaver on immediately (shortcut for --screensaver snake)",
	)
	snakeCount := flag.Int("snake-count", 0, "Number of worms (0 = auto based on terminal size)")
	screensaver := flag.Bool("screensaver", false, "Enable xlock-style worm screensaver")
	screensaverDelay := flag.Int(
		"screensaver-delay",
		30,
		"Seconds idle before screensaver starts (0 = immediate)",
	)
	flag.Parse()

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set raw mode: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = term.Restore(fd, oldState) }()

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
			if runWormDemo(*snakeCount) {
				return
			}
		} else if *screensaver {
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
				if runWormDemo(*snakeCount) {
					return
				}
				continue
			}
		} else {
			pw := readPasswordOverlay(false)
			if handleAuth(pw) {
				return
			}
		}
	}
}
```

- [ ] **Step 6: Build and verify**

Run: `cd /Users/john/git/tlock && go build -o tlock .`
Expected: Compiles with no errors. Identical behavior to before.

- [ ] **Step 7: Commit**

```bash
git add terminal.go auth.go style.go grid.go main.go
git commit -m "refactor: extract shared code from main.go into focused files"
```

---

### Task 2: Extract worm screensaver into its own file

**Files:**
- Create: `screensaver_worm.go`
- Modify: `main.go` (remove `runWormDemo`)

- [ ] **Step 1: Create `screensaver_worm.go`**

Move the `worm` struct and `runWormDemo` function from `main.go` (they should still be there from the trimmed version — actually they were left in the Step 5 main.go above by mistake, but in reality the worm code is still in `main.go` after step 5 since we only showed the entry point code; in practice, move ALL remaining worm code to this file):

```go
package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
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

func runWormDemo(numWorms int) bool {
	// ... exact existing code from main.go lines 458-771 ...
}
```

Copy the `runWormDemo` function body exactly as-is from the current `main.go` (lines 458-771). Add `"github.com/charmbracelet/lipgloss"` to the import if needed for `lipgloss.Color`.

- [ ] **Step 2: Remove worm code from `main.go`**

Delete the `worm` struct and `runWormDemo` function from `main.go`, leaving only the entry point.

- [ ] **Step 3: Build and verify**

Run: `cd /Users/john/git/tlock && go build -o tlock .`
Expected: Compiles clean.

- [ ] **Step 4: Commit**

```bash
git add screensaver_worm.go main.go
git commit -m "refactor: extract worm screensaver to screensaver_worm.go"
```

---

### Task 3: Add screensaver interface and unified CLI flags

**Files:**
- Create: `screensaver.go`
- Modify: `main.go`

- [ ] **Step 1: Create `screensaver.go`**

```go
package main

import "math/rand"

// screensaver defines the interface all screensavers implement.
// run() owns the animation loop and auth flow.
// Returns true if the user authenticated successfully.
type screensaver interface {
	run(gridW, gridH int) bool
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
```

- [ ] **Step 2: Adapt worm screensaver to interface**

In `screensaver_worm.go`, wrap `runWormDemo` in a struct:

```go
type wormScreensaver struct {
	numWorms int
}

func (ws *wormScreensaver) run(gridW, gridH int) bool {
	return runWormDemo(ws.numWorms)
}
```

- [ ] **Step 3: Add stub types for dvd and pipes** (so it compiles)

Append to `screensaver.go`:

```go
type dvdScreensaver struct{}

func (d *dvdScreensaver) run(gridW, gridH int) bool {
	// TODO: implement in Task 5
	return false
}

type pipesScreensaver struct{}

func (p *pipesScreensaver) run(gridW, gridH int) bool {
	// TODO: implement in Task 6
	return false
}
```

- [ ] **Step 4: Rewrite `main.go` with unified flags**

```go
// Package main implements tlock, a terminal lock screen for macOS with
// Touch ID and password authentication.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/term"
)

func renderLockScreen() {
	clearScreen()
}

func main() {
	// Legacy alias
	snake := flag.Bool("snake", false, "Alias for --screensaver snake (immediate)")
	snakeCount := flag.Int("snake-count", 0, "Number of worms (0 = auto)")
	wormCount := flag.Int("worm-count", 0, "Number of worms (0 = auto)")

	screensaverName := flag.String("screensaver", "", "Screensaver mode: snake, pipes, dvd, random")
	screensaverDelay := flag.Int("screensaver-delay", 30, "Seconds idle before screensaver starts (0 = immediate)")
	screensaverCycle := flag.Int("screensaver-cycle", 0, "Minutes between screensaver rotation (requires random)")
	flag.Parse()

	// --snake is alias for --screensaver snake --screensaver-delay 0
	if *snake {
		*screensaverName = "snake"
		*screensaverDelay = 0
	}

	// --snake-count / --worm-count
	numWorms := *snakeCount
	if *wormCount > 0 {
		numWorms = *wormCount
	}

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set raw mode: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	hideCursor()
	defer showCursor()
	defer clearScreen()

	signal.Ignore(syscall.SIGINT, syscall.SIGTERM, syscall.SIGTSTP)

	runSelected := func() bool {
		name := *screensaverName
		if name == "random" {
			name = pickRandomScreensaver("")
		}

		ss := screensaverFactory[name]()
		// Pass worm count config for snake screensaver
		if ws, ok := ss.(*wormScreensaver); ok {
			ws.numWorms = numWorms
		}

		tw, th := getTermSize()
		gridW := tw / cellW
		gridH := th / cellH
		return ss.run(gridW, gridH)
	}

	// Cycle mode: rotate screensavers every N minutes
	runWithCycle := func() bool {
		cycleDur := time.Duration(*screensaverCycle) * time.Minute
		current := ""
		for {
			name := pickRandomScreensaver(current)
			current = name

			ss := screensaverFactory[name]()
			if ws, ok := ss.(*wormScreensaver); ok {
				ws.numWorms = numWorms
			}

			tw, th := getTermSize()
			gridW := tw / cellW
			gridH := th / cellH

			// Run screensaver with a timer
			done := make(chan bool, 1)
			go func() {
				done <- ss.run(gridW, gridH)
			}()

			timer := time.NewTimer(cycleDur)
			select {
			case authenticated := <-done:
				timer.Stop()
				if authenticated {
					return true
				}
				// Auth failed, screensaver returned — pick next
			case <-timer.C:
				// Time to rotate — the screensaver is still running
				// We need a way to stop it; for now, continue (will be refined)
				continue
			}
		}
	}

	for {
		if *screensaverName != "" && *screensaverDelay == 0 {
			var authenticated bool
			if *screensaverName == "random" && *screensaverCycle > 0 {
				authenticated = runWithCycle()
			} else {
				authenticated = runSelected()
			}
			if authenticated {
				return
			}
		} else if *screensaverName != "" {
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
				var authenticated bool
				if *screensaverName == "random" && *screensaverCycle > 0 {
					authenticated = runWithCycle()
				} else {
					authenticated = runSelected()
				}
				if authenticated {
					return
				}
				continue
			}
		} else {
			pw := readPasswordOverlay(false)
			if handleAuth(pw) {
				return
			}
		}
	}
}
```

- [ ] **Step 5: Build and verify**

Run: `cd /Users/john/git/tlock && go build -o tlock .`
Expected: Compiles. `tlock --screensaver snake` works same as `tlock --snake`.

- [ ] **Step 6: Commit**

```bash
git add screensaver.go screensaver_worm.go main.go
git commit -m "feat: add screensaver interface and unified --screensaver flag"
```

---

### Task 4: Implement bouncing DVD logo screensaver

**Files:**
- Modify: `screensaver_dvd.go` (replace stub in `screensaver.go` — actually, create this as its own file and remove stub)
- Modify: `screensaver.go` (remove dvd stub)

- [ ] **Step 1: Create `screensaver_dvd.go`**

```go
package main

import (
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// dvd logo as block characters — 3 lines of glitch-bordered "tlock"
var dvdLogo = []string{
	"\u2591\u2592\u2593\u2588\u2588\u2588\u2588\u2588\u2588\u2593\u2592\u2591",
	"\u2593 t l o c k \u2593",
	"\u2591\u2592\u2593\u2588\u2588\u2588\u2588\u2588\u2588\u2593\u2592\u2591",
}

// dvdLogoWidth is the character width of the logo
const dvdLogoWidth = 12

// dvdLogoHeight is the line count of the logo
const dvdLogoHeight = 3

type dvdScreensaver struct{}

func (d *dvdScreensaver) run(gridW, gridH int) bool {
	tw, th := getTermSize()
	clearScreen()

	// Logo position in terminal chars (not grid cells)
	x := rand.Intn(tw - dvdLogoWidth - 2) + 1
	y := rand.Intn(th - dvdLogoHeight - 2) + 1
	vx := 1
	vy := 1

	color := wormColors[rand.Intn(len(wormColors))]

	drawDVD := func(px, py int, c lipgloss.Color) {
		style := lipgloss.NewStyle().Foreground(c)
		for i, line := range dvdLogo {
			fmt.Printf("\033[%d;%dH%s", py+i, px, style.Render(line))
		}
	}

	eraseDVD := func(px, py int) {
		blank := "            " // dvdLogoWidth spaces
		for i := range dvdLogo {
			fmt.Printf("\033[%d;%dH%s", py+i, px, blank)
		}
	}

	drawLockIcon()
	drawDVD(x, y, color)

	ticker := time.NewTicker(120 * time.Millisecond)
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

	for {
		select {
		case key := <-keyCh:
			if key < 32 && key != 13 && key != 10 {
				startKeyReader()
				continue
			}
			pw := readPasswordOverlay(true)
			if handleAuth(pw) {
				return true
			}
			// Redraw after failed auth
			tw, th = getTermSize()
			clearScreen()
			drawLockIcon()
			drawDVD(x, y, color)
			startKeyReader()
			continue
		case <-sigwinch:
			tw, th = getTermSize()
			clearScreen()
			drawLockIcon()
			// Clamp position to new size
			if x+dvdLogoWidth >= tw {
				x = tw - dvdLogoWidth - 1
			}
			if y+dvdLogoHeight >= th {
				y = th - dvdLogoHeight - 1
			}
			drawDVD(x, y, color)
			continue
		case <-ticker.C:
		}

		// Erase old position
		eraseDVD(x, y)

		// Move
		x += vx
		y += vy

		// Bounce off walls, change color on hit
		bounced := false
		if x <= 1 {
			x = 1
			vx = 1
			bounced = true
		} else if x+dvdLogoWidth >= tw {
			x = tw - dvdLogoWidth - 1
			vx = -1
			bounced = true
		}
		if y <= 1 {
			y = 1
			vy = 1
			bounced = true
		} else if y+dvdLogoHeight >= th {
			y = th - dvdLogoHeight - 1
			vy = -1
			bounced = true
		}

		if bounced {
			color = wormColors[rand.Intn(len(wormColors))]
		}

		drawDVD(x, y, color)
		drawLockIcon()
	}
}
```

- [ ] **Step 2: Remove dvd stub from `screensaver.go`**

Delete the `dvdScreensaver` struct and method stub from `screensaver.go`.

- [ ] **Step 3: Build and verify**

Run: `cd /Users/john/git/tlock && go build -o tlock .`
Expected: Compiles. `tlock --screensaver dvd` shows bouncing logo.

- [ ] **Step 4: Manual test**

Run: `./tlock --screensaver dvd`
Verify:
- Logo bounces off all 4 walls
- Color changes on each bounce
- Keypress shows auth overlay
- Lock icon stays in top-left
- Resize redraws cleanly

- [ ] **Step 5: Commit**

```bash
git add screensaver_dvd.go screensaver.go
git commit -m "feat: add bouncing DVD logo screensaver"
```

---

### Task 5: Implement pipes screensaver

**Files:**
- Create: `screensaver_pipes.go` (replace stub)
- Modify: `screensaver.go` (remove pipes stub)

- [ ] **Step 1: Create `screensaver_pipes.go`**

```go
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
	x, y  int // current grid position
	dir   int // 0=up, 1=right, 2=down, 3=left
	color lipgloss.Color
	alive bool
}

type pipesScreensaver struct {
	fadeOut bool // true = fade out when full, false = hard reset
}

func (p *pipesScreensaver) run(gridW, gridH int) bool {
	tw, th := getTermSize()
	clearScreen()

	grid := make([][]gridCell, gridH)
	for y := range grid {
		grid[y] = make([]gridCell, gridW)
	}

	// Reserve lock icon area
	lockPad := 2
	for ly := 0; ly <= lockPad; ly++ {
		for lx := 0; lx <= lockPad; lx++ {
			if ly < gridH && lx < gridW {
				grid[ly][lx] = gridCell{state: cellLock}
			}
		}
	}

	drawLockIcon()

	// Spawn initial pipes
	maxPipes := 4
	if gridW*gridH > 1500 {
		maxPipes = 6
	}

	pipes := make([]pipe, 0, maxPipes)
	spawnPipe := func() pipe {
		for attempts := 0; attempts < 100; attempts++ {
			sx := rand.Intn(gridW)
			sy := rand.Intn(gridH)
			if grid[sy][sx].state == cellEmpty {
				color := wormColors[rand.Intn(len(wormColors))]
				dir := rand.Intn(4)
				grid[sy][sx] = gridCell{state: cellBody, color: color}
				drawBlock(sx, sy, "\u2588", color)
				return pipe{x: sx, y: sy, dir: dir, color: color, alive: true}
			}
		}
		return pipe{alive: false}
	}

	// Start with a few pipes
	for i := 0; i < maxPipes; i++ {
		pipes = append(pipes, spawnPipe())
	}

	ticker := time.NewTicker(120 * time.Millisecond)
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

	filledCells := 0
	totalCells := gridW * gridH

	// Count how full the grid is
	countFilled := func() int {
		count := 0
		for gy := 0; gy < gridH; gy++ {
			for gx := 0; gx < gridW; gx++ {
				if grid[gy][gx].state != cellEmpty {
					count++
				}
			}
		}
		return count
	}
	filledCells = countFilled()

	// Reset the grid — either hard or fade
	hardReset := func() {
		for gy := 0; gy < gridH; gy++ {
			for gx := 0; gx < gridW; gx++ {
				if grid[gy][gx].state != cellLock {
					grid[gy][gx] = gridCell{}
				}
			}
		}
		clearScreen()
		drawLockIcon()
		pipes = pipes[:0]
		for i := 0; i < maxPipes; i++ {
			pipes = append(pipes, spawnPipe())
		}
		filledCells = countFilled()
	}

	// Fade state
	fading := false
	fadeAge := 0

	fadeStep := func() bool {
		allGone := true
		for gy := 0; gy < gridH; gy++ {
			for gx := 0; gx < gridW; gx++ {
				cell := &grid[gy][gx]
				if cell.state == cellBody || cell.state == cellTrail {
					cell.state = cellTrail
					cell.trailAge++
					if cell.trailAge >= len(trailBlocks) {
						cell.state = cellEmpty
						eraseBlock(gx, gy)
					} else {
						drawBlock(gx, gy, trailBlocks[cell.trailAge], cell.color)
						allGone = false
					}
				}
			}
		}
		return allGone
	}

	fullRedraw := func() {
		tw, th = getTermSize()
		gridW = tw / cellW
		gridH = th / cellH
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

	for {
		select {
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
			fullRedraw()
			continue
		case <-ticker.C:
		}

		// If fading, run fade steps
		if fading {
			if fadeStep() {
				fading = false
				fadeAge = 0
				// Respawn pipes
				pipes = pipes[:0]
				for i := 0; i < maxPipes; i++ {
					pipes = append(pipes, spawnPipe())
				}
				filledCells = countFilled()
			}
			drawLockIcon()
			continue
		}

		// Check if screen is mostly full — trigger reset
		if filledCells > totalCells*80/100 {
			if p.fadeOut {
				fading = true
				fadeAge = 0
				// Kill all pipes
				for i := range pipes {
					pipes[i].alive = false
				}
			} else {
				hardReset()
			}
			continue
		}

		// Grow each pipe
		for i := range pipes {
			pp := &pipes[i]
			if !pp.alive {
				continue
			}

			// Random turn: 15% chance each tick
			if rand.Intn(100) < 15 {
				turn := 1
				if rand.Intn(2) == 0 {
					turn = -1
				}
				pp.dir = (pp.dir + turn + 4) % 4
			}

			// Next position
			nx := pp.x + dx[pp.dir]
			ny := pp.y + dy[pp.dir]

			// Wrap around
			nx = ((nx % gridW) + gridW) % gridW
			ny = ((ny % gridH) + gridH) % gridH

			// If blocked, try turning
			if grid[ny][nx].state != cellEmpty {
				turned := false
				for _, t := range []int{1, -1} {
					tryDir := (pp.dir + t + 4) % 4
					tx := ((pp.x + dx[tryDir]) + gridW) % gridW
					ty := ((pp.y + dy[tryDir]) + gridH) % gridH
					if grid[ty][tx].state == cellEmpty {
						pp.dir = tryDir
						nx = tx
						ny = ty
						turned = true
						break
					}
				}
				if !turned {
					pp.alive = false
					// Spawn replacement
					pipes = append(pipes, spawnPipe())
					continue
				}
			}

			pp.x = nx
			pp.y = ny
			grid[ny][nx] = gridCell{state: cellBody, color: pp.color}
			drawBlock(nx, ny, "\u2588", pp.color)
			filledCells++
		}

		drawLockIcon()
	}
}
```

- [ ] **Step 2: Remove pipes stub from `screensaver.go`**

Delete the `pipesScreensaver` struct and method stub from `screensaver.go`.

- [ ] **Step 3: Build and verify**

Run: `cd /Users/john/git/tlock && go build -o tlock .`
Expected: Compiles.

- [ ] **Step 4: Manual test both reset modes**

Test hard reset:
```go
// Temporarily in screensaver.go factory:
"pipes": func() screensaver { return &pipesScreensaver{fadeOut: false} },
```
Run: `./tlock --screensaver pipes` — verify pipes fill screen then hard reset.

Test fade out:
```go
"pipes": func() screensaver { return &pipesScreensaver{fadeOut: true} },
```
Run: `./tlock --screensaver pipes` — verify pipes fade through ░▒▓ stages then restart.

User picks which one to keep after seeing both.

- [ ] **Step 5: Commit**

```bash
git add screensaver_pipes.go screensaver.go
git commit -m "feat: add pipes screensaver"
```

---

### Task 6: Add ghost trail variant to DVD (for comparison)

**Files:**
- Modify: `screensaver_dvd.go`

- [ ] **Step 1: Add trail option to dvdScreensaver**

Add a `ghostTrail` bool field and trail rendering logic:

```go
type dvdScreensaver struct {
	ghostTrail bool
}
```

In the `run()` method, after `eraseDVD(x, y)` and before drawing at the new position, if `ghostTrail` is true, draw a faded copy at the old position instead of erasing:

```go
		// Erase or leave ghost trail at old position
		if d.ghostTrail {
			// Draw faded ghost at old position
			ghostStyle := lipgloss.NewStyle().Foreground(color)
			for i, line := range dvdLogo {
				// Replace solid blocks with faded ones
				faded := ""
				for _, ch := range line {
					switch ch {
					case '\u2588':
						faded += "\u2591"
					case '\u2593':
						faded += "\u2591"
					case '\u2592':
						faded += "\u2591"
					default:
						faded += string(ch)
					}
				}
				fmt.Printf("\033[%d;%dH%s", y+i, x, ghostStyle.Render(faded))
			}
			// Erase ghost after 2 ticks via a goroutine with timer
			oldX, oldY, oldColor := x, y, color
			go func() {
				time.Sleep(360 * time.Millisecond) // ~3 ticks
				blankLine := "            " // dvdLogoWidth spaces
				_ = oldColor
				for i := range dvdLogo {
					fmt.Printf("\033[%d;%dH%s", oldY+i, oldX, blankLine)
				}
			}()
		} else {
			eraseDVD(x, y)
		}
```

- [ ] **Step 2: Build and verify**

Run: `cd /Users/john/git/tlock && go build -o tlock .`

- [ ] **Step 3: Manual test**

Temporarily set `ghostTrail: true` in the factory, run `./tlock --screensaver dvd`, compare with `ghostTrail: false`. User picks which to keep.

- [ ] **Step 4: Commit**

```bash
git add screensaver_dvd.go
git commit -m "feat: add ghost trail option to DVD screensaver for comparison"
```

---

### Task 7: Wire up --screensaver-cycle and random

This is already implemented in Task 3's `main.go` rewrite. This task is for testing and refinement.

**Files:**
- Modify: `main.go` (if needed)

- [ ] **Step 1: Test random selection**

Run: `./tlock --screensaver random`
Verify: Picks one of snake/pipes/dvd at random each launch.

- [ ] **Step 2: Test cycle mode**

Run: `./tlock --screensaver random --screensaver-cycle 1`
Verify: Switches screensaver every 1 minute. Note: cycling requires a stop mechanism — the current screensaver needs to be interruptible. If this doesn't work cleanly, add a `stopCh chan struct{}` to the screensaver interface or use a context.

If cycling needs a stop channel, update the interface:

```go
type screensaver interface {
	run(gridW, gridH int, stopCh <-chan struct{}) bool
}
```

And pass it through each screensaver's select loop:

```go
case <-stopCh:
    return false
```

- [ ] **Step 3: Commit if changes needed**

```bash
git add -A
git commit -m "feat: wire up screensaver cycling with stop channel"
```

---

### Task 8: User visual review and cleanup

- [ ] **Step 1: Demo all screensavers for user**

Run each and let user compare:
```bash
./tlock --screensaver snake
./tlock --screensaver dvd
./tlock --screensaver pipes
./tlock --screensaver random
```

- [ ] **Step 2: User picks DVD trail mode**

Show both `ghostTrail: true` and `ghostTrail: false`. Remove the unchosen variant and the bool field.

- [ ] **Step 3: User picks pipes reset mode**

Show both `fadeOut: true` and `fadeOut: false`. Remove the unchosen variant and the bool field.

- [ ] **Step 4: Final cleanup commit**

```bash
git add -A
git commit -m "feat: finalize screensaver options based on user review"
```

- [ ] **Step 5: Lint**

Run: `golangci-lint run`
Fix any issues.

- [ ] **Step 6: Final commit if lint fixes needed**

```bash
git add -A
git commit -m "fix: resolve lint issues"
```
