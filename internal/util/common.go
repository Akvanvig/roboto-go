package util

import (
	"path/filepath"
	"runtime"
	"strings"
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
