# osc52
A command line tool to copy text to the system clipboard from anywhere using the [ANSI OSC52](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Operating-System-Commands) sequence.

When this sequence is printed, the terminal will copy the given text into the system clipboard. This is totally location independent, users can copy from anywhere including from remote SSH sessions.

OSC52 is widely supported, see [partial list of supported terminals](https://github.com/ojroques/vim-oscyank/blob/main/README.md#vim-oscyank)

## Installation

#### go 1.16 or later:

```
go install -v github.com/theimpostor/osc52@latest
```

#### go 1.15 or earlier:
```
GO111MODULE=on go get github.com/theimpostor/osc52@latest
```

This will install the latest version of osc52 to `$GOPATH/bin`. To find out where `$GOPATH` is, run `go env GOPATH`

## Usage
```
Usage: ./osc52 [file1 [...fileN]]
Copies file contents to system clipboard using the OSC52 escape sequence.
With no arguments, will read from stdin.
```

## Credits
Credit and thanks to the the [vim-ocsyank](https://github.com/ojroques/vim-oscyank) plugin
