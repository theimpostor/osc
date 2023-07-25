package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/jba/slog/handlers/loghandler"
	"golang.org/x/exp/slog"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

var (
	oscOpen  string = "\x1b]52;c;"
	oscClose string = "\a"
	isScreen bool
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

func _main() int {
	var fnames []string
	var err error
	var exitCode int

	flag.Usage = func() {
		template := `Reads or writes the system clipboard using the ANSI OSC52 escape sequence.

Usage:

COPY mode (default):

    %s [file1 [...fileN]]

With no arguments, will read from stdin.

PASTE mode:

    %s --paste

Outputs clipboard contents to stdout.

Options:
`
		fmt.Fprintf(flag.CommandLine.Output(), template, os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}

	var pasteFlag bool
	var verboseFlag bool
	var logfileFlag string
	flag.BoolVar(&pasteFlag, "paste", pasteFlag, "paste operation")
	flag.BoolVar(&verboseFlag, "verbose", verboseFlag, "verbose logging")
	flag.BoolVar(&verboseFlag, "v", verboseFlag, "verbose logging")
	flag.StringVar(&logfileFlag, "logFile", logfileFlag, "redirect logs to file")

	flag.Parse()

	logLevel := &slog.LevelVar{} // INFO
	logOutput := os.Stdout

	if logfileFlag != "" {
		if logOutput, err = os.OpenFile(logfileFlag, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644); err != nil {
			log.Fatalf("Failed to open file %v: %v", logfileFlag, err)
		} else {
			defer logOutput.Close()
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

	if !pasteFlag {
		// copy
		if len(flag.Args()) > 0 {
			fnames = flag.Args()
		} else {
			fnames = []string{"-"}
		}

		slog.Debug("Beginning osc52 copy operation")
		func() {
			tty, err := opentty()
			if err != nil {
				slog.Error(fmt.Sprintf("opentty: %v", err))
				exitCode = 1
				return
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
		}()
		slog.Debug("Ended osc52")
	} else {
		// paste

		slog.Debug("Beginning osc52 paste operation")
		data := func() []byte {
			tty, err := opentty()
			if err != nil {
				slog.Error(fmt.Sprintf("opentty: %v", err))
				exitCode = 1
				return nil
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
				exitCode = 1
				return nil
			}

			for {
				if b, e := ttyReader.ReadByte(); e != nil {
					slog.Error(fmt.Sprintf("ReadByte: %v", e))
					exitCode = 1
					return nil
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
				exitCode = 1
				return nil
			}
			decodedBuf = decodedBuf[:n]

			return decodedBuf
		}()

		if data != nil {
			if _, err := os.Stdout.Write(data); err != nil {
				slog.Error(fmt.Sprintf("Error writing to stdout: %v", err))
				exitCode = 1
			}
		}
		slog.Debug("Ended osc52")
	}

	return exitCode
}

func main() {
	os.Exit(_main())
}
