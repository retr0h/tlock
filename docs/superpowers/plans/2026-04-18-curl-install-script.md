# Curl Install Script Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a POSIX `sh` installer at the repo root so users can install tlock with `curl -fsSL https://github.com/retr0h/tlock/raw/main/install.sh | sh`, with SHA256 verification and swamp.club-style destination logic.

**Architecture:** Single POSIX shell script (`install.sh`) at repo root. macOS-only. Resolves version from GitHub API (or `TLOCK_VERSION` env), picks an install dir per user privileges + PATH, downloads binary and `checksums.txt` from `github.com/retr0h/tlock/releases/latest/download/...`, verifies SHA256, strips quarantine xattr, installs with `install -m 755`. README's install section is rewritten to lead with the one-liner. Shellcheck enforced in CI.

**Tech Stack:** POSIX `sh`, `curl`/`wget`, `shasum`, GitHub Releases, shellcheck, GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-04-18-curl-install-script-design.md`

---

### Task 1: Wire up shellcheck in CI

Add a shellcheck job to the existing Go workflow so every PR that touches `install.sh` gets linted. We do this first so the script is under static analysis from the first commit.

**Files:**
- Modify: `.github/workflows/go.yml`

- [ ] **Step 1: Add shellcheck job**

Edit `.github/workflows/go.yml`, adding a second job after `build`:

```yaml
---
name: Go

on:
  push:
    branches: [ "main" ]
  pull_request:
    branches: [ "main" ]

jobs:
  build:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v6
      - name: Set up Go
        uses: actions/setup-go@v6
        with:
          go-version: stable
      - name: Build
        run: go build -o tlock .
      - name: Test
        run: go test -v ./...

  shellcheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - name: Run shellcheck
        run: |
          if [ -f install.sh ]; then
            shellcheck install.sh
          else
            echo "install.sh not present yet — skipping"
          fi
```

The conditional guard keeps the first commit green. We remove the guard in Task 2 once the file lands.

- [ ] **Step 2: Verify YAML parses**

Run: `yq '.jobs.shellcheck.runs-on' .github/workflows/go.yml`
Expected: `ubuntu-latest`

If `yq` is not installed: `python3 -c 'import yaml; print(yaml.safe_load(open(".github/workflows/go.yml"))["jobs"]["shellcheck"]["runs-on"])'` → `ubuntu-latest`

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/go.yml
git commit -m "ci: add shellcheck job (conditional until install.sh lands)"
```

---

### Task 2: Skeleton with OS/arch detection

Land the first version of `install.sh`: shebang, strict mode, OS gate, arch detection. No download yet — the script just prints what it would do and exits. This is the smallest commit that proves the header + platform checks work.

**Files:**
- Create: `install.sh`
- Modify: `.github/workflows/go.yml` (drop the conditional)

- [ ] **Step 1: Write `install.sh` skeleton**

Create `install.sh` at the repo root:

```sh
#!/bin/sh
#
# tlock installer
# Usage: curl -fsSL https://github.com/retr0h/tlock/raw/main/install.sh | sh
#
# Env overrides:
#   TLOCK_VERSION       — install a specific version (e.g. 1.1.1) instead of latest
#   TLOCK_INSTALL_DIR   — force install destination, skipping the default rules

set -eu

err() {
    printf 'tlock: %s\n' "$1" >&2
    exit 1
}

detect_os() {
    os=$(uname -s)
    if [ "$os" != "Darwin" ]; then
        err "macOS only. Build from source: https://github.com/retr0h/tlock#-build-from-source"
    fi
}

detect_arch() {
    machine=$(uname -m)
    case "$machine" in
        arm64)   arch=arm64 ;;
        x86_64)  arch=amd64 ;;
        *)       err "unsupported architecture: $machine" ;;
    esac
}

main() {
    detect_os
    detect_arch
    printf 'tlock: detected darwin/%s\n' "$arch"
}

main "$@"
```

- [ ] **Step 2: Make executable and run**

```bash
chmod +x install.sh
./install.sh
```

Expected on darwin/arm64: `tlock: detected darwin/arm64`
Expected on darwin/amd64: `tlock: detected darwin/amd64`

- [ ] **Step 3: Run shellcheck locally**

Run: `shellcheck install.sh`
Expected: no output (clean exit 0)

If shellcheck isn't installed: `brew install shellcheck` first.

- [ ] **Step 4: Drop the conditional guard from CI**

Edit `.github/workflows/go.yml`, replacing the shellcheck step with the unconditional form:

```yaml
  shellcheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - name: Run shellcheck
        run: shellcheck install.sh
```

