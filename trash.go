package notfs

import (
	"io"
	"io/fs"
	"path"
	"strings"
)

type Trash struct {
	F      FS
	Folder string
}

func (t *Trash) Rename(old, new string) error {
	return t.F.Rename(old, new)
}

func NewTrash(f FS, trashFolder string) FS {
	return &Trash{f, trashFolder}
}

func (t *Trash) Props() Props {
	return t.F.Props()
}

func (t *Trash) CopyTo(name string, w io.Writer) error {
	return t.F.CopyTo(name, w)
}

func (t *Trash) CopyFrom(name string, r io.Reader) error {
	return t.F.CopyFrom(name, r)
}

func (t *Trash) Remove(name string) error {
	if strings.HasPrefix(name, t.Folder+"/") {
		return t.F.Remove(name)
	} else {
		dest := path.Join(t.Folder, name)
		t.F.Remove(dest)
		return t.F.Rename(name, dest)
	}
}

func (t *Trash) MkdirAll(name string) error {
	return t.F.MkdirAll(name)
}

func (t *Trash) ReadDir(name string, opts ListOption) ([]fs.FileInfo, error) {
	return t.F.ReadDir(name, opts)
}

func (t *Trash) Watch(name string) chan string {
	return t.F.Watch(name)
}

func (t *Trash) Stat(name string) (fs.FileInfo, error) {
	return t.F.Stat(name)
}

func (t *Trash) Touch(name string) error {
	return t.F.Touch(name)
}

func (t *Trash) Close() error {
	return t.F.Close()
}
