package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
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
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [file1 [...fileN]]\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Copies input to system clipboard using an OSC52 escape sequence.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Multiple files will be concatenated.\n")
		fmt.Fprintf(flag.CommandLine.Output(), "With no file arguments, will read from stdin\n")
	}

	flag.Parse()
	if len(flag.Args()) > 0 {
		fnames = flag.Args()
	} else {
		fnames = []string{"-"}
	}

	// Start OSC52
	fmt.Print("\033]52;c;")
	b64 := base64.NewEncoder(base64.StdEncoding, os.Stdout)
	for _, fname := range fnames {
		encode(fname, b64)
	}
	b64.Close()

	// End OSC52
	fmt.Print("\a")
}