- [ ] **Step 5: Commit**

```bash
git add install.sh .github/workflows/go.yml
git commit -m "feat: add install.sh skeleton with OS/arch detection"
```

---

### Task 3: Version resolution

Add the logic that picks which version to install — `TLOCK_VERSION` env var if set, else the latest release tag from the GitHub API. Print it and still exit without downloading.

**Files:**
- Modify: `install.sh`

- [ ] **Step 1: Add `resolve_version`**

Insert after `detect_arch` and before `main`:

```sh
have() {
    command -v "$1" >/dev/null 2>&1
}

http_get() {
    # $1 = url, prints body to stdout
    if have curl; then
        curl -fsSL "$1"
    elif have wget; then
        wget -qO- "$1"
    else
        err "neither curl nor wget found on PATH"
    fi
}

resolve_version() {
    if [ -n "${TLOCK_VERSION:-}" ]; then
        version=${TLOCK_VERSION#v}
        return
    fi
    tag=$(http_get https://api.github.com/repos/retr0h/tlock/releases/latest \
        | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' \
        | head -n1)
    if [ -z "$tag" ]; then
        err "could not determine latest version from GitHub API"
    fi
    version=${tag#v}
}
```

Update `main` to call it and print:

```sh
main() {
    detect_os
    detect_arch
    resolve_version
    printf 'tlock: darwin/%s version %s\n' "$arch" "$version"
}
```

- [ ] **Step 2: Run with latest (default)**

Run: `./install.sh`
Expected (current latest): `tlock: darwin/arm64 version 1.1.1` (or amd64 on Intel)

- [ ] **Step 3: Run with a pinned version**

Run: `TLOCK_VERSION=1.1.0 ./install.sh`
Expected: `tlock: darwin/arm64 version 1.1.0`

Also verify the `v` prefix is stripped:

Run: `TLOCK_VERSION=v1.0.0 ./install.sh`
Expected: `tlock: darwin/arm64 version 1.0.0`

- [ ] **Step 4: Run shellcheck**

Run: `shellcheck install.sh`
Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add install.sh
git commit -m "feat(install): resolve latest version from GitHub API with TLOCK_VERSION override"
```

---

### Task 4: Install directory resolution

Add the swamp-style destination logic: root → `/usr/local/bin`, else `$HOME/.local/bin` or `$HOME/bin` if on PATH, else fallback to `$HOME/.tlock/bin` with a symlink flag. Respect `TLOCK_INSTALL_DIR`. Still no download.

**Files:**
- Modify: `install.sh`

- [ ] **Step 1: Add `path_contains` and `resolve_install_dir`**

Insert after `resolve_version`:

```sh
path_contains() {
    # $1 = dir to look for in $PATH
    case ":$PATH:" in
        *":$1:"*) return 0 ;;
        *)        return 1 ;;
    esac
}

resolve_install_dir() {
    needs_symlink=0
    if [ -n "${TLOCK_INSTALL_DIR:-}" ]; then
        install_dir=$TLOCK_INSTALL_DIR
        return
    fi
    if [ "$(id -u)" = "0" ]; then
        install_dir=/usr/local/bin
        return
    fi
    if path_contains "$HOME/.local/bin"; then
        install_dir=$HOME/.local/bin
        return
    fi
    if path_contains "$HOME/bin"; then
        install_dir=$HOME/bin
        return
    fi
    install_dir=$HOME/.tlock/bin
    needs_symlink=1
}
```

Update `main`:

```sh
main() {
    detect_os
    detect_arch
    resolve_version
    resolve_install_dir
    printf 'tlock: darwin/%s version %s → %s (symlink=%d)\n' \
        "$arch" "$version" "$install_dir" "$needs_symlink"
}
```

- [ ] **Step 2: Run with `.local/bin` on PATH**

```bash
PATH="$HOME/.local/bin:$PATH" ./install.sh
```

Expected: `tlock: darwin/... version X.Y.Z → /Users/.../.local/bin (symlink=0)`

- [ ] **Step 3: Run with nothing on PATH**

```bash
PATH=/usr/bin:/bin ./install.sh
```

Expected ends with `→ /Users/.../.tlock/bin (symlink=1)`

- [ ] **Step 4: Run with override**

```bash
TLOCK_INSTALL_DIR=/tmp/tlock-install ./install.sh
```

Expected: `→ /tmp/tlock-install (symlink=0)`

- [ ] **Step 5: Run with `$HOME/bin` on PATH but not `.local/bin`**

```bash
PATH="$HOME/bin:/usr/bin:/bin" ./install.sh
```

Expected: `→ /Users/.../bin (symlink=0)`

- [ ] **Step 6: Run shellcheck**

Run: `shellcheck install.sh`
Expected: no output.

- [ ] **Step 7: Commit**

```bash
git add install.sh
git commit -m "feat(install): resolve install directory with PATH-aware rules"
```

---

### Task 5: Download and checksum verification

Add the download + verify phase. Download the binary and `checksums.txt` into a temp dir, verify SHA256, strip the quarantine attribute. Still stop short of installing so we can test checksum handling in isolation.

**Files:**
- Modify: `install.sh`

- [ ] **Step 1: Add temp dir setup and download logic**

Insert after `resolve_install_dir`:

```sh
setup_tmp() {
    tmp=$(mktemp -d 2>/dev/null || mktemp -d -t tlock-install)
    trap 'rm -rf "$tmp"' EXIT
}

