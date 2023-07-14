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

	tty, err := opentty()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: opentty:", err)
		return
	}
	defer closetty(tty)

	if !*pasteFlag {
		// copy
		if len(flag.Args()) > 0 {
			fnames = flag.Args()
		} else {
			fnames = []string{"-"}
		}

		// Open buffered output, using default max OSC52 length as buffer size
		// TODO limit size
		out := bufio.NewWriterSize(tty, 1000000)

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
		fmt.Fprintf(tty, "\033]52;c;?\a")

		ttyReader := bufio.NewReader(tty)

		buf, err := ttyReader.ReadBytes('\a')
		if err != nil {
			fmt.Fprintln(os.Stderr, "Read error:", err)
			return
		}

		// fmt.Fprintf(os.Stderr, "Read %d bytes, %x\n", len(buf), buf)
		// fmt.Fprintf(os.Stderr, "buf[:7]: %q\n", buf[:7])
		// fmt.Fprintf(os.Stderr, "buf[len(buf)-1]: %q\n", buf[len(buf)-1])
		// fmt.Fprintf(os.Stderr, "%x\n", buf)
		buf = buf[7 : len(buf)-1]
		// fmt.Fprintf(os.Stderr, "%x\n", buf)

		dst := make([]byte, base64.StdEncoding.DecodedLen(len(buf)))
		n, err := base64.StdEncoding.Decode(dst, []byte(buf))
		if err != nil {
			fmt.Fprintln(os.Stderr, "decode error:", err)
			return
		}
		dst = dst[:n]
		if _, err := os.Stdout.Write(dst); err != nil {
			fmt.Fprintln(os.Stderr, "Error writing to stdout:", err)
		}
	}
}
