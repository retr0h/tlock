[![release](https://img.shields.io/github/release/retr0h/tlock.svg?style=for-the-badge)](https://github.com/retr0h/tlock/releases/latest)
[![go report card](https://goreportcard.com/badge/github.com/retr0h/tlock?style=for-the-badge)](https://goreportcard.com/report/github.com/retr0h/tlock)
[![license](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge)](LICENSE)
[![build](https://img.shields.io/github/actions/workflow/status/retr0h/tlock/go.yml?style=for-the-badge)](https://github.com/retr0h/tlock/actions/workflows/go.yml)
[![powered by](https://img.shields.io/badge/powered%20by-goreleaser-green.svg?style=for-the-badge)](https://github.com/goreleaser)
[![conventional commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-yellow.svg?style=for-the-badge)](https://conventionalcommits.org)
![macOS](https://img.shields.io/badge/macOS-000000?style=for-the-badge&logo=apple&logoColor=white)
[![go reference](https://img.shields.io/badge/go-reference-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://pkg.go.dev/github.com/retr0h/tlock)
![github commit activity](https://img.shields.io/github/commit-activity/m/retr0h/tlock?style=for-the-badge)

# tlock

```
____________   _____________________
7      77  7   7     77     77  7  7
!__  __!|  |   |  7  ||  ___!|   __!
  7  7  |  !___|  |  ||  7___|     |
  |  |  |     7|  !  ||     7|  7  |
  !__!  !_____!!_____!!_____!!__!__!
```

🔒 Lock your terminal. Unlock with your fingerprint.

A terminal lock screen for macOS that uses **Touch ID** for biometric unlock with **macOS password** fallback. Drop it into tmux as your `lock-command` and walk away.

## ✨ Features

- 🖐️ **Touch ID** fingerprint unlock via macOS LocalAuthentication
- 🔑 **macOS password** fallback with blinking block cursor
- 🎨 **Glitch-style** unicode bordered prompts (purple/teal palette)
- 🧠 **Auto-detects** Touch ID availability (skips when lid is closed)
- 🛡️ **Signal-proof** — Ctrl+C, Ctrl+Z won't bypass the lock
- 📐 **Terminal resize** aware
- 🖥️ Designed as a **tmux** `lock-command`

## 📦 Install

```bash
go install github.com/retr0h/tlock@latest
```

Or build from source:

```bash
git clone https://github.com/retr0h/tlock.git
cd tlock
go build -o tlock .
```

## 🚀 Usage

Run directly:

```bash
tlock
```

As a tmux lock command:

```tmux
# ~/.tmux.conf
set -g lock-command "tlock"
set -g lock-after-time 1800    # Lock after 30 min idle
bind ^X lock-server            # Ctrl+X to lock now
```

## ⚙️ How It Works

1. Terminal clears, enters raw mode, shows the lock screen
2. Press any key to begin unlock
3. 🖐️ Touch ID prompt appears — authenticate with your fingerprint
4. If Touch ID fails or is unavailable, falls back to 🔑 password prompt
5. Wrong password? Back to the lock screen. Try again.

All signals (SIGINT, SIGTERM, SIGTSTP) are ignored. The only way out is authentication. 🔐

## 📋 Requirements

- 🍎 macOS (uses LocalAuthentication.framework and PAM)
- 🐹 Go 1.21+ with CGo enabled
- 🖐️ Touch ID hardware (optional — password fallback always available)

## 🗺️ Roadmap

- [ ] 🐛 xlock-style worm screensaver with fading trails
- [ ] 🔤 Cycling figurine text screensaver
- [ ] ⚙️ Configuration file (`~/.config/tlock/config.yaml`)
- [ ] 🔐 1Password CLI integration

## 📄 License

[MIT](LICENSE) - John Dewey
