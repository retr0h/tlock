# tlock — Terminal Lock with Touch ID

## Overview

A Go terminal lock program that clears the screen, displays a minimal lock
message styled with lipgloss (purple/teal palette matching osapi), and requires
macOS Touch ID or password authentication to unlock. Designed to be used as a
tmux lock-command replacement.

## Phase 1: Core Lock & Auth (this spec)

### Lock Screen

- Clear terminal completely on start
- Display centered minimal lock message:
  - "Locked" in bold purple
  - Hostname or username in dim gray below
  - "Press any key to unlock" in dim gray at bottom
- Terminal set to raw mode to capture keypresses without echo
- Hide cursor while locked

### Unlock Flow

1. Program starts, clears screen, shows lock message
2. Waits for any keypress
3. On keypress: shows centered "Authenticating..." prompt
4. Attempts Touch ID via macOS `LocalAuthentication.framework` (CGo)
   - Prompt text: "tlock: unlock terminal"
5. If Touch ID succeeds: clear screen, restore terminal, exit 0
6. If Touch ID fails or unavailable: fall back to password prompt
   - Show "Password:" prompt styled in teal
   - User types macOS account password (input hidden)
   - Verify via PAM (`pam_authenticate`) using CGo
7. If password succeeds: clear screen, restore terminal, exit 0
8. If password fails: show brief red error "Authentication failed", pause 1s,
   return to lock screen

### Terminal Handling

- Enter raw mode on start (no echo, no line buffering)
- Capture terminal size for centering
- Restore terminal state on exit (defer cleanup)
- Handle SIGINT/SIGTERM gracefully — do NOT unlock on signal, stay locked
- Handle terminal resize (SIGWINCH) to re-center display

### Dependencies

- `charmbracelet/lipgloss` v1.x — terminal styling
- `golang.org/x/term` — raw mode, terminal size
- CGo: `LocalAuthentication.framework` — Touch ID
- CGo: PAM (`pam_authenticate`) — password verification

### Color Palette

```
Purple    = lipgloss.Color("99")       // Headers, labels, borders
Teal      = lipgloss.Color("#06ffa5")  // Accent, values, prompts
Gray      = lipgloss.Color("245")      // Dim/secondary text
White     = lipgloss.Color("15")       // Primary text
```

### Project Structure

```
locker/
  main.go          # Entry point, terminal setup, main loop
  auth_darwin.go   # Touch ID + PAM auth (macOS-specific, CGo)
  ui.go            # lipgloss styles, screen rendering, centering
  go.mod
  go.sum
```

### Usage

```bash
# Direct
tlock

# As tmux lock-command
set -g lock-command "tlock"
set -g lock-after-time 1800
bind ^X lock-server
```

### Signal Handling

- SIGINT (Ctrl+C): ignored — must authenticate to exit
- SIGTERM: ignored — must authenticate to exit
- SIGWINCH: re-render centered display
- SIGTSTP (Ctrl+Z): ignored — prevent backgrounding

## Phase 2: Screensaver (future)

- xlock-style worm screensaver with fading trails
- Multiple worms using purple/teal palette
- Any keypress pauses worms, shows unlock prompt
- Auth failure resumes worms
- Configurable worm count, speed, colors

## Phase 3: Configuration (future)

- Config file at `~/.config/tlock/config.yaml`
- Screensaver mode selection
- Custom lock message
- Auth method preference
