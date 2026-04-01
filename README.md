[![release](https://img.shields.io/github/release/retr0h/tlock.svg?style=for-the-badge)](https://github.com/retr0h/tlock/releases/latest)
[![go report card](https://goreportcard.com/badge/github.com/retr0h/tlock?style=for-the-badge)](https://goreportcard.com/report/github.com/retr0h/tlock)
[![license](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge)](LICENSE)
[![build](https://img.shields.io/github/actions/workflow/status/retr0h/tlock/go.yml?style=for-the-badge)](https://github.com/retr0h/tlock/actions/workflows/go.yml)
[![release](https://img.shields.io/github/actions/workflow/status/retr0h/tlock/release.yml?style=for-the-badge&label=release)](https://github.com/retr0h/tlock/actions/workflows/release.yml)
[![powered by](https://img.shields.io/badge/powered%20by-goreleaser-green.svg?style=for-the-badge)](https://github.com/goreleaser)
[![just](https://img.shields.io/badge/just-command%20runner-blue?style=for-the-badge)](https://github.com/casey/just)
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

<p align="center">
  <a href="asset/passphrase.png"><img src="asset/passphrase.png" width="400" alt="Passphrase Prompt"></a>
  <a href="asset/touchid.png"><img src="asset/touchid.png" width="400" alt="Touch ID Prompt"></a>
</p>

## ✨ Features

- 🖐️ **Touch ID** fingerprint unlock via macOS LocalAuthentication
- 🔑 **macOS password** fallback with blinking block cursor
- 🎨 **Glitch-style** unicode bordered prompts (purple/teal palette)
- 🧠 **Auto-detects** Touch ID availability (skips when lid is closed)
- 🛡️ **Signal-proof** — Ctrl+C, Ctrl+Z won't bypass the lock
- 📐 **Terminal resize** aware
- 🖥️ Designed as a **tmux** `lock-command`

## 📦 Install

> **Note:** tlock requires CGo and macOS frameworks (LocalAuthentication, PAM),
> so it must be built from source on a Mac.

```bash
git clone https://github.com/retr0h/tlock.git
cd tlock
go build -o tlock .
sudo mv tlock /usr/local/bin/
```

## 🚀 Usage

Run directly:

```bash
tlock                                        # Password prompt only
tlock --snake                                # Worms immediately
tlock --screensaver                          # Worms after 30s idle (default delay)
tlock --screensaver --screensaver-delay 60   # Worms after 1 min idle
```

As a tmux lock command:

```tmux
# ~/.tmux.conf
set -g lock-command "tlock --snake"
set -g lock-after-time 1800    # Lock after 30 min idle
bind ^X lock-server            # Ctrl+X to lock now
```

## ⚙️ How It Works

1. Terminal locks and shows the passphrase prompt with a blinking cursor
2. Type your macOS password and press Enter to unlock
3. Press **Esc** to switch to Touch ID — authenticate with your fingerprint
4. Wrong password? **ACCESS DENIED** — back to the prompt. Try again.

All signals (SIGINT, SIGTERM, SIGTSTP) are ignored. The only way out is authentication. 🔐

## 📋 Requirements

- 🍎 macOS (uses LocalAuthentication.framework and PAM)
- 🐹 Go 1.21+ with CGo enabled
- 🖐️ Touch ID hardware (optional — password fallback always available)

## 💡 Inspiration

tlock is inspired by [xlock](https://linux.die.net/man/1/xlock), the classic X11 screen locker from the 90s that shipped with most Unix workstations. The worm screensaver mode (`xlock -mode worm`) by David Bagley was a staple of SGI Indigos and Sun workstations in dimly lit server rooms everywhere.

## 🔀 Alternatives

| Tool | Platform | Description |
|------|----------|-------------|
| [xlock / xlockmore](https://github.com/zevlg/xlockmore) | X11 / Unix | The OG screen locker with 50+ screensaver modes |
| [vlock](https://github.com/hwhw/vlock) | Linux | Virtual console lock — locks Linux TTYs |
| [bashlock](https://github.com/njhartwell/bashlock) | macOS / Linux | Simple bash-based terminal lock |
| [slock](https://tools.suckless.org/slock/) | X11 | Suckless screen locker — minimal, no frills |

## 🗺️ Roadmap

- [ ] 🐛 xlock-style worm screensaver with fading trails
- [ ] 🔤 Cycling figurine text screensaver
- [ ] ⚙️ Configuration file (`~/.config/tlock/config.yaml`)
- [ ] 🔐 1Password CLI integration

## 📄 License

[MIT](LICENSE) - John Dewey
