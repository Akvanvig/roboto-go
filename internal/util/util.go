package util

import (
	"unsafe"
)

func GetInt16Representation(bytes []byte) []int16 {
	return unsafe.Slice((*int16)(unsafe.Pointer(&bytes[0])), len(bytes)/2)
}
