# osc52
A command line tool to copy text to the system clipboard from anywhere using the [ANSI OSC52](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Operating-System-Commands) sequence.

When this sequence is printed, the terminal will copy the given text into the system clipboard. This is totally location independent, users can copy from anywhere including from remote SSH sessions.

[List of supported terminals](https://github.com/ojroques/vim-oscyank/blob/main/README.md#vim-oscyank)

## Installation
```
go install -v github.com/theimpostor/osc52@latest
```

## Usage
```
Usage: ./osc52 [file1 [...fileN]]
Copies input to system clipboard using the OSC52 escape sequence.
With no file arguments, will read from stdin
```

## Credits
Credit and thanks to the the [vim-ocsyank](https://github.com/ojroques/vim-oscyank) plugin
