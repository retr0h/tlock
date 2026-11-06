# Development Guide

## Prerequisites

- macOS (required — uses LocalAuthentication.framework and PAM)
- [Go](https://go.dev/dl/) 1.21+ with CGo enabled
- [just](https://github.com/casey/just) — command runner
- [golangci-lint](https://golangci-lint.run/) — Go linter
- Touch ID hardware (optional — password fallback works without it)

## Getting Started

```bash
git clone https://github.com/retr0h/tlock.git
cd tlock
just fetch    # Fetch shared justfiles
just deps     # Install tool dependencies
```

## Common Commands

```bash
just deps          # Install all dependencies
just test          # Run all tests (lint + format check + unit + coverage)
just ready         # Format, lint before committing
just go-unit       # Run unit tests only
just go-vet        # Run golangci-lint
just go-fmt        # Auto-format (gofumpt + golines)
just just-fmt      # Format justfiles
```

## Running

```bash
# Will lock your terminal — authenticate to exit!
go run .
```

**Warning:** `tlock` locks your terminal and requires Touch ID or password to
unlock. Signals (Ctrl+C, Ctrl+Z) are ignored by design.

## Architecture

Single-binary Go program using CGo for macOS system integration:

```
main.go                   # entry point, calls cmd.Execute()
cmd/
├── root.go               # `tlock` flags (--worms, --pipes, --dvd, --random,
│                         #   --cycle, --delay), builds tlock.Config
└── version.go            # `tlock version`
internal/cli/
└── theme.go              # CLI-only output theme, used by cmd/ only
internal/tlock/
├── lock.go               # Run(): raw mode, signals, stdin reader, dispatch
├── auth.go               # CGo preamble: touchid_available(),
│                         #   authenticate_touchid(), authenticate_password();
│                         #   readPasswordOverlay() with blinking cursor
├── style.go              # teal/gray/red palette, glitch border boxes,
│                         #   renderMessage()
├── terminal.go           # clearScreen, hide/showCursor, centerText,
│                         #   centerBlock, getTermSize, drawLockIcon
├── grid.go               # 3x2-character grid cells, block drawing,
│                         #   wormColors phosphor palette
├── screensaver.go        # screensaver interface, factory map, random pick
├── screensaver_worm.go   # worms (xlock-style)
├── screensaver_pipes.go  # growing pipes
└── screensaver_dvd.go    # bouncing padlock
```

The password prompt is the default lock screen, and Esc switches to Touch ID.
The password cursor is a block (`\u2588`) that blinks every 500ms. Status and
error messages render in boxes with a glitch border built from
`\u2591\u2592\u2593\u2588`.

The UI palette is three colors in `style.go`: teal `#06ffa5` for accents,
prompts and the cursor, gray `245` for hints, and red `196` for errors. The
screensavers share the 13-color retro phosphor palette in `wormColors`.

## Dependencies

| Package                              | Purpose                           |
| ------------------------------------ | --------------------------------- |
| `charmbracelet/lipgloss`             | Terminal styling, colors, borders |
| `golang.org/x/term`                  | Raw terminal mode, terminal size  |
| CGo: `LocalAuthentication.framework` | Touch ID biometric authentication |
| CGo: `Foundation.framework`          | Objective-C runtime support       |
| CGo: `libpam`                        | macOS user password verification  |

## How Authentication Works

### Touch ID Flow

1. `touchid_available()` calls `LAContext canEvaluatePolicy:` to check hardware
2. If available, `authenticate_touchid()` calls `evaluatePolicy:` with a
   callback
3. A `dispatch_semaphore` bridges the async Objective-C callback to synchronous
   C
4. Returns 1 (success) or 0 (failure/unavailable)

### PAM Password Flow

1. `authenticate_password()` gets the current username via `getpwuid(getuid())`
2. Starts a PAM session with the "login" service
3. Custom `pam_conv_func` supplies the password when PAM prompts
4. `pam_authenticate()` verifies against the macOS user account
5. Returns 1 (success) or 0 (failure)

## Raw Terminal Mode

tlock uses `term.MakeRaw()` to put the terminal in raw mode:

- No echo (typed characters aren't displayed)
- No line buffering (each keypress is immediate)
- No signal generation (Ctrl+C doesn't send SIGINT through the terminal driver)
- **Important:** `\n` does NOT include carriage return in raw mode — always use
  `\r\n`

Terminal state is restored via `defer term.Restore()` on exit.

## Signal Handling

- `SIGINT` (Ctrl+C): ignored
- `SIGTERM`: ignored
- `SIGTSTP` (Ctrl+Z): ignored
- `SIGWINCH` (resize): re-renders the lock screen

## Platform Constraints

tlock is macOS-only due to:

- `LocalAuthentication.framework` (Touch ID) — Apple-only API
- PAM configuration differences between macOS and Linux
- CGo with Objective-C (`-x objective-c` flag)

The `.goreleaser.yaml` is configured for `darwin` only with `CGO_ENABLED=1`.

## Sister Projects

| Project                                                        | Description                              |
| -------------------------------------------------------------- | ---------------------------------------- |
| [osapi](https://github.com/osapi-io/osapi)                     | Linux system management REST API and CLI |
| [osapi-justfiles](https://github.com/osapi-io/osapi-justfiles) | Shared justfile recipes for Go projects  |
