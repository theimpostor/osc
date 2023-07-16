# osc52
A command line tool to access the system clipboard from anywhere on the command line using the [ANSI OSC52](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Operating-System-Commands) sequence.

System clipboard access includes writing (i.e. copy) and reading (i.e. paste), even while logged into a remote machine via ssh.

OSC52 is widely supported, see [partial list of supported terminals](https://github.com/ojroques/vim-oscyank/blob/main/README.md#vim-oscyank). Note that clipboard read operation is less widely supported than write.

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
Reads or writes the system clipboard using the ANSI OSC52 escape sequence.

Usage:

COPY mode (default):

    osc52 [file1 [...fileN]]

With no arguments, will read from stdin.

PASTE mode:

    osc52 --paste

Outputs clipboard contents to stdout.

Options:
  -logFile string
    	redirect logs to file
  -paste
    	paste operation
  -v	verbose logging
  -verbose
    	verbose logging
```

## Credits
Credit and thanks to the the [vim-ocsyank](https://github.com/ojroques/vim-oscyank) plugin
