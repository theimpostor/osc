package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
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
		template := `
Usage:
%s [file1 [...fileN]]
Copies file contents to system clipboard using the ANSI OSC52 escape sequence.
With no arguments, will read from stdin.`
		fmt.Fprintf(flag.CommandLine.Output(), strings.TrimSpace(template), os.Args[0])
	}

	flag.Parse()
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
}
