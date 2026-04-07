# New Screensavers Design Spec

## Overview

Add two new screensavers to tlock (Bouncing DVD Logo, Pipes) and unify the CLI flag structure under `--screensaver <name>`.

## Screensaver 1: Bouncing DVD Logo (`dvd`)

- "tlock" text built from block characters (░▒▓█) with glitch-style border (same style as auth prompt: `\u2591\u2592\u2593\u2588`)
- Bounces around terminal, reversing direction on wall hits
- ~120ms tick rate (consistent with worms)
- Clean bounce — erases previous position before drawing new one, no ghost trail
- Changes to a random color from the phosphor palette on each wall hit
- Lock icon preserved in top-left corner
- Keypress triggers auth overlay (same flow as worms)
- Uses the same 3×2 grid cell system for positioning

## Screensaver 2: Pipes (`pipes`)

- Multiple pipes grow simultaneously from random start points
- Each pipe grows one cell per tick, turning 90° at random intervals
- Drawn with block characters (░▒▓█) on the same 3×2 grid cell system
- Each pipe gets a distinct color from the existing phosphor palette
- When screen fills up: implement **both** hard-reset and fade-out modes so user can compare visually and pick
  - Hard reset: clear screen, restart immediately
  - Fade out: trails fade through ░▒▓ stages (like worm trail aging), then new pipes begin
- Lock icon preserved in top-left
- Keypress triggers auth overlay (same flow as worms)

## CLI Flag Changes

### New unified flag

`--screensaver <name>` accepts: `snake`, `pipes`, `dvd`, `random`

- `random` picks one screensaver at random on startup
- `--screensaver-cycle N` rotates to a different random screensaver every N minutes (requires `--screensaver random`)
- `--screensaver-delay N` unchanged — idle seconds before screensaver activates

### Backwards compatibility

- `--snake` kept as alias for `--screensaver snake`
- `--snake-count` kept as alias for `--worm-count`
- Existing `--screensaver` boolean flag becomes a string flag; bare `--screensaver` (no value) is an error — must specify a name

## Architecture

### Screensaver interface

```go
type screensaver interface {
    run(width, height int) bool // returns true if user authenticated
}
```

Each screensaver implements this interface. The `run()` method owns the animation loop, input handling, and auth overlay flow.

### File structure

Split `main.go` into focused files:

- `main.go` — entry point, flag parsing, terminal setup, screensaver dispatch
- `terminal.go` — shared terminal utilities (clearScreen, hideCursor, showCursor, centerText, centerBlock, getTermSize, drawLockIcon, clearRect)
- `auth.go` — authentication flow (handleAuth, readPasswordOverlay, Touch ID, PAM)
- `style.go` — lipgloss styles, color palette, block character constants
- `grid.go` — grid cell types, drawBlock, eraseBlock, shared grid logic
- `screensaver_worm.go` — existing worm/snake screensaver (extracted from main.go)
- `screensaver_dvd.go` — bouncing DVD logo
- `screensaver_pipes.go` — pipes screensaver

### Shared infrastructure

All screensavers share:
- Grid cell system (3 chars wide × 2 lines tall)
- `drawBlock()` / `eraseBlock()` rendering
- Lock icon in top-left corner
- Phosphor color palette (13 colors)
- Auth overlay triggered by keypress
- SIGWINCH resize handling
- Signal ignore (SIGINT, SIGTERM, SIGTSTP)

### Screensaver dispatch

```go
func runScreensaver(name string, width, height int) bool {
    switch name {
    case "snake":
        return (&wormScreensaver{}).run(width, height)
    case "dvd":
        return (&dvdScreensaver{}).run(width, height)
    case "pipes":
        return (&pipesScreensaver{}).run(width, height)
    case "random":
        // pick random from [snake, dvd, pipes]
    }
}
```

Cycling (`--screensaver-cycle N`) wraps this in a timer that calls `runScreensaver("random", ...)` every N minutes, skipping the currently active one.

## Visual Consistency

All three screensavers use:
- Same phosphor color palette (teal, greens, amber, cyan, magenta, purple, pink, yellow, lavender, orange)
- Same block character set (░▒▓█)
- Same 3×2 grid cell sizing
- Same lock icon placement and style
- Same auth overlay appearance and flow

## User Review Items

After implementation, user wants to visually compare in the terminal:
1. DVD logo with clean bounce (confirmed) — user also wants to see ghost trail variant to compare
2. Pipes hard-reset vs fade-out when screen fills — user picks after seeing both
