package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"io"
	"log"
	"os"
	"syscall"
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

func main() {
	var fnames []string

	flag.Usage = func() {
		template := `Usage:
%s [file1 [...fileN]]
Copies file contents to system clipboard using the ANSI OSC52 escape sequence.
With no arguments, will read from stdin.
`
		fmt.Fprintf(flag.CommandLine.Output(), template, os.Args[0])
	}

	pasteFlag := flag.Bool("paste", false, "paste operation")

	flag.Parse()
	if !*pasteFlag {
		// copy
		if len(flag.Args()) > 0 {
			fnames = flag.Args()
		} else {
			fnames = []string{"-"}
		}

		// Open buffered output, using default max OSC52 length as buffer size
		out := bufio.NewWriterSize(os.Stdout, 1000000)

		// Start OSC52
		fmt.Fprintf(out, "\033]52;c;")

		b64 := base64.NewEncoder(base64.StdEncoding, out)
		for _, fname := range fnames {
			encode(fname, b64)
		}
		b64.Close()

		// End OSC52
		fmt.Fprintf(out, "\a")

		out.Flush()
	} else {
		// paste

		// Start OSC52
		fmt.Printf("\033]52;c;?\a")

		readFunc := func() (data []byte) {
			var err error
			tty, err := tcell.NewDevTty()
			if err != nil {
				fmt.Fprintln(os.Stderr, "ERROR: tcell.NewDevTty:", err)
				return
			}
			defer func() {
				if err = tty.Drain(); err != nil {
					fmt.Fprintln(os.Stderr, "Drain error:", err)
				}
				if err = tty.Stop(); err != nil {
					fmt.Fprintln(os.Stderr, "Drain error:", err)
				}
				if err = tty.Close(); err != nil {
					fmt.Fprintln(os.Stderr, "Drain error:", err)
				}
			}()
			if err = tty.Start(); err != nil {
				fmt.Fprintln(os.Stderr, "Start error:", err)
			}

			ttyReader := bufio.NewReader(tty)
			data, err = ttyReader.ReadBytes('\a')
			if err != nil && err != syscall.EAGAIN {
				fmt.Fprintln(os.Stderr, "Read error:", err)
				return
			}
			return
		}
		buf := readFunc()
		// fmt.Fprintln(os.Stderr, "Read", len(buf), "bytes")
		fmt.Fprintf(os.Stderr, "Read %d bytes, %x\n", len(buf), buf)
		// fmt.Fprintln(os.Stderr, "Read:", string(buf[:]))

		// tty, err := os.OpenFile("/dev/tty", os.O_RDONLY, 0)
		// if err != nil {
		// 	fmt.Fprintln(os.Stderr, "Error opening /dev/tty:", err)
		// 	return
		// }
		// defer tty.Close()

		// // set nonblocking
		// err = unix.SetNonblock(int(tty.Fd()), true)
		// if err != nil {
		// 	fmt.Fprintln(os.Stderr, "Error setting nonblock:", err)
		// 	return
		// }

		// // Create a byte slice to read into
		// buf := make([]byte, 1)

		// for {
		// 	// Try to read
		// 	n, err := tty.Read(buf)
		// 	if err != nil && err != syscall.EAGAIN {
		// 		fmt.Fprintln(os.Stderr, "Read error:", err)
		// 		return
		// 	}

		// 	// If we got some data, print it
		// 	if n > 0 {
		// 		fmt.Fprintln(os.Stderr, "Read:", string(buf[:n]))

		// 		if buf[0] == byte('\a') {
		// 			fmt.Fprintln(os.Stderr, "END")
		// 			break
		// 		}
		// 	}
		// }
	}
}
