package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-isatty"

	"runtime/debug"

	"github.com/spf13/cobra"
)

const (
	ESC       = '\x1b'
	BEL       = '\a'
	BS        = '\\'
	OSC       = string(ESC) + "]52;"
	DCS_OPEN  = string(ESC) + "P"
	DCS_CLOSE = string(ESC) + string(BS)
)

var (
	oscOpen     string = OSC + "c;"
	oscClose    string = string(BEL)
	isScreen    bool
	isTmux      bool
	isZellij    bool
	verboseFlag bool
	logfileFlag string
	deviceFlag  string
	timeoutFlag float64
	debugLog    *log.Logger
	errorLog    *log.Logger
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

func closetty(tty tcell.Tty) {
	_ = tty.Drain()
	_ = tty.Stop()
	tty.Close()
}

// log levels to handle:
// debug
// error
// discarding logger: myLogger = log.New(io.Discard, "", 0)
// printing methods:
// Print: multiple args, adds space between non-string arguments
// Printf: first arg format, rest args
// Println multiple args, always adds space between args and a newline
// all print functions add a new line if absent
// File io.Writer is 'safe for concurrent use'
// Lmsgefix                    // move the "prefix" from the beginning of the line to before the message
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

	log.SetOutput(logOutput)
	errorLog = log.New(logOutput, "ERROR ", log.LstdFlags|log.Lmsgprefix)
	if verboseFlag {
		debugLog = log.New(logOutput, "DEBUG ", log.LstdFlags|log.Lmsgprefix)
	} else {
		debugLog = log.New(io.Discard, "", 0)
	}

	debugLog.Println("logging started")

	return
}

func identifyTerm() {
	if os.Getenv("ZELLIJ") != "" {
		isZellij = true
	}
	if os.Getenv("TMUX") != "" {
		isTmux = true
	} else if ti, err := tcell.LookupTerminfo(os.Getenv("TERM")); err != nil {
		if runtime.GOOS != "windows" {
			errorLog.Println("Failed to lookup terminfo:", err)
		} else {
			debugLog.Println("On Windows, failed to lookup terminfo:", err)
		}
	} else {
		debugLog.Printf("term name: %s, aliases: %q", ti.Name, ti.Aliases)
		if strings.HasPrefix(ti.Name, "screen") {
			isScreen = true
		}
	}

	if isScreen {
		debugLog.Println("Setting screen dcs passthrough")
		oscOpen = DCS_OPEN + oscOpen
		oscClose = oscClose + DCS_CLOSE
	} else if isTmux {
		debugLog.Println("Setting tmux dcs passthrough")
		oscOpen = DCS_OPEN + "tmux;" + string(ESC) + oscOpen
		oscClose = oscClose + DCS_CLOSE
	}
}

// Inserts screen dcs end + start sequence into long sequences
// Based on: https://github.com/chromium/hterm/blob/6846a85f9579a8dfdef4405cc50d9fb17d8944aa/etc/osc52.sh#L23
const chunkSize = 76

type chunkingWriter struct {
	bytesWritten int64
	writer       io.Writer
}

func (w *chunkingWriter) Write(p []byte) (n int, err error) {
	debugLog.Println("chunkingWriter got", len(p), "bytes")

	for err == nil && len(p) > 0 {
		bytesWritten := 0
		chunksWritten := w.bytesWritten / chunkSize
		nextChunkBoundary := (chunksWritten + 1) * chunkSize

		if w.bytesWritten+int64(len(p)) < nextChunkBoundary {
			bytesWritten, err = w.writer.Write(p)
		} else {
			bytesWritten, err = w.writer.Write(p[:nextChunkBoundary-w.bytesWritten])
			if err == nil {
				_, err = w.writer.Write([]byte(DCS_CLOSE + DCS_OPEN))
			}
		}
		w.bytesWritten += int64(bytesWritten)
		n += bytesWritten
		p = p[bytesWritten:]
	}

	return
}

func copy(fnames []string) error {
	// copy
	if isTmux {
		if out, err := exec.Command("tmux", "show", "-v", "allow-passthrough").Output(); err != nil {
			return fmt.Errorf("error running 'tmux show -v allow-passthrough': %w", err)
		} else {
			outStr := strings.TrimSpace(string(out))
			debugLog.Println("'tmux show -v allow-passthrough':", outStr)
			if outStr != "on" && outStr != "all" {
				return fmt.Errorf("tmux allow-passthrough must be set to 'on' or 'all'")
			}
		}
	}
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

	debugLog.Println("Beginning osc52 copy operation")
	err := func() error {
		tty, err := opentty()
		if err != nil {
			errorLog.Println("opentty:", err)
			return err
		}
		defer closetty(tty)

		// Open buffered output, using default max OSC52 length as buffer size
		out := bufio.NewWriterSize(tty, 1000000)

		// Start OSC52
		fmt.Fprint(out, oscOpen)

		var b64 io.WriteCloser
		if !isScreen {
			b64 = base64.NewEncoder(base64.StdEncoding, out)
		} else {
			b64 = base64.NewEncoder(base64.StdEncoding, &chunkingWriter{writer: out})
		}
		for _, fname := range fnames {
			encode(fname, b64)
		}
		b64.Close()

		// End OSC52
		fmt.Fprint(out, oscClose)

		out.Flush()
		return nil
	}()
	debugLog.Println("Ended osc52")
	return err
}

