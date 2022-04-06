//go:build !windows
// +build !windows

package store

func hideFile(p string) error {
	return nil
}
