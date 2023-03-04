//go:build linux

package util

import (
	"os"
	"syscall"
)

// Note(Fredrico):
// See https://github.com/golang/go/issues/46345
func SendOSInterruptSignal() {
	pid := syscall.Getpid()
	process, _ := os.FindProcess(pid)
	process.Signal(os.Interrupt)
}
