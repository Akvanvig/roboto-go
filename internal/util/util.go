package util

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

var RootPath string

func GetInt16Representation(bytes []byte) []int16 {
	return unsafe.Slice((*int16)(unsafe.Pointer(&bytes[0])), len(bytes)/2)
}

func GetCallingFuncFileName() string {
	_, fileName, _, _ := runtime.Caller(2)
	fileName = filepath.Base(fileName)
	fileName = strings.TrimSuffix(fileName, filepath.Ext(fileName))
	return fileName
}

// Note(Fredrico):
// See https://github.com/golang/go/issues/46345
func SendOSInterruptSignal() {
	pid := syscall.Getpid()

	switch runtime.GOOS {
	case "windows":
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
	default:
		process, _ := os.FindProcess(pid)
		process.Signal(os.Interrupt)
	}
}
