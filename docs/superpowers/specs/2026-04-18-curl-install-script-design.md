# Curl Install Script Design Spec

## Overview

Add a POSIX `sh` installer at the repo root so users can install tlock with a
single line, matching the ergonomics of tools like rustup, starship, and
swamp.club:

```bash
curl -fsSL https://github.com/retr0h/tlock/raw/main/install.sh | sh
```

The README's install section is rewritten to lead with this one-liner. The
existing manual-download + checksum-verification snippets move into a
collapsed `<details>` block, and the "Build from source" section is unchanged.

## Goals

- One-line install that Just Works on macOS (Apple Silicon and Intel).
- Verifies SHA256 checksums against the release's `checksums.txt` before
  installing. Abort on mismatch.
- No `sudo` required for the common case (normal user with `~/.local/bin` on
  PATH). Fall back to a private dir + symlink if nothing on PATH is writable.
- Clear, actionable errors when the host is not macOS, the arch is
  unrecognized, or a required tool is missing.

## Non-Goals

- Uninstaller script (deferred).
- Homebrew tap (tracked separately).
- Auto-update / self-update.
- Landing page or GitHub Pages site.
- Linux support (tlock is macOS-only by design — CGo + LocalAuthentication +
  PAM).

## Install Destination Logic

Adopted from swamp.club's installer, renamed for tlock:

```
1. root (id -u == 0)                  → /usr/local/bin
2. $HOME/.local/bin is in $PATH       → $HOME/.local/bin
3. $HOME/bin is in $PATH              → $HOME/bin
4. fallback                           → $HOME/.tlock/bin
                                        + symlink /usr/local/bin/tlock → $HOME/.tlock/bin/tlock
                                        (only if /usr/local/bin is writable without sudo;
                                         otherwise skip the symlink and print a PATH hint)
```

Environment overrides:

- `TLOCK_INSTALL_DIR=/some/path` — force destination, skip the rules above.
- `TLOCK_VERSION=1.1.0` — install a specific version instead of latest.

## Script Flow

1. `set -eu` at the top. Trap `EXIT` to clean up the temp dir.
2. **OS check** — `uname -s` must be `Darwin`. Otherwise print an error
   referring the user to the "Build from source" section and exit 1.
3. **Arch detection** — `uname -m`:
   - `arm64` → `arm64`
   - `x86_64` → `amd64`
   - anything else → error with "unsupported architecture: <value>" and exit 1.
4. **Resolve version** — if `$TLOCK_VERSION` is set, use it verbatim. Else
   fetch `https://api.github.com/repos/retr0h/tlock/releases/latest` and parse
   the `tag_name` field (e.g. `v1.1.1`), stripping the leading `v` for asset
   naming.
5. **Pick install dir** — apply the rules above. Compute a boolean
   `needs_symlink` for case 4.
6. **Create temp dir** — `tmp=$(mktemp -d)`; `trap 'rm -rf "$tmp"' EXIT`.
7. **Download** — fetch into `$tmp`:
   - `tlock_${version}_darwin_${arch}` → the binary (saved as `tlock`)
   - `checksums.txt`
   Use `curl -fsSL -o` (or `wget -q -O` as fallback).
8. **Verify checksum** — filter `checksums.txt` to the line matching the
   downloaded asset name, rewrite the filename to `tlock`, and pipe into
   `shasum -a 256 -c -` (run from `$tmp`). Abort on failure with the expected
   vs. actual hash.
9. **Strip quarantine** — `xattr -d com.apple.quarantine "$tmp/tlock" 2>/dev/null || true`
   (harmless if the attr isn't set; curl-downloaded binaries normally won't
   have it, but this matches the swamp.club installer and protects users who
   curl into a file then run the installer separately).
10. **Install** — `install -m 755 "$tmp/tlock" "$dir/tlock"`. Create
    `$dir` first if it doesn't exist.
11. **Symlink (case 4 only)** — if `needs_symlink` and `/usr/local/bin` is
    writable without sudo, `ln -sf "$dir/tlock" /usr/local/bin/tlock`.
    Otherwise skip silently (the PATH hint in step 12 covers it).
12. **Summary** — print `tlock vX.Y.Z installed to <dir>/tlock`. If `<dir>` is
    not on PATH, print an explicit suggestion like:
    ```
    Add this to your shell rc:
      export PATH="$HOME/.local/bin:$PATH"
    ```

## Error Messages

All errors print to stderr with a `tlock:` prefix and exit non-zero. Each
error names the problem and what the user should do next:

- **Not macOS** — `tlock: macOS only. Build from source: https://github.com/retr0h/tlock#-build-from-source`
- **Bad arch** — `tlock: unsupported architecture: <value>`
- **No curl/wget** — `tlock: neither curl nor wget found on PATH`
- **Download failed** — `tlock: failed to download <url>`
- **Checksum mismatch** — `tlock: checksum mismatch for tlock_<ver>_darwin_<arch>`
  followed by expected vs actual hashes.
- **Install dir write failure** — `tlock: cannot write to <dir>` with the
  actual error from `install`.

## README Rewrite

Replace the current `## 📦 Install` section. Keep the header and emoji style.
Lead with the one-liner, move the current manual-download and
verify-checksum blocks into a collapsed `<details>` disclosure, leave "Build
from source" unchanged.

New structure:

```markdown
## 📦 Install

```bash
curl -fsSL https://github.com/retr0h/tlock/raw/main/install.sh | sh
```

Installs to `~/.local/bin`, `~/bin`, or `/usr/local/bin` (root only).
Checksums are verified. Override with `TLOCK_INSTALL_DIR=/some/path` or
pin a version with `TLOCK_VERSION=1.1.1`.

<details>
<summary>Manual install</summary>

(existing per-arch curl snippets + verify-checksum block go here, unchanged)

</details>

### 🔨 Build from source

(unchanged)
```

## Testing

- **Lint** — add a `shellcheck install.sh` step as a new job in
  `.github/workflows/go.yml`. Runs on every PR.
- **Manual smoke test before merge** — on a clean macOS user:
  - `TLOCK_VERSION=1.1.1 sh install.sh` with `$HOME/.local/bin` on PATH →
    lands in `.local/bin`, is executable, `tlock --help` works.
  - Same with `$HOME/.local/bin` NOT on PATH → lands in `$HOME/.tlock/bin`,
    and the PATH hint is printed.
  - `TLOCK_INSTALL_DIR=/tmp/foo sh install.sh` → lands at `/tmp/foo/tlock`.
  - Corrupt the downloaded binary after step 7 (simulate by editing the
    script locally) → checksum failure aborts cleanly.
- **No automated end-to-end test.** A real-download test needs a macOS
  runner and hits GitHub release infra; cost outweighs value for a ~80-line
  shell script covered by shellcheck + manual smoke.

## Out of Scope

- Uninstaller.
- Auto-update.
- Homebrew tap.
- Linux support.
- GitHub Pages landing page.
- Publishing the installer as a release asset (URL stays on `main`, so the
  latest installer always reflects current behavior; version pinning is a
  script-level env var, not a URL-level concern).
