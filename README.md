# osc
A command line tool to access the system clipboard from anywhere on the command line using the [ANSI OSC52](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Operating-System-Commands) sequence.

System clipboard access includes writing (i.e. copy) and reading (i.e. paste), even while logged into a remote machine via ssh.

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

## Compatibility

OSC52 is overall [widely supported](https://github.com/ojroques/vim-oscyank/blob/main/README.md#vim-oscyank), but clipboard read operation is less widely supported than write.

Terminal | Terminal OS | Shell OS | Copy | Paste | Notes
---      | ---         | ---      | ---  | ---   | ---
[alacritty](https://github.com/alacritty/alacritty) 0.12.2 | macOS | linux | &check; | &check; |
[alacritty](https://github.com/alacritty/alacritty) 0.12.2 | macOS | macOS | &check; | &check; |
[alacritty](https://github.com/alacritty/alacritty) 0.12.0 | Windows | linux | &check; | &cross; |
[kitty](https://github.com/kovidgoyal/kitty) 0.29.0 | macOS | linux | &check; | &check; | Prompts for access
[kitty](https://github.com/kovidgoyal/kitty) 0.29.0 | macOS | macOS | &check; | &check; | Prompts for access
[iterm2](https://iterm2.com/) | macOS | linux | &check; | &check; | Paste requires version 3.5.0 (currently beta). Prompts for access.
[iterm2](https://iterm2.com/) | macOS | macOS | &check; | &check; | Paste requires version 3.5.0 (currently beta). Prompts for access.
[hterm](https://chrome.google.com/webstore/detail/secure-shell/iodihamcpbpeioajjeobimgagajmlibd) | ChromeOS | linux | &check; | &cross; |

#### Terminal Multiplexer support

Using [alacritty](https://github.com/alacritty/alacritty) as the terminal,
default configuration options for multiplexer:

Terminal Multiplexer | Copy | Paste | Notes
---                  | ---  | ---   | ---
[screen](https://www.gnu.org/software/screen/) 4.09.00 | &check; | &check; |
[zellij](https://zellij.dev/) 0.37.2 | &check; | [&cross;](https://github.com/zellij-org/zellij/issues/2647) |
[tmux](https://github.com/tmux/tmux) 3.2a | &cross; | &cross; | May require configuration

## Installation

#### go 1.16 or later:

```
go install -v github.com/theimpostor/osc@latest
```

#### go 1.15 or earlier:
```
GO111MODULE=on go get github.com/theimpostor/osc@latest
```

This will install the latest version of osc to `$GOPATH/bin`. To find out where `$GOPATH` is, run `go env GOPATH`

## TODO

- [ ] [copy] check is stdin is readable
- [ ] tmux support

## Credits
-  [ojroques/vim-ocsyank](https://github.com/ojroques/vim-oscyank) - inspiration and introduction to OSC52
-  [rumpelsepp/oscclip](https://github.com/rumpelsepp/oscclip) - working python implementation
-  [gdamore/tcell](https://github.com/gdamore/tcell) - terminal handling
