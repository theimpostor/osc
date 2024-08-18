package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jba/slog/handlers/loghandler"
	"github.com/mattn/go-isatty"

	"runtime/debug"

	"github.com/spf13/cobra"
)

var (
	oscOpen     string = "\x1b]52;c;"
	oscClose    string = "\a"
	isScreen    bool
	isTmux      bool
	isZellij    bool
	verboseFlag bool
	logfileFlag string
	deviceFlag  string
	timeoutFlag float64
)

func encode(fname string, encoder io.WriteCloser) {
	var f *os.File
	var err error

	if fname == "-" {
		f = os.Stdin
	} else {
		if f, err = os.Open(fname); err != nil {
			log.Fatalf("Failed to open file %v: %v", fname, err)
		} else {
			defer f.Close()
		}
	}

	if _, err = io.Copy(encoder, f); err != nil {
		log.Fatal(err)
	}
}

func opentty() (tty tcell.Tty, err error) {
	tty, err = tcell.NewDevTtyFromDev(deviceFlag)
	if err == nil {
		err = tty.Start()
	}
	return
}

func closetty(tty tcell.Tty) {
	tty.Drain()
	tty.Stop()
	tty.Close()
}

func initLogging() (logfile *os.File) {
	var err error
	logOutput := os.Stdout

	if logfileFlag != "" {
		if logOutput, err = os.OpenFile(logfileFlag, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644); err != nil {
			log.Fatalf("Failed to open file %v: %v", logfileFlag, err)
		} else {
			logfile = logOutput
		}
	}

	var opts *slog.HandlerOptions
	if verboseFlag {
		opts = &slog.HandlerOptions{Level: slog.LevelDebug}
	}

	logger := slog.New(loghandler.New(logOutput, opts))

	slog.SetDefault(logger)
	slog.Debug("logging started")

	return
}

func identifyTerm() {
	if os.Getenv("ZELLIJ") != "" {
		isZellij = true
	}
	if os.Getenv("TMUX") != "" {
		isTmux = true
	} else if ti, err := tcell.LookupTerminfo(os.Getenv("TERM")); err != nil {
		slog.Error(fmt.Sprintf("Failed to lookup terminfo: %v", err))
	} else {
		slog.Debug(fmt.Sprintf("term name: %s, aliases: %q", ti.Name, ti.Aliases))
		if strings.HasPrefix(ti.Name, "screen") {
			isScreen = true
		}
	}

	if isScreen {
		slog.Debug("Setting screen dcs passthrough")
		oscOpen = "\x1bP" + oscOpen
		oscClose = oscClose + "\x1b\\"
	} else if isTmux {
		slog.Debug("Setting tmux dcs passthrough")
		oscOpen = "\x1bPtmux;\x1b" + oscOpen
		oscClose = oscClose + "\x1b\\"
	}
}

func copy(fnames []string) error {
	// copy
	if len(fnames) == 0 {
		if isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd()) {
			return fmt.Errorf("nothing on stdin")
		}

		fnames = []string{"-"}
	} else {
		for _, fname := range fnames {
			if f, err := os.Open(fname); err != nil {
				return err
			} else {
				f.Close()
			}
		}
	}

	slog.Debug("Beginning osc52 copy operation")
	err := func() error {
		tty, err := opentty()
		if err != nil {
			slog.Error(fmt.Sprintf("opentty: %v", err))
			return err
		}
		defer closetty(tty)

		// Open buffered output, using default max OSC52 length as buffer size
		// TODO limit size
		out := bufio.NewWriterSize(tty, 1000000)

		// Start OSC52
		fmt.Fprint(out, oscOpen)

		b64 := base64.NewEncoder(base64.StdEncoding, out)
		for _, fname := range fnames {
			encode(fname, b64)
		}
		b64.Close()

		// End OSC52
		fmt.Fprint(out, oscClose)

		out.Flush()
		return nil
	}()
	slog.Debug("Ended osc52")
	return err
}

