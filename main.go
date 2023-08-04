package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/jba/slog/handlers/loghandler"
	"golang.org/x/exp/slog"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"runtime/debug"
)

var (
	oscOpen     string = "\x1b]52;c;"
	oscClose    string = "\a"
	isScreen    bool
	verboseFlag bool
	logfileFlag string
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
	tty, err = tcell.NewDevTty()
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
	logLevel := &slog.LevelVar{} // INFO
	logOutput := os.Stdout

	if logfileFlag != "" {
		if logOutput, err = os.OpenFile(logfileFlag, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644); err != nil {
			log.Fatalf("Failed to open file %v: %v", logfileFlag, err)
		} else {
			logfile = logOutput
		}
	}

	if verboseFlag {
		logLevel.Set(slog.LevelDebug)
	}

	logger := slog.New(loghandler.New(logOutput, &slog.HandlerOptions{
		Level: logLevel,
	}))

	slog.SetDefault(logger)
	slog.Debug("logging started")

	return
}

func identifyTerm() {
	if ti, err := tcell.LookupTerminfo(os.Getenv("TERM")); err != nil {
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
	}
}

func copy(fnames []string) error {
	// copy
	if len(fnames) == 0 {
		fnames = []string{"-"}
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
	slog.Debug("Beginning osc52 paste operation")
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

		// time out intial read in 100 milliseconds
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
		case <-time.After(100 * time.Millisecond):
			slog.Debug("tty read timeout")
			return nil, fmt.Errorf("tty read timeout")
		}

		for {
			if b, e := ttyReader.ReadByte(); e != nil {
				slog.Error(fmt.Sprintf("ReadByte: %v", e))
				return nil, e
			} else {
				slog.Debug(fmt.Sprintf("Read: %x '%s'", b, string(b)))
				// Terminator might be BEL (\a) or ESC-backslash (\x1b\\)
				if b == '\a' {
					break
				}
				buf = append(buf, b)
				// Skip initial 7 bytes of response
				if len(buf) > 9 && buf[len(buf)-2] == '\x1b' && buf[len(buf)-1] == '\\' {
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

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "verbose logging")
	rootCmd.PersistentFlags().StringVarP(&logfileFlag, "log", "l", "", "write logs to file")

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
