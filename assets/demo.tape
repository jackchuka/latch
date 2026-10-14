# Regenerate with: vhs assets/demo.tape (from the repository root)

Output assets/demo.gif

Set Shell "bash"
Set FontSize 22
Set Width 1440
Set Height 760
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms

Hide
Type `export D=/tmp/latch-demo && rm -rf "$D" && mkdir -p "$D" && go build -o "$D/latch" . && cp assets/demo/release.yaml "$D" && cd "$D" && export HOME="$D" XDG_CONFIG_HOME="$D/.config" XDG_DATA_HOME="$D/.local/share" PATH="$D:$PATH" PS1='\[\e[1;35m\]$ \[\e[0m\]' && clear`
Enter
Sleep 4s
Show

Type "cat release.yaml"
Sleep 400ms
Enter
Sleep 3s

Type "latch task add -f release.yaml && latch task run release"
Sleep 400ms
Enter
Sleep 1.5s

Type "latch queue list"
Sleep 400ms
Enter
Sleep 3s

Type "clear"
Enter
Type "ID=$(latch queue list -o json | jq -r '.[0].id') && latch queue show $ID"
Sleep 400ms
Enter
Sleep 3.5s

Type "latch queue approve $ID"
Sleep 400ms
Enter
Sleep 1.5s

Type "latch queue show $ID"
Sleep 400ms
Enter
Sleep 4s
