# Contributing to tlock

First off, thanks for taking the time to contribute!

## How Can I Contribute?

### Reporting Bugs

- Use the [GitHub issue tracker](https://github.com/retr0h/tlock/issues) to
  report bugs
- Include your macOS version, Go version, and terminal emulator
- Include steps to reproduce the issue
- Note whether Touch ID is available on your hardware

### Suggesting Features

- Open an issue describing the feature you'd like to see
- Explain why this feature would be useful
- Consider whether it fits the project's scope (terminal locking on macOS)

### Code Contributions

#### Small Fixes

Small changes like typos, grammar fixes, and formatting can be submitted
directly as a pull request.

#### Larger Changes

For bug fixes, new features, or significant changes:

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Make your changes
4. Ensure the project builds: `go build -o tlock .`
5. Run the linter: `golangci-lint run`
6. Commit using [Conventional Commits](https://conventionalcommits.org/) format
7. Push to your fork and open a pull request

### Commit Messages

This project uses [Conventional Commits](https://conventionalcommits.org/).
Format: `type(scope): description`

Types: `feat`, `fix`, `docs`, `chore`, `ci`, `build`, `test`, `refactor`

Examples:

```
feat: add worm screensaver mode
fix: handle terminal resize during password input
docs: update README with new configuration options
```

## Development Setup

### Prerequisites

- macOS (required — uses LocalAuthentication.framework and PAM)
- Go 1.21+ with CGo enabled
- Touch ID hardware (optional for development — password fallback works without
  it)

### Building

```bash
git clone https://github.com/retr0h/tlock.git
cd tlock
go build -o tlock .
```

### Testing

```bash
# Run the lock screen (will lock your terminal!)
./tlock

# Build only (won't lock)
go build -o tlock .
```

**Note:** Be careful when testing — `tlock` will lock your terminal and require
authentication to unlock. Signals (Ctrl+C, Ctrl+Z) are ignored by design.

## Code Style

- Follow existing patterns in the codebase
- Use multi-line function signatures
- Use `\r\n` for output in raw terminal mode (not just `\n`)
- Keep the purple/teal/gray color palette consistent
