set allow-duplicate-variables

# Optional modules: import? allows `just fetch` to work before .just/remote/ exists.

import? '.just/remote/go.just'
import? '.just/remote/md.just'
import? '.just/remote/just.just'

# No documentation site, so md formats every markdown file in the repository.

md_site_dir := ""

# tlock has no tests, so the module's default of 100 cannot be met. Declaring
# the real floor keeps the gate honest rather than disabling the check.

go_coverage_target := "0"

# --- Fetch ---

# Fetch shared justfiles from osapi-justfiles
fetch:
    mkdir -p .just/remote
    curl -sSfL https://raw.githubusercontent.com/osapi-io/osapi-justfiles/refs/heads/main/go/go.just -o .just/remote/go.just
    curl -sSfL https://raw.githubusercontent.com/osapi-io/osapi-justfiles/refs/heads/main/md/md.just -o .just/remote/md.just
    curl -sSfL https://raw.githubusercontent.com/osapi-io/osapi-justfiles/refs/heads/main/just/just.just -o .just/remote/just.just

# --- Top-level orchestration ---

# Install all dependencies
deps:
    just go-deps
    just go-mod

# Run all tests
test:
    just just-fmt-check
    just go-test

# Format, lint before committing
ready:
    just just-fmt
    just md-fmt
    just go-fmt
    just go-vet
