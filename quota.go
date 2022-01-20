package notfs

import (
	"io"
	"io/fs"
)

type QuotaFS struct {
	F     FS
	Limit int64
	Current int64
}

func NewQuota(f FS, limit int64) FS {
	var current int64 = 0
	_ = Walk(f, "", IncludeHiddenFiles, func(dir string, file fs.FileInfo) {
		current += file.Size()
	})

	return QuotaFS{
		F:       f,
		Limit:   limit,
		Current: current,
	}
}

func (q QuotaFS) Props() Props {
	return Props{
		Quota:       q.Limit,
		Free:        q.Limit - q.Current,
		MinFileSize: 0,
		MaxFileSize: q.Limit,
	}
}

func (q QuotaFS) MkdirAll(name string) error {
	return q.F.MkdirAll(name)
}

func (q QuotaFS) ReadDir(name string, opts ListOption) ([]fs.FileInfo, error) {
	return q.F.ReadDir(name, opts)
}

func (q QuotaFS) Watch(name string) chan string {
	return q.F.Watch(name)
}

func (q QuotaFS) Stat(name string) (fs.FileInfo, error) {
	return q.F.Stat(name)
}

func (q QuotaFS) Remove(name string) error {
	return q.F.Remove(name)
}

func (q QuotaFS) Touch(name string) error {
	return q.F.Touch(name)
}

func (q QuotaFS) Rename(old, new string) error {
	return q.F.Rename(old, new)
}

func (q QuotaFS) CopyTo(name string, w io.Writer) error {
	return q.F.CopyTo(name, w)
}

type CountingReader struct {
	R   io.Reader
	Cnt int64
}

func (r *CountingReader) Read(bs []byte) (int, error) {
	n, err := r.R.Read(bs)
	r.Cnt += int64(n)
	return n, err
}

func (q QuotaFS) CopyFrom(name string, r io.Reader) error {
	if q.Current > q.Limit {
		return ErrOffQuota
	}
	l, _ := q.F.Stat(name)
	cr := CountingReader{r, 0}
	err := q.F.CopyFrom(name, &cr)
	q.Current += cr.Cnt - l.Size()
	return err
}

func (q QuotaFS) Close() error {
	return q.F.Close()
}
