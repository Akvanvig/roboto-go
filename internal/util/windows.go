//go:build windows

package util

import (
	"os"
	"syscall"
)

// Note(Fredrico):
// See https://github.com/golang/go/issues/46345
func SendOSInterruptSignal() {
	pid := syscall.Getpid()
	dll, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		os.Exit(1)
	}
	procedure, err := dll.FindProc("GenerateConsoleCtrlEvent")
	if err != nil {
		os.Exit(1)
	}
	result, _, _ := procedure.Call(syscall.CTRL_BREAK_EVENT, uintptr(pid))
	if result == 0 {
		os.Exit(1)
	}
}
