package store

import (
	"github.com/patrickmn/go-cache"
	"io"
	"io/fs"
	"math"
	"os"
	"path"
	"time"
)

type Access struct {
	F          FS
	Groups     []Group
	GroupCache *cache.Cache
}

func NewAccess(f FS, groups []Group, cacheExpiration time.Duration) FS {
	return &Access{
		F:          f,
		Groups:     groups,
		GroupCache: cache.New(cacheExpiration, 2*cacheExpiration),
	}
}

func (a Access) Props() Props {
	return Props{
		Quota:       math.MaxInt64,
		Free:        math.MaxInt64,
		MinFileSize: 0,
		MaxFileSize: math.MaxInt64,
	}
}

func (a Access) GetGroup(name string) Group {
	group, ok := a.GroupCache.Get(name)
	if !ok {
		var m Attr
		_ = GetMeta(a.F, name, &m)
		group = m.Group
		a.GroupCache.Set(name, m.Group, cache.DefaultExpiration)
	}
	return group.(Group)
}

func (a Access) isAccessible(name string) bool {
	group := a.GetGroup(name)
	for _, g := range a.Groups {
		if g == group {
			return true
		}
	}
	return false
}

func (a Access) ReadDir(name string, opts ListOption) ([]fs.FileInfo, error) {
	ls, err := a.F.ReadDir(name, opts)
	if err != nil {
		return nil, err
	}
	var is []fs.FileInfo
	for _, l := range ls {
		if a.isAccessible(path.Join(name, l.Name())) {
			is = append(is, l)
		}
	}
	return is, nil
}

func (a Access) Watch(name string) chan string {
	if a.isAccessible(name) {
		return a.F.Watch(name)
	} else {
		return nil
	}
}

func (a Access) Stat(name string) (fs.FileInfo, error) {
	if a.isAccessible(name) {
		return a.F.Stat(name)
	} else {
		return nil, os.ErrPermission
	}
}

func (a Access) Remove(name string) error {
	if a.isAccessible(name) {
		return a.Remove(name)
	} else {
		return os.ErrPermission
	}
}

func (a Access) Touch(name string) error {
	if a.isAccessible(name) {
		return a.F.Touch(name)
	} else {
		return os.ErrPermission
	}
}

func (a Access) Rename(old, new string) error {
	if a.isAccessible(old) {
		return a.F.Rename(old, new)
	} else {
		return os.ErrPermission
	}
}

func (a Access) MkdirAll(name string) error {
	return a.F.MkdirAll(name)
}

func (a Access) Pull(name string, w io.Writer) error {
	if !a.isAccessible(name) {
		return os.ErrPermission
	}
	return a.F.Pull(name, w)
}

func (a Access) Push(name string, r io.Reader) error {
	_, err := a.F.Stat(name)
	if err == nil && !a.isAccessible(name) {
		return os.ErrPermission
	}

	return a.F.Push(name, r)
}

func (a Access) Close() error {
	return a.F.Close()
}
