package common

import (
	"reflect"
	"unsafe"
)

func BytesToString(bytes []byte) string {
	var length = len(bytes)

	if length == 0 {
		return ""
	}

	var s string
	stringHeader := (*reflect.StringHeader)(unsafe.Pointer(&s))
	stringHeader.Data = uintptr(unsafe.Pointer(&bytes[0]))
	stringHeader.Len = length
	return s
}
