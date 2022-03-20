package fs

import (
	"context"
	"io"
	"time"
)



func Copy(from, to FS, src, dest string, includeMeta bool, timeout time.Duration) error {
	if includeMeta {
		err := copyFile(from, to, metaName(src), metaName(dest), timeout)
		if err == context.Canceled {
			return err
		}
	}
	err := copyFile(from, to, src, dest, timeout)
	if err != nil && includeMeta {
		_ = to.Remove(metaName(dest))
	}
	return err
}

func copyFile(from, to FS, src, dest string, timeout time.Duration) error {
	var ctx = context.Background()
	var cancel context.CancelFunc

	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	e := make(chan error)
	defer close(e)
	pr, pw := io.Pipe()

	go func() {
		_ = from.Pull(src, pw)
		pw.Close()
	}()
	go func() {
		defer pr.Close()
		e <- to.Push(dest, pr)
	}()

	for {
		select {
		case err := <-e:
			return err
		case <-ctx.Done():
			return context.Canceled
		}
	}
}

type PeekWriter struct {
	size int
	data []byte
}

func (w *PeekWriter) Write(bs []byte) (int, error) {
	l := w.size - len(w.data)
	if l > len(bs) {
		l = len(bs)
	}
	w.data = append(w.data, bs[0:l]...)
	if len(w.data) == w.size {
		return l, io.EOF
	} else {
		return l, nil
	}
}

func Peek(f FS, name string, size int) ([]byte, error) {
	w := &PeekWriter{
		size: size,
		data: nil,
	}
	err := f.Pull(name, w)
	return w.data, err
}
