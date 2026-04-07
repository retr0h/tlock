# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

tlock is a terminal lock program for macOS written in Go. It locks the terminal with a styled lock screen and requires Touch ID (biometric) or macOS user password to unlock. Designed as a tmux `lock-command` replacement.

## Architecture

Single-binary Go program using CGo for macOS system integration:

- **`main.go`** - Entry point, flag parsing, terminal setup, screensaver dispatch
- **`terminal.go`** - Terminal utilities (clear, cursor, centering, lock icon, resize)
- **`auth.go`** - CGo Touch ID + PAM authentication, password overlay
- **`style.go`** - Lipgloss styles, color palette, message rendering
- **`grid.go`** - Grid cell system (3x2 chars), block drawing, phosphor colors
- **`screensaver.go`** - Screensaver interface, factory registry, random selection
- **`screensaver_worm.go`** - Worms screensaver (xlock-style)
- **`screensaver_dvd.go`** - Bouncing padlock screensaver
- **`screensaver_pipes.go`** - Growing pipes screensaver
- **CGo** - Touch ID via `LocalAuthentication.framework`, password via PAM (`pam_authenticate`)
- **lipgloss** - Terminal styling (teal/gray/red color palette)
- **golang.org/x/term** - Raw terminal mode, terminal size detection

## Key Technical Details

- Uses raw terminal mode — all output needs `\r\n` not just `\n`
- Signals (SIGINT, SIGTERM, SIGTSTP) are ignored to prevent bypass
- Touch ID is async (Objective-C block callback), bridged to sync via `dispatch_semaphore`
- `touchid_available()` checks biometric hardware before prompting
- PAM auth uses the "login" service with the current user
- Glitch-style unicode border (`\u2591\u2592\u2593\u2588`) for auth prompts
- Blinking block cursor (`\u2588`) on password input, 500ms interval
- Password prompt is the default lock screen — Esc switches to Touch ID

## Building

```bash
go build -o tlock .    # Build binary
go run . --worms       # Run directly (will lock terminal!)
```

## Usage

```bash
# Direct (password prompt only)
tlock

# Screensavers (immediate)
tlock --worms                  # Worms
tlock --pipes                  # Growing pipes
tlock --dvd                    # Bouncing padlock
tlock --random                 # Random pick
tlock --random --cycle 5m      # Rotate every 5 min

# With idle delay
tlock --worms --delay 30s      # Worms after 30s idle

# As tmux lock-command
set -g lock-command "tlock --random --cycle 5m"
set -g lock-after-time 1800
bind ^X lock-server
```

## Color Palette

```
Teal    = lipgloss.Color("#06ffa5")  // Accent, prompts, blinking cursor
Gray    = lipgloss.Color("245")      // Dim/secondary text (hostname, hints)
Red     = lipgloss.Color("196")      // Errors (auth failed)
```

Screensavers use a 13-color retro phosphor CRT palette defined in `wormColors` (grid.go).

## Code Standards

- Follow [Conventional Commits](https://www.conventionalcommits.org/) for commit messages
- Use `testify/suite` with table-driven tests
- Multi-line function signatures
- macOS-only (CGo dependencies on LocalAuthentication + PAM)
- golangci-lint with: errcheck, errname, govet, prealloc, predeclared, revive, staticcheck

## Roadmap

- 1Password CLI integration
