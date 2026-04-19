# VHS tape: worms screensaver preview.
#
# Render from the repo root with:
#   vhs asset/worms.tape
#
# Produces asset/worms.gif — ~12s of worms wriggling around.
# Ctrl+C drops us out cleanly at the end so the GIF doesn't freeze on
# the passphrase prompt.

Output asset/worms.gif

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
Type "tlock --worms"
Enter
Sleep 800ms
Show

Sleep 12s

Ctrl+C
Sleep 300ms