download() {
    base=https://github.com/retr0h/tlock/releases/download/v${version}
    asset=tlock_${version}_darwin_${arch}

    if have curl; then
        curl -fsSL -o "$tmp/tlock" "$base/$asset" \
            || err "failed to download $base/$asset"
        curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt" \
            || err "failed to download $base/checksums.txt"
    else
        wget -q -O "$tmp/tlock" "$base/$asset" \
            || err "failed to download $base/$asset"
        wget -q -O "$tmp/checksums.txt" "$base/checksums.txt" \
            || err "failed to download $base/checksums.txt"
    fi
}

verify_checksum() {
    asset=tlock_${version}_darwin_${arch}
    expected=$(grep " $asset\$" "$tmp/checksums.txt" | awk '{print $1}')
    if [ -z "$expected" ]; then
        err "no checksum entry for $asset in checksums.txt"
    fi
    actual=$(shasum -a 256 "$tmp/tlock" | awk '{print $1}')
    if [ "$expected" != "$actual" ]; then
        printf 'tlock: checksum mismatch for %s\n  expected: %s\n  actual:   %s\n' \
            "$asset" "$expected" "$actual" >&2
        exit 1
    fi
}

strip_quarantine() {
    xattr -d com.apple.quarantine "$tmp/tlock" 2>/dev/null || true
}
```

Update `main`:

```sh
main() {
    detect_os
    detect_arch
    resolve_version
    resolve_install_dir
    setup_tmp
    download
    verify_checksum
    strip_quarantine
    printf 'tlock: verified %s, ready to install to %s\n' "$version" "$install_dir"
}
```

- [ ] **Step 2: Run the happy path**

```bash
TLOCK_INSTALL_DIR=/tmp/tlock-test ./install.sh
```

Expected final line: `tlock: verified 1.1.1, ready to install to /tmp/tlock-test`

Confirm temp dir was cleaned up:

```bash
ls /tmp/ | grep -i tlock.*install
```

Expected: only `tlock-test` (the override dir, if it existed), no `mktemp` leftovers.

- [ ] **Step 3: Test checksum failure path**

Temporarily break the checksum by patching the script to download a different asset:

```bash
sed -i.bak 's/tlock_${version}_darwin_${arch}/tlock_${version}_darwin_amd64/' install.sh
# On an arm64 Mac, this now downloads the amd64 binary but verifies the arm64 checksum
./install.sh || echo "exit=$?"
mv install.sh.bak install.sh
```

Expected output includes `tlock: checksum mismatch` and `exit=1`.

(Skip this step on Intel Macs — the mismatch won't trigger. Instead, manually edit a checksum byte in a local copy of `checksums.txt` and point the script at a file URL.)

- [ ] **Step 4: Run shellcheck**

Run: `shellcheck install.sh`
Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add install.sh
git commit -m "feat(install): download binary and verify SHA256 checksum"
```

---

### Task 6: Install, symlink, and PATH hint

The final piece of the script: actually install the binary, optionally symlink into `/usr/local/bin`, and print a PATH hint when the install dir isn't on PATH.

**Files:**
- Modify: `install.sh`

- [ ] **Step 1: Add install logic**

Insert after `strip_quarantine`:

```sh
install_binary() {
    mkdir -p "$install_dir" || err "cannot create $install_dir"
    install -m 755 "$tmp/tlock" "$install_dir/tlock" \
        || err "cannot write to $install_dir/tlock"
}

maybe_symlink() {
    [ "$needs_symlink" = "1" ] || return 0
    if [ -w /usr/local/bin ] || { [ ! -e /usr/local/bin/tlock ] && [ -w /usr/local ]; }; then
        ln -sf "$install_dir/tlock" /usr/local/bin/tlock 2>/dev/null || true
    fi
}

print_summary() {
    printf 'tlock v%s installed to %s/tlock\n' "$version" "$install_dir"
    if ! path_contains "$install_dir"; then
        printf '\nAdd this to your shell rc:\n  export PATH="%s:$PATH"\n' "$install_dir"
    fi
}
```

