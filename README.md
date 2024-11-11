# osc
A command line tool to access the system clipboard from anywhere using the [ANSI OSC52](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Operating-System-Commands) sequence.

System clipboard access includes writing (i.e. copy) and reading (i.e. paste), even while logged into a remote machine via ssh.

## Examples

#### Copying to the clipboard

```
❯ echo -n asdf | osc copy
# String 'asdf' copied to clipboard
```

#### Pasting from the clipboard

```
❯ osc paste
asdf
```

#### Clearing the clipboard

```
❯ osc copy  /dev/null
# Clipboard cleared
```

## Usage

```
Reads or writes the system clipboard using the ANSI OSC52 escape sequence.

Usage:
  osc [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  copy        Copies input to the system clipboard
  help        Help about any command
  paste       Outputs system clipboard contents to stdout
  version     Outputs version information

Flags:
  -d, --device string   select device (default "/dev/tty")
  -h, --help            help for osc
  -l, --log string      write logs to file
  -t, --timeout float   tty read timeout in seconds (default 5)
  -v, --verbose         verbose logging

Use "osc [command] --help" for more information about a command.
```

## Compatibility

OSC52 is overall [widely supported](https://github.com/ojroques/vim-oscyank/blob/main/README.md#vim-oscyank), but clipboard read operation is less widely supported than write.

Terminal | Terminal OS | Shell OS | Copy | Paste | Notes
---      | ---         | ---      | ---  | ---   | ---
[alacritty](https://github.com/alacritty/alacritty) 0.13.1 | macOS | linux | &check; | &check; | Paste support requires [setting](https://alacritty.org/config-alacritty.html) `terminal.osc52` to `CopyPaste` or `OnlyPaste`
[alacritty](https://github.com/alacritty/alacritty) 0.13.1 | macOS | macOS | &check; | &check; | Paste support requires [setting](https://alacritty.org/config-alacritty.html) `terminal.osc52` to `CopyPaste` or `OnlyPaste`
[alacritty](https://github.com/alacritty/alacritty) 0.12.1 | Windows | linux | &check; | &cross; |
[alacritty](https://github.com/alacritty/alacritty) 0.13.2 | Windows | Windows | &check; | &cross; |
[kitty](https://github.com/kovidgoyal/kitty) 0.29.0 | macOS | linux | &check; | &check; | Prompts for access
[kitty](https://github.com/kovidgoyal/kitty) 0.29.0 | macOS | macOS | &check; | &check; | Prompts for access
[windows terminal](https://github.com/microsoft/terminal) v1.17.11461.0 | Windows | Windows | &check; | &cross; |
[windows terminal](https://github.com/microsoft/terminal) v1.17.11461.0 | Windows | linux | &check; | &cross; |
[iterm2](https://iterm2.com/) | macOS | linux | &check; | &check; | Paste requires version 3.5.0. Prompts for access.
[iterm2](https://iterm2.com/) | macOS | macOS | &check; | &check; | Paste requires version 3.5.0. Prompts for access.
[hterm](https://chrome.google.com/webstore/detail/secure-shell/iodihamcpbpeioajjeobimgagajmlibd) | ChromeOS | linux | &check; | &cross; |

#### Terminal Multiplexer support

Using [alacritty](https://github.com/alacritty/alacritty) as the terminal:

Terminal Multiplexer | Copy | Paste | Notes
---                  | ---  | ---   | ---
[screen](https://www.gnu.org/software/screen/) 4.09.00 | &check; | &check; |
[zellij](https://zellij.dev/) 0.37.2 | &check; | [&cross;](https://github.com/zellij-org/zellij/issues/2647) |
[tmux](https://github.com/tmux/tmux) 3.3 | &check; | &check; | [`allow-passthrough`](https://github.com/tmux/tmux/wiki/FAQ#what-is-the-passthrough-escape-sequence-and-how-do-i-use-it) (for copy) and `set-clipboard` (for paste) should be enabled

## Installation

```
go install -v github.com/theimpostor/osc@latest
```

This will install the latest version of osc to `$GOPATH/bin`. To find out where `$GOPATH` is, run `go env GOPATH`

## Neovim clipboard Provider

osc can be used as the clipboard provider for Neovim:

```lua
vim.cmd([[
let g:clipboard = {
  \   'name': 'osc-copy',
  \   'copy': {
  \      '+': 'osc copy',
  \      '*': 'osc copy',
  \    },
  \   'paste': {
  \      '+': 'osc paste',
  \      '*': 'osc paste',
  \   },
  \   'cache_enabled': 0,
  \ }
]])
```

N.B. Neovim 0.10 introduced native support for OSC52, so this may not be needed. See the [Neovim documentation](https://neovim.io/doc/user/provider.html#clipboard-osc52).

## Credits
-  [ojroques/vim-ocsyank](https://github.com/ojroques/vim-oscyank) - inspiration and introduction to OSC52
-  [rumpelsepp/oscclip](https://github.com/rumpelsepp/oscclip/tree/v0.4.1) - working python implementation
-  [gdamore/tcell](https://github.com/gdamore/tcell) - terminal handling
