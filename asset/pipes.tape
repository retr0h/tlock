# VHS tape: pipes screensaver preview.
#
# Render from the repo root with:
#   vhs asset/pipes.tape
#
# Produces asset/pipes.gif — ~12s of pipes growing across the screen.
# Ctrl+C drops us out cleanly at the end so the GIF doesn't freeze on
# the passphrase prompt.

Output asset/pipes.gif

Set Shell "zsh"
Set FontSize 14
Set Width 960
Set Height 540
Set Padding 12
Set TypingSpeed 10ms

Hide
Type "source ~/.zshrc"
Enter
Sleep 3s
Type "clear"
Enter
Sleep 200ms
Type "tlock --pipes"
Enter
Sleep 800ms
Show

Sleep 12s

Ctrl+C
Sleep 300ms
