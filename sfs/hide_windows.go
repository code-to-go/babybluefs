// +build windows

package sfs

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
