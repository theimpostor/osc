//go:build windows
// +build windows

package main

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"os"
)

// stdIoTty is an implementation of the Tty API based upon stdin/stdout.
type stdIoTty struct {
	in  *os.File
	out *os.File
}

func (tty *stdIoTty) Read(b []byte) (int, error) {
	return tty.in.Read(b)
}

func (tty *stdIoTty) Write(b []byte) (int, error) {
	return tty.out.Write(b)
}

func (tty *stdIoTty) Close() error {
	return nil
}

func (tty *stdIoTty) Start() error {
	tty.in = os.Stdin
	tty.out = os.Stdout
	return nil
}

func (tty *stdIoTty) Drain() error {
	return nil
}

func (tty *stdIoTty) Stop() error {
	return nil
}

func (tty *stdIoTty) WindowSize() (tcell.WindowSize, error) {
	return tcell.WindowSize{}, fmt.Errorf("not implemented")
}

func (tty *stdIoTty) NotifyResize(cb func()) {
}

// NewStdioTty opens a tty using standard input/output.
func NewStdIoTty() (tcell.Tty, error) {
	tty := &stdIoTty{
		in:  os.Stdin,
		out: os.Stdout,
	}
	return tty, nil
}

func opentty() (tty tcell.Tty, err error) {
	debugLog.Println("Using stdio tty")
	tty, err = NewStdIoTty()
	return
}
