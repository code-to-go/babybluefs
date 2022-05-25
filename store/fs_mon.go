package store

import (
	"fmt"
	"github.com/fatih/color"
	"io"
	"io/fs"
	"math"
	"strings"
)

type Op string

const (
	OpRead   Op = "R"
	OpWrite  Op = "W"
	OpRemove Op = "D"
)

type Progress struct {
	Name string
	Size int64
	Op   Op
}

type Mon struct {
	F  FS
	Ch chan Progress
}

type monPipe struct {
	R    io.Reader
	W    io.Writer
	Name string
	Size int64
	Ch   chan Progress
}

func (m monPipe) Write(p []byte) (n int, err error) {
	n, err = m.W.Write(p)
	if err == nil {
		m.Size += int64(n)
		m.Ch <- Progress{m.Name, m.Size, OpWrite}
	}
	return
}

func (m monPipe) Read(p []byte) (n int, err error) {
	n, err = m.R.Read(p)
	if err == nil {
		m.Size += int64(n)
		m.Ch <- Progress{m.Name, m.Size, OpRead}
	}
	return
}

func NewMon(f FS, ch chan Progress) FS {
	return &Mon{
		F:  f,
		Ch: ch,
	}
}

func NewConsoleMon(f FS) FS {
	ch := make(chan Progress)
	go func() {
		for p := range ch {
			progress := strings.Repeat(string(p.Op), int(math.Trunc(math.Log10(float64(p.Size)))))
			color.Green("%s [%s] %d\r", p.Name, progress, p.Size)
		}
	}()
	return NewMon(f, ch)
}

func (m Mon) Props() Props {
	return m.F.Props()
}

func (m Mon) ReadDir(path string, opts ListOption) ([]fs.FileInfo, error) {
	return m.F.ReadDir(path, opts)
}

func (m Mon) Stat(name string) (fs.FileInfo, error) {
	return m.F.Stat(name)
}

func (m Mon) Remove(name string) error {
	stat, _ := m.F.Stat(name)
	err := m.F.Remove(name)
	if err == nil {
		m.Ch <- Progress{name, stat.Size(), OpRemove}
	}
	return err
}

func (m Mon) Touch(name string) error {
	return m.F.Touch(name)
}

func (m Mon) Watch(name string) chan string {
	return m.F.Watch(name)
}

func (m Mon) Rename(old, new string) error {
	return m.F.Rename(old, new)
}

func (m Mon) MkdirAll(name string) error {
	return m.F.MkdirAll(name)
}

func (m Mon) Pull(name string, w io.Writer) error {
	return m.F.Pull(name, monPipe{
		W:    w,
		Name: name,
		Ch:   m.Ch,
	})
}

func (m Mon) Push(name string, r io.Reader) error {
	return m.F.Push(name, monPipe{
		R:    r,
		Name: name,
		Ch:   m.Ch,
	})
}

func (m Mon) Close() error {
	close(m.Ch)
	return m.F.Close()
}

func (m Mon) String() string {
	return fmt.Sprintf("%s#mon", m.F)
}
