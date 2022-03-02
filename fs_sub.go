package main

import (
	"io"
	"io/fs"
	"path"
)

type Sub struct {
	F   FS
	Dir string
}

func NewSub(f FS, dir string) FS {
	if e, ok := f.(*Encrypted);ok {
		return NewEncrypted(NewSub(e.F, dir), e.B)
	}
	if s, ok := f.(*Sub); ok {
		return NewSub(s.F, path.Join(s.Dir, dir))
	}
	if l, ok := f.(*Local);ok {
		return NewLocal(path.Join(l.Mount, dir), l.Perm)
	}
	return &Sub{
		F:       f,
		Dir:     dir,
	}
}

func (s *Sub) Props() Props {
	return s.F.Props()
}

func (s *Sub) MkdirAll(name string) error {
	name = path.Join(s.Dir, name)
	return s.F.MkdirAll(name)
}

func (s *Sub) ReadDir(name string, opts ListOption) ([]fs.FileInfo, error) {
	name = path.Join(s.Dir, name)
	return s.F.ReadDir(name, opts)
}


func (s *Sub) Watch(name string) chan string {
	name = path.Join(s.Dir, name)
	return s.F.Watch(name)
}

func (s *Sub) Stat(name string) (fs.FileInfo, error) {
	name = path.Join(s.Dir, name)
	return s.F.Stat(name)
}

func (s *Sub) Remove(name string) error {
	name = path.Join(s.Dir, name)
	return s.F.Remove(name)
}

func (s *Sub) Touch(name string) error {
	name = path.Join(s.Dir, name)
	return s.F.Touch(name)
}

func (s *Sub) Rename(old, new string) error {
	old = path.Join(s.Dir, old)
	new = path.Join(s.Dir, new)
	return s.F.Rename(old, new)
}

func (s *Sub) Pull(name string, w io.Writer) error {
	name = path.Join(s.Dir, name)
	return s.F.Pull(name, w)
}

func (s *Sub) Push(name string, r io.Reader) error {
	name = path.Join(s.Dir, name)
	return s.F.Push(name, r)
}

func (s *Sub) Close() error {
	return nil
}