func tmux_paste() error {
	if out, err := exec.Command("tmux", "show", "-v", "set-clipboard").Output(); err != nil {
		return fmt.Errorf("Error running 'tmux show -v set-clipboard': %w", err)
	} else {
		outStr := strings.TrimSpace(string(out))
		debugLog.Println("'tmux show -v set-clipboard':", outStr)
		if outStr != "on" && outStr != "external" {
			return fmt.Errorf("tmux set-clipboard must be set to 'on' or 'external'")
		}
	}
	// refresh client list
	if out, err := exec.Command("tmux", "refresh-client", "-l").Output(); err != nil {
		return fmt.Errorf("Error running 'tmux refresh-client -l': %v", err)
	} else {
		debugLog.Println("tmux refresh-client output:", string(out))
	}
	// give terminal time to sync
	// https://github.com/rumpelsepp/oscclip/blob/6a4847ed5497baa9a9357b389f492f5d52c6867c/oscclip/__init__.py#L73
	time.Sleep(50 * time.Millisecond)
	if out, err := exec.Command("tmux", "save-buffer", "-").Output(); err != nil {
		return fmt.Errorf("error running 'tmux save-buffer -': %v", err)
	} else if _, err := os.Stdout.Write(out); err != nil {
		errorLog.Println("Error writing to stdout:", err)
		return err
	}
	return nil
}

// wraps an io.Reader, reads until it encounters an ESC or BEL
type pasteReader struct {
	r io.Reader
}

func (pr *pasteReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	if i := bytes.IndexByte(p, BEL); i >= 0 {
		return i, io.EOF
	}
	if i := bytes.IndexByte(p, ESC); i >= 0 {
		if i+1 == n {
			// closing sequence is ESC+BS
			// read and discard one more byte
			b := make([]byte, 1)
			if _, err = pr.r.Read(b); err != nil {
				return i, err
			}
		}
		return i, io.EOF
	}
	return n, err
}

type debugReader struct {
	prefix string
	r      io.Reader
}

func (dr *debugReader) Read(p []byte) (int, error) {
	n, err := dr.r.Read(p)
	debugLog.Printf("%s: %v %v %q", dr.prefix, n, err, p[:n])
	return n, err
}

func paste() error {
	if isTmux {
		return tmux_paste()
	} else if isZellij {
		return fmt.Errorf("paste unsupported under zellij, unset ZELLIJ env var to force")
	}
	timeout := time.Duration(timeoutFlag*1_000_000_000) * time.Nanosecond
	debugLog.Println("Beginning osc52 paste operation, timeout:", timeout)

	if data, err := func() ([]byte, error) {
		tty, err := opentty()
		if err != nil {
			return nil, fmt.Errorf("Error opening tty: %w", err)
		}
		defer closetty(tty)

		// Start OSC52
		fmt.Fprint(tty, oscOpen+"?"+oscClose)

		var ttyReader *bufio.Reader
		if verboseFlag {
			ttyReader = bufio.NewReader(&debugReader{
				prefix: "tty read",
				r:      tty,
			})
		} else {
			ttyReader = bufio.NewReader(tty)
		}

		// Define a struct to hold the read bytes and any error
		type readResult struct {
			data []byte
			err  error
		}

		// Time out initial read
		readChan := make(chan readResult, 1)
		defer close(readChan)

		go func() {
			b, e := ttyReader.ReadSlice(';')
			readChan <- readResult{data: b, err: e}
		}()

		select {
		case res := <-readChan:
			if res.err != nil {
				return nil, fmt.Errorf("Initial ReadSlice error: %w", res.err)
			} else if !bytes.Equal(res.data, []byte(OSC)) {
				return nil, fmt.Errorf("osc header mismatch: %q", res.data)
			}
		case <-time.After(timeout):
			return nil, fmt.Errorf("tty read timeout")
		}

		// ignore clipboard info
		if _, e := ttyReader.ReadSlice(';'); e != nil {
			return nil, fmt.Errorf("Clipboard metadata ReadSlice error: %w", e)
		}

		pr := pasteReader{r: ttyReader}
		decoder := base64.NewDecoder(base64.StdEncoding, &pr)
		if data, err := io.ReadAll(decoder); err != nil {
			return nil, fmt.Errorf("Error reading from decoder: %w", err)
		} else {
			return data, nil
		}
	}(); err != nil {
		return err
	} else if _, err := os.Stdout.Write(data); err != nil {
		return fmt.Errorf("Error writing to stdout: %w", err)
	}

	debugLog.Println("Ended osc52")

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

func ttyDevice() string {
	if deviceFlag != "" {
		return deviceFlag
	} else if isScreen {
		return "/dev/tty"
	} else if sshtty := os.Getenv("SSH_TTY"); sshtty != "" {
		return sshtty
	} else {
		return "/dev/tty"
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "verbose logging")
	rootCmd.PersistentFlags().StringVarP(&logfileFlag, "log", "l", "", "write logs to file")
	rootCmd.PersistentFlags().StringVarP(&deviceFlag, "device", "d", "", "use specific tty device")
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