Update `main`:

```sh
main() {
    detect_os
    detect_arch
    resolve_version
    resolve_install_dir
    setup_tmp
    download
    verify_checksum
    strip_quarantine
    install_binary
    maybe_symlink
    print_summary
}
```

- [ ] **Step 2: Smoke test with override**

```bash
rm -f /tmp/tlock-test/tlock
TLOCK_INSTALL_DIR=/tmp/tlock-test ./install.sh
/tmp/tlock-test/tlock --help 2>&1 | head -5
```

Expected: binary installed, `--help` output shows tlock usage.

- [ ] **Step 3: Smoke test the PATH hint path**

```bash
rm -rf "$HOME/.tlock"
PATH=/usr/bin:/bin ./install.sh
```

Expected output includes:
```
tlock vX.Y.Z installed to /Users/.../.tlock/bin/tlock

Add this to your shell rc:
  export PATH="/Users/.../.tlock/bin:$PATH"
```

Clean up: `rm -rf "$HOME/.tlock"`

- [ ] **Step 4: Smoke test the `.local/bin` path**

```bash
mkdir -p "$HOME/.local/bin"
rm -f "$HOME/.local/bin/tlock"
PATH="$HOME/.local/bin:$PATH" ./install.sh
ls -la "$HOME/.local/bin/tlock"
```

Expected: file exists, mode `-rwxr-xr-x`, no PATH hint printed.

Clean up: `rm -f "$HOME/.local/bin/tlock"`

- [ ] **Step 5: Run shellcheck**

Run: `shellcheck install.sh`
Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add install.sh
git commit -m "feat(install): install binary with optional symlink and PATH hint"
```

---

### Task 7: Rewrite README install section

Replace the verbose per-arch curl blocks with the one-liner, collapse the existing instructions under `<details>`, keep Build from source unchanged.

**Files:**
- Modify: `README.md` (lines 61–92)

- [ ] **Step 1: Read the current section**

Open `README.md`. The target block is everything between `## 📦 Install` and `## 🚀 Usage`.

- [ ] **Step 2: Replace the install section**

Replace that block with:

````markdown
## 📦 Install

```bash
curl -fsSL https://github.com/retr0h/tlock/raw/main/install.sh | sh
```

Installs to `~/.local/bin`, `~/bin`, or `/usr/local/bin` (root). SHA256 checksums are verified against the release. Override the destination with `TLOCK_INSTALL_DIR=/some/path` or pin a version with `TLOCK_VERSION=1.1.1`.

<details>
<summary>Manual install</summary>

### ⬇️ Download binary (macOS)

Grab the latest release for your architecture:

```bash
# Apple Silicon (M1/M2/M3/M4)
curl -sL https://github.com/retr0h/tlock/releases/latest/download/tlock_$(curl -sL https://api.github.com/repos/retr0h/tlock/releases/latest | grep tag_name | cut -d '"' -f4 | tr -d v)_darwin_arm64 -o tlock

# Intel Mac
curl -sL https://github.com/retr0h/tlock/releases/latest/download/tlock_$(curl -sL https://api.github.com/repos/retr0h/tlock/releases/latest | grep tag_name | cut -d '"' -f4 | tr -d v)_darwin_amd64 -o tlock

chmod +x tlock
sudo mv tlock /usr/local/bin/
```

### 🔏 Verify checksum

```bash
curl -sL https://github.com/retr0h/tlock/releases/latest/download/checksums.txt -o checksums.txt
grep "$(uname -s | tr A-Z a-z)_$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')" checksums.txt | sed 's/tlock_.*$/tlock/' | shasum -a 256 -c
```

</details>

### 🔨 Build from source

```bash
git clone https://github.com/retr0h/tlock.git
cd tlock
go build -o tlock .
sudo mv tlock /usr/local/bin/
```

````

- [ ] **Step 3: Verify rendered markdown**

Run: `grep -c 'curl -fsSL https://github.com/retr0h/tlock/raw/main/install.sh' README.md`
Expected: `1`

Run: `grep -c '<details>' README.md`
Expected: `1`

Run: `grep -c '### 🔨 Build from source' README.md`
Expected: `1`

