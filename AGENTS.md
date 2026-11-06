# AGENTS.md

Test: `just test` | Before committing: `just ready`

Read [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) first. It covers prerequisites,
setup, the `just` recipes, the architecture, how authentication works, raw
terminal mode, signal handling, and platform constraints.
[docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) covers bug reports, commit
messages, code style, and the pull request workflow. All of it applies to agents
exactly as it applies to people. This file carries only what is specific to
agents.

## Running tools

Invoke tools through `mise`, not from your path:

```bash
mise exec -- just test
```

`mise` is active in a person's shell and supplies the versions `.mise.toml`
declares. An agent's shell has no activation, so a bare `just` resolves to
whatever is installed globally, usually an older version.

The symptom is a check that fails here and passes in continuous integration, on
a file nobody edited. When that happens, establish which version ran before
treating the failure as real.

## Never run tlock

Do not run `tlock`, `./tlock` or `go run .`, with or without flags. It locks the
terminal, ignores SIGINT, SIGTERM and SIGTSTP, and unlocks only for Touch ID or
the user's password. An agent can supply neither, so the session hangs until a
person kills the process. `go build -o tlock .` and `just test` are safe; they
compile without running the lock.

## Commit trailer

When committing via Claude Code, end the message with:

```
🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
```
