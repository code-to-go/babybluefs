package main

import (
	"github.com/sirupsen/logrus"
	"io"
	"io/fs"
	"io/ioutil"
	"math"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type Local struct {
	Mount string
	Perm fs.FileMode
}

func (l *Local) realPath(name string) string {
	return filepath.Join(l.Mount, filepath.FromSlash(name))
}

func isUnixHidden(name string) bool {
	base := filepath.Base(name)
	return base[0] == '.'
}

func dirPerm(perm fs.FileMode) fs.FileMode {
	if perm & 0600 != 0 {
		perm |= 0100
	}
	if perm & 0060 != 0 {
		perm |= 0010
	}
	if perm & 0006 != 0 {
		perm |= 0001
	}
	return perm
}

func (l *Local) Rename(old, new string) error {
	old = l.realPath(old)
	new =  l.realPath(new)
	err := os.MkdirAll(path.Dir(new), 0755)
	if err != nil {
		logrus.Errorf("Cannot create parent folder for %s: %v", new, err)
		return err
	}

	err = os.Rename(old, new)
	if err != nil {
		logrus.Errorf("Cannot move file to %s: %v", new, err)
		return err
	}
	if isUnixHidden(new) {
		_ = hideFile(new)
	}
	return nil
}

func NewLocal(mount string, perm fs.FileMode) FS {
	mount, _ = filepath.Abs(mount)
	return &Local{mount, perm}
}

func (l *Local) Props() Props {
	return Props{
		Quota:       math.MaxInt64,
		Free:        math.MaxInt64,
		MinFileSize: 0,
		MaxFileSize: math.MaxInt64,
	}
}

func (l *Local) Pull(name string, w io.Writer) error {
	name = l.realPath(name)
	f, err := os.OpenFile(name, os.O_RDONLY, l.Perm)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	return err
}

func (l *Local) Push(name string, r io.Reader) error {
	name = l.realPath(name)
	_ = os.MkdirAll(filepath.Dir(name), 0755)
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, l.Perm)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	if isUnixHidden(name) {
		_ = hideFile(name)
	}
	return err
}

func (l *Local) Remove(name string) error {
	name = l.realPath(name)
	return os.Remove(name)
}

func (l *Local) MkdirAll(name string) error {
	name = l.realPath(name)
	err := os.MkdirAll(name, dirPerm(l.Perm))
	if isUnixHidden(name) {
		_ = hideFile(name)
	}
	return err
}

func (l *Local) ReadDir(name string, opts ListOption) ([]fs.FileInfo, error) {
	name = l.realPath(name)

	is, err := ioutil.ReadDir(name)
	if err != nil {
		return nil, err
	}

	var fis []fs.FileInfo
	for _, i := range is {
		if (opts &IncludeHiddenFiles) != 0  || !strings.HasPrefix(i.Name(), ".") {
			fis = append(fis, i)
		}
	}
	return fis, err
}

func (l *Local) Watch(name string) chan string {
	return nil
}

func (l *Local) Stat(name string) (fs.FileInfo, error) {
	name = l.realPath(name)
	return os.Stat(name)
}

func (l *Local) Touch(name string) error {
	name = l.realPath(name)

	currentTime := time.Now().Local()
	return os.Chtimes(name, currentTime, currentTime)
}

func (l *Local) Close() error {
	return nil
}
