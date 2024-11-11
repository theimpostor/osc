//go:build !windows
// +build !windows

package main

import (
	"github.com/gdamore/tcell/v2"
)

func opentty() (tty tcell.Tty, err error) {
	device := ttyDevice()
	debugLog.Println("Using tty device:", device)
	tty, err = tcell.NewDevTtyFromDev(device)
	if err == nil {
		err = tty.Start()
	}
	return
}
