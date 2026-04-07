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