func paste() error {
	if isZellij {
		return fmt.Errorf("paste unsupported under zellij, unset ZELLIJ env var to force")
	}
	timeout := time.Duration(timeoutFlag*1_000_000_000) * time.Nanosecond
	slog.Debug(fmt.Sprintf("Beginning osc52 paste operation, timeout: %s", timeout))
	if data, err := func() ([]byte, error) {
		tty, err := opentty()
		if err != nil {
			slog.Error(fmt.Sprintf("opentty: %v", err))
			return nil, err
		}
		defer closetty(tty)

		// Start OSC52
		fmt.Fprint(tty, oscOpen+"?"+oscClose)

		ttyReader := bufio.NewReader(tty)
		buf := make([]byte, 0, 1024)

		// time out intial read
		readChan := make(chan byte, 1)
		defer close(readChan)
		go func() {
			if b, e := ttyReader.ReadByte(); e != nil {
				slog.Debug(fmt.Sprintf("Initial ReadByte error: %v", e))
			} else {
				readChan <- b
			}
		}()
		select {
		case b := <-readChan:
			buf = append(buf, b)
		case <-time.After(timeout):
			slog.Debug("tty read timeout")
			return nil, fmt.Errorf("tty read timeout")
		}

		for {
			if b, e := ttyReader.ReadByte(); e != nil {
				slog.Error(fmt.Sprintf("ReadByte: %v", e))
				return nil, e
			} else {
				slog.Debug(fmt.Sprintf("Read: %x %q", b, string(b)))
				// Terminator might be BEL (\a) or ESC-backslash (\x1b\\)
				if b == '\a' {
					break
				}
				buf = append(buf, b)
				// Skip initial 7 bytes of response
				if len(buf) >= 9 && buf[len(buf)-2] == '\x1b' && buf[len(buf)-1] == '\\' {
					buf = buf[:len(buf)-2]
					break
				}
			}
		}

		slog.Debug(fmt.Sprintf("buf[:7]: %q", buf[:7]))
		buf = buf[7:]

		decodedBuf := make([]byte, base64.StdEncoding.DecodedLen(len(buf)))
		n, err := base64.StdEncoding.Decode(decodedBuf, []byte(buf))
		if err != nil {
			slog.Error(fmt.Sprintf("decode error: %v", err))
			return nil, err
		}
		decodedBuf = decodedBuf[:n]

		return decodedBuf, nil
	}(); err != nil {
		return err
	} else {
		if _, err = os.Stdout.Write(data); err != nil {
			slog.Error(fmt.Sprintf("Error writing to stdout: %v", err))
			return err
		}
	}
	slog.Debug("Ended osc52")

	return nil
}

func closeSilently(f *os.File) {
	if f != nil {
		f.Close()
	}
}

var copyCmd = &cobra.Command{
	Use:   "copy",
	Short: "Copies input to the system clipboard",
	Long: `Copies input to the system clipboard. Usage:

osc copy [file1 [...fileN]]

With no arguments, will read from stdin.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logfile := initLogging()
		defer closeSilently(logfile)
		identifyTerm()
		return copy(args)
	},
}

var pasteCmd = &cobra.Command{
	Use:   "paste",
	Short: "Outputs system clipboard contents to stdout",
	Long: `Outputs system clipboard contents to stdout. Usage:

osc paste`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logfile := initLogging()
		defer closeSilently(logfile)
		identifyTerm()
		return paste()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Outputs version information",
	Long:  `Outputs version information`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if info, ok := debug.ReadBuildInfo(); !ok {
			fmt.Println(`Unable to obtain build info.`)
		} else {
			fmt.Println(info.Main.Version)
		}
	},
}

var rootCmd = &cobra.Command{
	Use:   "osc",
	Short: "Reads or writes the system clipboard using the ANSI OSC52 escape sequence",
	Long:  `Reads or writes the system clipboard using the ANSI OSC52 escape sequence.`,
}

func defaultDevice() string {
	sshtty := os.Getenv("SSH_TTY")
	if sshtty != "" {
		return sshtty
	}
	return "/dev/tty"
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "verbose logging")
	rootCmd.PersistentFlags().StringVarP(&logfileFlag, "log", "l", "", "write logs to file")
	rootCmd.PersistentFlags().StringVarP(&deviceFlag, "device", "d", defaultDevice(), "select device")
	rootCmd.PersistentFlags().Float64VarP(&timeoutFlag, "timeout", "t", 5, "tty read timeout in seconds")

	rootCmd.AddCommand(copyCmd)
	rootCmd.AddCommand(pasteCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
