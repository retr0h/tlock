#!/usr/bin/env bash
#
# tlock installer
# Usage: curl -fsSL https://github.com/retr0h/tlock/raw/main/install.sh | bash
#
# Env overrides:
#   TLOCK_VERSION       install a specific version (e.g. 1.1.1) instead of latest
#   TLOCK_INSTALL_DIR   force install destination, skipping the default rules

set -euo pipefail
APP=tlock

# Visual style mirrors kvlt/grind's installer with tlock's accent.
# ACCENT is mhCyan from the shared maxheadroom palette (#00d4ff)
# used by the in-app TUI; truecolor (24-bit) escape so the install
# banner and the running app paint with the exact same hue.
MUTED='\033[0;2m'
RED='\033[0;31m'
ACCENT='\033[38;2;0;212;255m'
NC='\033[0m' # reset

err() {
    printf "${RED}tlock: %s${NC}\n" "$1" >&2
    exit 1
}

print_message() {
    local level=$1
    local message=$2
    local color=""
    case $level in
        info)    color="${NC}" ;;
        warning) color="${ACCENT}" ;;
        error)   color="${RED}" ;;
    esac
    printf "${color}${message}${NC}\n"
}

have() {
    command -v "$1" >/dev/null 2>&1
}

# unbuffered_sed picks the right -u/-l/no-buffer flag for the local
# sed; required so the progress reader sees curl trace lines as they
# arrive rather than after the download completes.
unbuffered_sed() {
    if echo | sed -u -e "" >/dev/null 2>&1; then
        sed -nu "$@"
    elif echo | sed -l -e "" >/dev/null 2>&1; then
        sed -nl "$@"
    else
        local pad
        pad="$(printf "\n%512s" "")"
        sed -ne "s/$/\\${pad}/" "$@"
    fi
}

print_progress() {
    local bytes="$1"
    local length="$2"
    [ "$length" -gt 0 ] || return 0

    local width=50
    local percent=$(( bytes * 100 / length ))
    [ "$percent" -gt 100 ] && percent=100
    local on=$(( percent * width / 100 ))
    local off=$(( width - on ))

    local filled
    filled=$(printf "%*s" "$on" "")
    filled=${filled// /■}
    local empty
    empty=$(printf "%*s" "$off" "")
    empty=${empty// /･}

    printf "\r${ACCENT}%s%s %3d%%${NC}" "$filled" "$empty" "$percent" >&4
}

# download_with_progress reads curl --trace-ascii output to drive a
# block-character progress bar. Falls back to plain curl/wget when:
#   - stderr is not a TTY (CI, piped to file)
#   - curl is unavailable (we use wget without progress)
#   - the trace plumbing fails for any reason
download_with_progress() {
    local url="$1"
    local output="$2"

    if [ -t 2 ]; then
        exec 4>&2
    else
        exec 4>/dev/null
    fi

    local tmp_dir=${TMPDIR:-/tmp}
    local basename="${tmp_dir}/tlock_install_$$"
    local tracefile="${basename}.trace"

    rm -f "$tracefile"
    mkfifo "$tracefile"

    # Hide cursor while the bar animates.
    printf "\033[?25l" >&4
    trap "trap - RETURN; rm -f \"$tracefile\"; printf '\033[?25h' >&4; exec 4>&-" RETURN

    (
        curl --trace-ascii "$tracefile" -fsSL -o "$output" "$url"
    ) &
    local curl_pid=$!

    unbuffered_sed \
        -e 'y/ACDEGHLNORTV/acdeghlnortv/' \
        -e '/^0000: content-length:/p' \
        -e '/^<= recv data/p' \
        "$tracefile" | \
    {
        local length=0
        local bytes=0

        while IFS=" " read -r -a line; do
            [ "${#line[@]}" -lt 2 ] && continue
            local tag="${line[0]} ${line[1]}"

            if [ "$tag" = "0000: content-length:" ]; then
                length="${line[2]}"
                length=$(echo "$length" | tr -d '\r')
                bytes=0
            elif [ "$tag" = "<= recv" ]; then
                local size="${line[3]}"
                bytes=$(( bytes + size ))
                if [ "$length" -gt 0 ]; then
                    print_progress "$bytes" "$length"
                fi
            fi
        done
    }

    wait $curl_pid
    local ret=$?
    echo "" >&4
    return $ret
}

http_get() {
    if have curl; then
        curl -fsSL "$1"
    elif have wget; then
        wget -qO- "$1"
    else
        err "neither curl nor wget found on PATH"
    fi
}

# fetch downloads $url to $output. The styled progress bar fires only
# when (1) curl is available, (2) stderr is a TTY, and (3) the caller
# opted in via "progress" as $3. Tiny side fetches (the checksums
# file) pass anything else; their bar would jump to 100% instantly
# and just spam scrollback.
fetch() {
    local url="$1"
    local output="$2"
    local mode="${3:-quiet}"
    if [ "$mode" = "progress" ] && have curl && [ -t 2 ]; then
        download_with_progress "$url" "$output" || curl -fsSL -o "$output" "$url"
    elif have curl; then
        curl -fsSL -o "$output" "$url"
    elif have wget; then
        wget -q -O "$output" "$url"
    else
        err "neither curl nor wget found on PATH"
    fi
}

detect_os() {
    raw=$(uname -s)
    if [ "$raw" != "Darwin" ]; then
        err "macOS only — tlock relies on Touch ID / LocalAuthentication. Build from source: https://github.com/retr0h/tlock#-build-from-source"
    fi
    os=darwin
}

detect_arch() {
    machine=$(uname -m)
    case "$machine" in
        arm64|aarch64) arch=arm64 ;;
        x86_64|amd64)  arch=amd64 ;;
        *)             err "unsupported architecture: $machine" ;;
    esac
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