- [ ] **Step 4: Run docs lint if present**

```bash
just docs::fmt-check 2>&1 || just docs::fmt
```

If `just` isn't wired up for docs in this checkout, skip.

- [ ] **Step 5: Commit**

```bash
git add README.md
git commit -m "docs: lead install section with curl one-liner"
```

---

### Task 8: End-to-end smoke test and PR

Verify the complete script against the actual URL from a fresh shell and open the PR.

**Files:** none (manual verification + PR creation).

- [ ] **Step 1: Test from a fresh shell via the raw URL**

```bash
rm -f "$HOME/.local/bin/tlock"
env -i HOME="$HOME" PATH="$HOME/.local/bin:/usr/bin:/bin" sh -c \
    'curl -fsSL https://github.com/retr0h/tlock/raw/REPLACE_WITH_BRANCH/install.sh | sh'
"$HOME/.local/bin/tlock" --help 2>&1 | head -3
```

Replace `REPLACE_WITH_BRANCH` with your feature branch name (since the URL isn't on `main` yet). Expected: version banner + help text.

Clean up: `rm -f "$HOME/.local/bin/tlock"`

- [ ] **Step 2: Test `TLOCK_VERSION` pin end-to-end**

```bash
TLOCK_VERSION=1.1.0 TLOCK_INSTALL_DIR=/tmp/tlock-pin sh install.sh
/tmp/tlock-pin/tlock --help 2>&1 | head -1
```

Expected: binary installed, help output works (1.1.0 behavior, whatever that was).

Clean up: `rm -rf /tmp/tlock-pin`

- [ ] **Step 3: Push branch**

```bash
git push -u origin HEAD
```

- [ ] **Step 4: Open PR**

```bash
gh pr create --title "feat: add curl install script" --body "$(cat <<'EOF'
## Summary
- Adds `install.sh` at repo root for one-line install: `curl -fsSL https://github.com/retr0h/tlock/raw/main/install.sh | sh`
- Verifies SHA256 checksums against `checksums.txt` before installing
- Install destination follows swamp.club logic: `$HOME/.local/bin` or `$HOME/bin` if on PATH, `/usr/local/bin` for root, `$HOME/.tlock/bin` as fallback
- `TLOCK_VERSION` and `TLOCK_INSTALL_DIR` env overrides
- README install section rewritten to lead with the one-liner
- Shellcheck enforced in CI

Spec: `docs/superpowers/specs/2026-04-18-curl-install-script-design.md`

## Test plan
- [ ] CI `shellcheck` job green
- [ ] CI `build` job green
- [ ] Manually verified install on Apple Silicon via raw URL from the branch
- [ ] Manually verified checksum mismatch aborts installation
- [ ] Manually verified `TLOCK_VERSION` and `TLOCK_INSTALL_DIR` overrides work

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

- [ ] **Step 5: Verify CI is green**

```bash
gh pr checks
```

Expected: both `build` and `shellcheck` pass.

---

## Self-Review

**Spec coverage:**
- One-liner install command → Task 2, 7
- macOS-only OS gate → Task 2
- Arch detection (arm64, amd64) → Task 2
- `TLOCK_VERSION` env var → Task 3
- `TLOCK_INSTALL_DIR` env var → Task 4
- Install destination rules (root / `.local/bin` / `bin` / fallback) → Task 4
- Symlink for fallback case → Task 6
- PATH hint when install dir not on PATH → Task 6
- SHA256 verification against `checksums.txt` → Task 5
- Quarantine xattr strip → Task 5
- Error messages with `tlock:` prefix → Task 2 (`err` helper) + individual sites
- Install with `install -m 755` → Task 6
- Temp dir cleanup via trap → Task 5
- README rewrite with collapsed manual section → Task 7
- Shellcheck in CI → Task 1
- Manual smoke tests → embedded in Tasks 2–8

All spec requirements are covered.

**Placeholder scan:** No TBDs, TODOs, "implement later", or "similar to" references. Every code step has complete code.

**Type / name consistency:**
- `$arch`, `$version`, `$install_dir`, `$needs_symlink`, `$tmp` — defined in Tasks 2–5, used consistently after.
- `err`, `have`, `http_get`, `path_contains`, `resolve_version`, `resolve_install_dir`, `setup_tmp`, `download`, `verify_checksum`, `strip_quarantine`, `install_binary`, `maybe_symlink`, `print_summary` — all defined once, called once from `main`.
- `tlock_${version}_darwin_${arch}` asset name is consistent between `download` and `verify_checksum`.

Plan is ready for execution.
