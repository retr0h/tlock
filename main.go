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

func renderLockScreen() {
	w, h := getTermSize()
	clearScreen()

	title := lockTitleStyle.Render("LOCKED")
	hostname, _ := os.Hostname()
	host := subtitleStyle.Render(hostname)
	hint := subtitleStyle.Render("Press any key to unlock")

	mid := h / 2
	fmt.Printf("\033[%d;0H", mid-1)
	fmt.Printf("%s\r\n", centerText(title, w))
	fmt.Printf("%s\r\n", centerText(host, w))
	fmt.Print("\r\n")
	fmt.Printf("%s\r\n", centerText(hint, w))
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
	Foreground(teal).
	Bold(true)

var errBoxStyle = lipgloss.NewStyle().
	Border(glitchBorder).
	BorderForeground(red).
	Padding(1, 4).
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

func readPassword() string {
	w, h := getTermSize()
	clearScreen()

	prefix := "Password: "
	var pw []byte

	// Blinking cursor state
	cursorVisible := true
	stopBlink := make(chan struct{})
	blinkBlock := lipgloss.NewStyle().Foreground(teal).Render("\u2588")

	redrawPrompt := func() {
		clearScreen()
		stars := strings.Repeat("*", len(pw))
		var cursor string
		if cursorVisible {
			cursor = blinkBlock
		} else {
			cursor = " "
		}
		content := prefix + stars + cursor
		box := msgBoxStyle.Render(content)
		lines := strings.Split(box, "\n")
		startRow := (h - len(lines)) / 2
		fmt.Printf("\033[%d;0H", startRow)
		fmt.Printf("%s", centerBlock(box, w))
	}

	redrawPrompt()

	// Blink goroutine
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopBlink:
				return
			case <-ticker.C:
				cursorVisible = !cursorVisible
				redrawPrompt()
			}
		}
	}()

	buf := make([]byte, 1)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			continue
		}
		b := buf[0]
		switch {
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

func main() {
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

	// Handle resize
	sigwinch := make(chan os.Signal, 1)
	signal.Notify(sigwinch, syscall.SIGWINCH)

	for {
		renderLockScreen()

		// Wait for keypress or resize
		keyCh := make(chan byte, 1)
		go func() {
			b := make([]byte, 1)
			os.Stdin.Read(b)
			keyCh <- b[0]
		}()

	waitLoop:
		for {
			select {
			case <-sigwinch:
				renderLockScreen()
			case <-keyCh:
				break waitLoop
			}
		}

		// Try Touch ID if available (e.g. lid open, hardware present)
		if C.touchid_available() == 1 {
			renderMessage("Authenticating with Touch ID...", promptStyle)
			if C.authenticate_touchid() == 1 {
				return
			}
		}

		// Fall back to password
		pw := readPassword()
		cpw := C.CString(pw)
		result := C.authenticate_password(cpw)
		C.free(unsafe.Pointer(cpw))

		if result == 1 {
			return
		}

		renderMessage("Authentication failed", errorStyle)
		time.Sleep(1 * time.Second)
	}
}
