// +build windows

package main

import (
	"syscall"
)


func hideFile(ph string) error {
	p, err := syscall.UTF16PtrFromString(ph)
	if err != nil {
		return err
	}

	return syscall.SetFileAttributes(p, syscall.FILE_ATTRIBUTE_HIDDEN)
}
