package store

import (
	"bytes"
	"fmt"
	"github.com/jlaffaye/ftp"
	"io"
	"io/fs"
	"math"
	"os"
	"path"
	"strings"
	"time"
)

type FTPConfig struct {
	Addr     string        `json:"addr" yaml:"addr"`
	Username string        `json:"username" yaml:"username"`
	Password string        `json:"password" yaml:"password"`
	Base     string        `json:"base" yaml:"base"`
	Timeout  time.Duration `json:"timeout" yaml:"timeout"`
}

type FTP struct {
	c *ftp.ServerConn
}

func NewFTP(config FTPConfig) (FS, error) {
	var addr = config.Addr
	if !strings.ContainsRune(addr, ':') {
		addr = fmt.Sprintf("%s:21", addr)
	}
	c, err := ftp.Dial(addr, ftp.DialWithTimeout(config.Timeout))
	if err != nil {
		return nil, err
	}

	if err = c.Login(config.Username, config.Password); err != nil {
		return nil, err
	}

	return &FTP{c}, nil
}

func (f *FTP) Props() Props {
	return Props{
		Quota:       math.MaxInt64,
		Free:        math.MaxInt64,
		MinFileSize: 0,
		MaxFileSize: math.MaxInt64,
	}
}

func (f *FTP) MkdirAll(name string) error {
	if _, err := f.c.List(name); err == nil {
		return nil
	}

	p := ""
	for _, s := range strings.Split(name, "/") {
		p = path.Join(p, s)
		_ = f.c.MakeDir(p)
	}

	_, err := f.c.List(name)
	return err
}

func (f *FTP) mkParent(name string) error {
	dir := path.Dir(name)
	if dir == "" {
		return nil
	}
	return f.MkdirAll(dir)
}

func (f FTP) Pull(name string, w io.Writer) error {
	r, err := f.c.Retr(name)
	if err != nil {
		return err
	}
	defer r.Close()

	_, err = io.Copy(w, r)
	return err
}

func (f FTP) Push(name string, r io.Reader) error {
	err := f.mkParent(name)
	if err != nil {
		return err
	}
	err = f.Touch(name)
	if err != nil {
		return err
	}

	return f.c.Stor(name, r)
}

func (f *FTP) ReadDir(name string, opts ListOption) ([]fs.FileInfo, error) {
	entries, err := f.c.List(name)
	if err != nil {
		return nil, err
	}

	var fis []fs.FileInfo
	for _, e := range entries {
		n := e.Name
		if (opts&IncludeHiddenFiles) == 0 && strings.HasPrefix(n, ".") {
			continue
		}

		isDir := e.Type == ftp.EntryTypeFolder
		fis = append(fis, simpleFileInfo{
			name:    e.Name,
			size:    int64(e.Size),
			isDir:   isDir,
			modTime: e.Time,
		})
	}
	return fis, err
}

func (f *FTP) Watch(name string) chan string {
	return nil
}

func (f *FTP) Stat(name string) (fs.FileInfo, error) {
	entries, _ := f.c.List(name)
	switch len(entries) {
	case 0:
		return nil, os.ErrNotExist
	case 1:
		return simpleFileInfo{
			name:    path.Base(name),
			size:    int64(entries[0].Size),
			isDir:   false,
			modTime: entries[0].Time,
		}, nil
	default:
		return nil, os.ErrInvalid
	}
}

func (f *FTP) Remove(name string) error {
	return f.c.Delete(name)
}

func (f *FTP) Touch(name string) error {
	s, err := f.c.FileSize(name)
	if err != nil {
		return err
	}
	return f.c.StorFrom(name, bytes.NewReader(nil), uint64(s))
}

func (f *FTP) Rename(old, new string) error {
	_ = f.mkParent(new)
	return f.c.Rename(old, new)
}

func (f *FTP) Close() error {
	return f.c.Quit()
}
