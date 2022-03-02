package main

import (
	"io"
	"io/fs"
	"path"
)

type ByteStream struct {
	Data []byte
	P    int
}

func (s *ByteStream) Write(bs []byte) (int, error) {
	s.Data = append(s.Data, bs...)
	return len(bs), nil
}

func (s *ByteStream) Read(bs []byte) (int, error) {
	c := len(s.Data) - s.P
	if c > len(bs) {
		c = len(bs)
	}

	copy(bs, s.Data[s.P:s.P+c])
	s.P += c
	if s.P == len(s.Data) {
		return c, io.EOF
	} else {
		return c, nil
	}
}

func CopyTo(f FS, name string, w io.Writer, metas ...interface{}) error {
	if len(metas) > 0 {
		if err := GetMeta(f, name, metas...); err != nil {
			return err
		}
	}
	return f.Pull(name, w)
}

func CopyFrom(f FS, name string, r io.Reader, metas ...interface{}) error {
	if len(metas) > 0 {
		if err := SetMeta(f, name, metas...); err != nil {
			return err
		}
	}
	err := f.Push(name, r)
	if err != nil {
		_ = RemoveMeta(f, name)
	}
	return err
}

func ReadFile(f FS, name string, metas ...interface{}) ([]byte, error) {
	var s ByteStream
	err := CopyTo(f, name, &s, metas...)
	if err == nil {
		return s.Data, nil
	} else {
		return nil, err
	}
}

func WriteFile(f FS, name string, data []byte, metas ...interface{}) error {
	s := ByteStream{Data: data}
	return CopyFrom(f, name, &s, metas...)
}

func Walk(f FS, name string, opts ListOption, walk func(dir string, file fs.FileInfo)) error {
	ls, err := f.ReadDir(name, opts)
	for _, l := range ls {
		if l.IsDir() {
			err = Walk(f, path.Join(name, l.Name()), opts, walk)
			if err != nil {
				return err
			}
		} else {
			walk(name, l)
		}
	}
	return nil
}

