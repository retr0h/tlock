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
just go::unit      # Run unit tests only
just go::vet       # Run golangci-lint
just go::fmt       # Auto-format (gofumpt + golines)
just just::fmt     # Format justfiles
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
main.go
├── CGo preamble (Objective-C)
│   ├── touchid_available()     — check if Touch ID hardware is present
│   ├── authenticate_touchid()  — biometric auth via LocalAuthentication.framework
│   └── authenticate_password() — password auth via PAM
├── Styles (lipgloss)
│   ├── lockTitleStyle          — bold purple for "LOCKED"
│   ├── subtitleStyle           — gray for hostname/hints
│   ├── promptStyle             — teal for auth prompts
│   ├── errorStyle              — bold red for failures
│   ├── msgBoxStyle             — glitch border box (teal)
│   └── errBoxStyle             — glitch border box (red)
├── Terminal helpers
│   ├── clearScreen()           — ANSI escape to clear
│   ├── hideCursor/showCursor() — ANSI cursor visibility
│   ├── centerText()            — center text horizontally
│   ├── centerBlock()           — center multi-line block
│   └── getTermSize()           — terminal dimensions
├── Rendering
│   ├── renderLockScreen()      — main lock screen display
│   ├── renderMessage()         — boxed status messages
│   └── readPassword()          — password input with blinking cursor
└── main()                      — entry point, auth loop
```

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
