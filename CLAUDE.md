# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

tlock is a terminal lock program for macOS written in Go. It locks the terminal with a styled lock screen and requires Touch ID (biometric) or macOS user password to unlock. Designed as a tmux `lock-command` replacement.

## Architecture

Single-binary Go program using CGo for macOS system integration:

- **`main.go`** - Entry point, terminal handling, lock screen UI, auth flow
- **CGo** - Touch ID via `LocalAuthentication.framework`, password via PAM (`pam_authenticate`)
- **lipgloss** - Terminal styling (purple/teal/gray color palette)
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
go run main.go         # Run directly (will lock terminal!)
```

## Usage

```bash
# Direct (password prompt only)
tlock

# Worms immediately
tlock --snake

# Screensaver after 30s idle
tlock --screensaver

# Custom delay
tlock --screensaver --screensaver-delay 60

# As tmux lock-command
set -g lock-command "tlock --snake"
set -g lock-after-time 1800
bind ^X lock-server
```

## Color Palette

```
Purple  = lipgloss.Color("99")       // Headers, lock title
Teal    = lipgloss.Color("#06ffa5")  // Accent, prompts, blinking cursor
Gray    = lipgloss.Color("245")      // Dim/secondary text (hostname, hints)
Red     = lipgloss.Color("196")      // Errors (auth failed)
```

## Code Standards

- Follow [Conventional Commits](https://www.conventionalcommits.org/) for commit messages
- Use `testify/suite` with table-driven tests
- Multi-line function signatures
- macOS-only (CGo dependencies on LocalAuthentication + PAM)
- golangci-lint with: errcheck, errname, govet, prealloc, predeclared, revive, staticcheck

## Roadmap

- Phase 2: xlock-style worm screensaver with fading trails + cycling figurine text
- Phase 3: Configuration file support (`~/.config/tlock/config.yaml`)
