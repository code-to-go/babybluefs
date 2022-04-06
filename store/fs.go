package store

import (
	"errors"
	"io"
	"io/fs"
	"time"
)

type Group string

var ErrNotSupported = errors.New("operation is not supported")
var ErrOffQuota = errors.New("quota exceeded")

type ListOption uint32

const (
	// IncludeHiddenFiles includes hidden files in a list operation
	IncludeHiddenFiles ListOption = 1
)

/**
A file storage interface
*/
type FS interface {
	// ReadDir gets the list of files in the provided path. Use IncludeHiddenFiles to include hidden files in the result
	ReadDir(path string, opts ListOption) ([]fs.FileInfo, error)

	// Watch reacts to change in the file storage
	Watch(name string) chan string

	// Stat gets information about a file
	Stat(name string) (fs.FileInfo, error)

	// Remove deletes a file
	Remove(name string) error

	// Touch updates the modification time of a file. On some storage (e.g. S3) this operation is not supported
	Touch(name string) error

	// Rename changes the name from a old name to a new one
	Rename(old, new string) error

	// MkdirAll creates a folder and all required intermediate folders
	MkdirAll(name string) error

	// Pull reads the file name and writes the content in w
	Pull(name string, w io.Writer) error

	// Push writes the file name by using data coming from r
	Push(name string, r io.Reader) error

	// Close realises resources
	Close() error

	// Props returns properties specific to the file storage
	Props() Props
}

type Props struct {
	// Retention is minimum time before a file is deleted since its last change
	Retention time.Time
	// MinFileSize is the smallest size a file can have
	MinFileSize int64
	// MaxFileSize is the biggest size a file can have
	MaxFileSize int64
	// Free is the available space in bytes
	Free int64
	// Quota is the maximal possible amount of bytes
	Quota int64
}

type simpleFileInfo struct {
	name    string
	size    int64
	isDir   bool
	modTime time.Time
}

func (f simpleFileInfo) Name() string {
	return f.name
}

func (f simpleFileInfo) Size() int64 {
	return f.size
}

func (f simpleFileInfo) Mode() fs.FileMode {
	return 0644
}

func (f simpleFileInfo) ModTime() time.Time {
	return f.modTime
}

func (f simpleFileInfo) IsDir() bool {
	return f.isDir
}

func (f simpleFileInfo) Sys() interface{} {
	return nil
}
