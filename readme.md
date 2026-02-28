# Fahhhh
This CLI tools integrates into terminals and plays the "fahhhh" sound whenever a command fails.

If you're paranoid about binaries running on your system:
- There are many vscode extensions about this exact thing that plays a fahhhh sound whenever a command fails.
- The code is there and it is built with github actions.

## Usage:
```
Usage: fahhhh <command>

The commands are:
  install    installs the utility for the current shell
  uninstall  uninstalls the utility for the current shell
  play       internal command to play the sound
```

##  Support
You'll need a x86_64 windows system.
We currently **only** natively support:
- Bash/Git bash
- Powershell

## For unsupported terminals
You can add a script to your terminal's .bashrc equivalent file:


Whenever the last command's exit code is non-zero, you can run:
`<path_to_fahhhh_binary> play`

You can see `main.go` for the hooks and take inspiration.

PRs are welcome to extend to more shells.