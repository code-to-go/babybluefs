package fs

import (
	"crypto/cipher"
	"io"
	"io/fs"
)

type Encrypted struct {
	F FS
	B cipher.Block
}

func NewEncrypted(f FS, b cipher.Block) FS {
	return &Encrypted{
		F: f,
		B: b,
	}
}


func (e Encrypted) Props() Props {
	return e.F.Props()
}

func (e Encrypted) ReadDir(path string, opts ListOption) ([]fs.FileInfo, error) {
	return e.F.ReadDir(path, opts)
}

func (e Encrypted) Stat(name string) (fs.FileInfo, error) {
	return e.F.Stat(name)
}

func (e Encrypted) Remove(name string) error {
	return e.F.Remove(name)
}

func (e Encrypted) Touch(name string) error {
	return e.F.Touch(name)
}

func (e Encrypted) Watch(name string) chan string {
	return e.F.Watch(name)
}



func (e Encrypted) Rename(old, new string) error {
	return e.F.Rename(old, new)
}

func (e Encrypted) MkdirAll(name string) error {
	return e.F.MkdirAll(name)
}

func (e Encrypted) Pull(name string, w io.Writer) error {
	return e.F.Pull(name, CipherWriter(e.B,w))
}

func (e Encrypted) Push(name string, r io.Reader) error {
	return e.F.Push(name, CipherReader(e.B, r))
}

func (e Encrypted) Close() error {
	return e.F.Close()
}
