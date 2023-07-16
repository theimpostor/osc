package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/exp/slog"
	"io"
	"log"
	"os"
	// "golang.org/x/sys/unix"
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

func main() {
	var fnames []string
	var err error

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

	logger := slog.New(slog.NewTextHandler(logOutput, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)
	slog.Debug("logging started")

	if !pasteFlag {
		// copy
		if len(flag.Args()) > 0 {
			fnames = flag.Args()
		} else {
			fnames = []string{"-"}
		}

		tty, err := opentty()
		if err != nil {
			slog.Error("ERROR: opentty:", err)
			return
		}
		defer closetty(tty)

		// Open buffered output, using default max OSC52 length as buffer size
		// TODO limit size
		out := bufio.NewWriterSize(tty, 1000000)

		// Start OSC52
		slog.Debug("Beginning osc52 copy operation")
		fmt.Fprintf(out, "\033]52;c;")

		b64 := base64.NewEncoder(base64.StdEncoding, out)
		for _, fname := range fnames {
			encode(fname, b64)
		}
		b64.Close()

		// End OSC52
		fmt.Fprintf(out, "\a")

		out.Flush()
		slog.Debug("Ended osc52")
	} else {
		// paste

		data := func() []byte {

			tty, err := opentty()
			if err != nil {
				slog.Error("ERROR: opentty:", err)
				return nil
			}
			defer closetty(tty)

			// Start OSC52
			slog.Debug("Beginning osc52 paste operation")
			fmt.Fprintf(tty, "\033]52;c;?\a")

			ttyReader := bufio.NewReader(tty)
			buf := make([]byte, 0, 1024)
			for {
				if b, err := ttyReader.ReadByte(); err != nil {
					slog.Error("ReadByte error:", err)
					return nil
				} else {
					slog.Debug(fmt.Sprintf("Read: %x '%s'", b, string(b)))
					if b == '\a' {
						break
					}
					buf = append(buf, b)
					if len(buf) > 2 && buf[len(buf)-2] == '\033' && buf[len(buf)-1] == '\\' {
						buf = buf[:len(buf)-2]
						break
					}
				}
			}

			// buf, err := ttyReader.ReadBytes('\a')
			// if err != nil {
			// 	slog.Error("Read error:", err)
			// 	return nil
			// }

			// slog.Debug("Read %d bytes, %x\n", len(buf), buf)
			// fmt.Fprintf(os.Stderr, "buf[:7]: %q\n", buf[:7])
			// fmt.Fprintf(os.Stderr, "buf[len(buf)-1]: %q\n", buf[len(buf)-1])
			// fmt.Fprintf(os.Stderr, "%x\n", buf)
			buf = buf[7:]
			// fmt.Fprintf(os.Stderr, "%x\n", buf)

			dst := make([]byte, base64.StdEncoding.DecodedLen(len(buf)))
			n, err := base64.StdEncoding.Decode(dst, []byte(buf))
			if err != nil {
				slog.Error("decode error:", err)
				return nil
			}
			dst = dst[:n]

			return dst
		}()

		if data != nil {
			if _, err := os.Stdout.Write(data); err != nil {
				slog.Error("Error writing to stdout:", err)
			}
		}
		slog.Debug("Ended osc52")
	}
}