path_contains() {
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

setup_tmp() {
    tmp=$(mktemp -d 2>/dev/null || mktemp -d -t tlock-install)
    trap 'rm -rf "$tmp"' EXIT
}

download() {
    base=https://github.com/retr0h/tlock/releases/download/v${version}
    asset=tlock_${version}_${os}_${arch}

    print_message info "\n${MUTED}Installing ${NC}tlock ${MUTED}version: ${NC}$version"
    fetch "$base/$asset" "$tmp/tlock" progress \
        || err "failed to download $base/$asset"
    fetch "$base/checksums.txt" "$tmp/checksums.txt" \
        || err "failed to download $base/checksums.txt"
}

verify_checksum() {
    asset=tlock_${version}_${os}_${arch}
    printf "${MUTED}Verifying checksum…${NC}\n"
    expected=$(grep " $asset\$" "$tmp/checksums.txt" | awk '{print $1}')
    if [ -z "$expected" ]; then
        err "no checksum entry for $asset in checksums.txt"
    fi
    if have shasum; then
        actual=$(shasum -a 256 "$tmp/tlock" | awk '{print $1}')
    elif have sha256sum; then
        actual=$(sha256sum "$tmp/tlock" | awk '{print $1}')
    else
        err "neither shasum nor sha256sum found on PATH"
    fi
    if [ "$expected" != "$actual" ]; then
        printf "${RED}tlock: checksum mismatch for %s${NC}\n  expected: %s\n  actual:   %s\n" \
            "$asset" "$expected" "$actual" >&2
        exit 1
    fi
}

strip_quarantine() {
    xattr -d com.apple.quarantine "$tmp/tlock" 2>/dev/null || true
}

install_binary() {
    mkdir -p "$install_dir" || err "cannot create $install_dir"
    install -m 755 "$tmp/tlock" "$install_dir/tlock" \
        || err "cannot write to $install_dir/tlock"
}

maybe_symlink() {
    [ "$needs_symlink" = "1" ] || return 0
    if [ -w /usr/local/bin ]; then
        ln -sf "$install_dir/tlock" /usr/local/bin/tlock 2>/dev/null || true
    fi
}

print_summary() {
    printf "\n"
    printf "${MUTED}▀█▀ █░░ █▀█ █▀▀ █░█${NC}   ${MUTED}installed to${NC} ${ACCENT}%s/tlock${NC}\n" "$install_dir"
    printf "${ACCENT}░█░ █▄▄ █▄█ █▄▄ █▀▄${NC}   ${MUTED}version${NC} ${NC}%s${NC}\n" "$version"
    printf "\n"
    if ! path_contains "$install_dir"; then
        print_message warning "Add this to your shell rc:"
        printf "  ${NC}export PATH=\"%s:\$PATH\"${NC}\n\n" "$install_dir"
    fi
    printf "${MUTED}Lock the terminal:${NC}\n"
    printf "  tlock                 ${MUTED}# Touch ID or password to unlock${NC}\n"
    printf "  tlock --pipes         ${MUTED}# pipes screensaver while locked${NC}\n"
    printf "\n"
    printf "${MUTED}Docs:${NC} https://github.com/retr0h/tlock\n"
    printf "\n"
}

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

main "$@"
