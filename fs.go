package main

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
	IncludeHiddenFiles ListOption = 1
)

type FS interface {
	ReadDir(path string, opts ListOption) ([]fs.FileInfo, error)

	Watch(name string) chan string

	Stat(name string) (fs.FileInfo, error)
	Remove(name string) error
	Touch(name string) error
	Rename(old, new string) error

	MkdirAll(name string) error
	Pull(name string, w io.Writer) error
	Push(name string, r io.Reader) error

	Close() error
	Props() Props
}

type Props struct {
	MaxRetention time.Time
	MinFileSize  int64
	MaxFileSize  int64
	Free         int64
	Quota        int64
}

type SimpleFileInfo struct {
	name    string
	size    int64
	isDir   bool
	modTime time.Time
}

func (f SimpleFileInfo) Name() string {
	return f.name
}

func (f SimpleFileInfo) Size() int64 {
	return f.size
}

func (f SimpleFileInfo) Mode() fs.FileMode {
	return 0644
}

func (f SimpleFileInfo) ModTime() time.Time {
	return f.modTime
}

func (f SimpleFileInfo) IsDir() bool {
	return f.isDir
}

func (f SimpleFileInfo) Sys() interface{} {
	return nil
}
