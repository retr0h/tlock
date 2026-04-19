#!/usr/bin/env bash
# tlock.tmux — entry point for the tlock tmux plugin.
#
# Install via TPM:
#   set -g @plugin 'retr0h/tlock'
# then <prefix> I
#
# Configurable via the user's tmux.conf:
#   set -g @tlock-binary   '/custom/path/to/tlock'  # override binary location
#                                                   # (default: ~/.local/bin/tlock,
#                                                   # falls back to $PATH lookup)
#   set -g @tlock-args     '--random --cycle 5m'    # flags passed to tlock
#   set -g @tlock-timeout  '1800'                   # lock-after-time in seconds
#                                                   # (0 disables auto-lock)
#   set -g @tlock-key      'C-x'                    # <prefix> + this key to
#                                                   # lock on demand

set -u

tmux_opt() {
    local name="$1" default="$2"
    local val
    val="$(tmux show-option -gqv "$name")"
    echo "${val:-$default}"
}

tlock_bin="$(tmux_opt '@tlock-binary' "$HOME/.local/bin/tlock")"
if [ ! -x "$tlock_bin" ]; then
    if command -v tlock >/dev/null 2>&1; then
        tlock_bin="$(command -v tlock)"
    else
        tmux display-message "tlock: binary not found at $tlock_bin or on \$PATH — see https://github.com/retr0h/tlock#-install"
        exit 0
    fi
fi

tlock_args="$(tmux_opt '@tlock-args' '--random --cycle 5m')"
tlock_timeout="$(tmux_opt '@tlock-timeout' '1800')"
tlock_key="$(tmux_opt '@tlock-key' 'C-x')"

# Own tmux's native lock machinery:
#   lock-command     — what tmux runs to lock (tlock + configured flags)
#   lock-after-time  — idle seconds before auto-lock (0 disables)
#   bind <key>       — manual `lock-server` invocation
tmux set-option -g lock-command "$tlock_bin $tlock_args"
tmux set-option -g lock-after-time "$tlock_timeout"
tmux bind-key "$tlock_key" lock-server
