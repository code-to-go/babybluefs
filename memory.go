package notfs

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

type fileInMemory struct {
	modTime      time.Time
	creationTime time.Time
	data         []byte
}

type Memory struct {
	files      map[string]fileInMemory
	source     FS
	filesLock  sync.Mutex
	expiration time.Duration
	setupTime  time.Time
}

func cloneFS(source, dest FS, ph string) {
	stat, err := source.Stat(ph)
	if err == nil {
		if stat.IsDir() {
			dest.MkdirAll(ph)
			ls, _ := source.ReadDir(ph, IncludeHiddenFiles)
			for _, l := range ls {
				cloneFS(source, dest, path.Join(ph, l.Name()))
			}
		} else {
			_ = Copy(source, dest, ph, ph, true, 0)
		}
	}
}

func NewMemory(clone FS, expiration time.Duration) FS {
	m := &Memory{
		make(map[string]fileInMemory),
		clone,
		sync.Mutex{},
		expiration,
		time.Time{},
	}

	if clone != nil {
		cloneFS(clone, m, "")
		m.setupTime = time.Now()
	}
	m.MkdirAll("")

	go func() {
		for m.expiration > 0 {
			m.filesLock.Lock()
			now := time.Now()

			for n, f := range m.files {
				if now.Sub(f.modTime) > expiration {
					if f.creationTime.After(m.setupTime) {
						delete(m.files, n)
					} else if f.modTime.After(m.setupTime) && f.data != nil {
						data, err := ReadFile(m.source, n)
						if err == nil {
							m.files[n] = fileInMemory{
								modTime:      f.creationTime,
								creationTime: f.creationTime,
								data:         data,
							}
						}
					}
				}
			}
			m.filesLock.Unlock()
			time.Sleep(time.Minute)
		}
	}()

	return m
}

func (m *Memory) Props() Props {
	return Props{
		Quota:       -1,
		Free:        -1,
		MaxFileSize: -1,
	}
}

func (m *Memory) MkdirAll(name string) error {
	m.filesLock.Lock()
	if _, ok := m.files[name]; ok {
		m.filesLock.Unlock()
		return nil
	}

	m.files[name] = fileInMemory{
		modTime: time.Now(),
		data:    nil,
	}
	m.filesLock.Unlock()

	dir := path.Dir(name)
	if dir != "" {
		m.MkdirAll(name)
	}

	return nil
}

func (m *Memory) CopyTo(name string, w io.Writer) error {
	m.filesLock.Lock()
	defer m.filesLock.Unlock()

	f, ok := m.files[name]
	if !ok {
		return os.ErrNotExist
	}

	_, err := io.Copy(w, &ByteStream{f.data, 0})
	return err
}

func (m *Memory) CopyFrom(name string, r io.Reader) error {
	var s ByteStream

	_, err := io.Copy(&s, r)
	if err != nil {
		return err
	}

	dir := path.Dir(name)
	if _, ok := m.files[dir]; !ok {
		m.MkdirAll(dir)
	}

	m.filesLock.Lock()
	defer m.filesLock.Unlock()

	now := time.Now()
	if f, ok := m.files[name]; ok {
		m.files[name] = fileInMemory{
			modTime: now,
			creationTime: f.creationTime,
			data: s.Data,
		}
	} else {
		m.files[name] = fileInMemory{
			modTime:      now,
			creationTime: now,
			data:         s.Data,
		}
	}

	f := m.files[name]
	fmt.Println("****", name, f.modTime, f.creationTime)

	return nil
}

func (m *Memory) Remove(name string) error {
	m.filesLock.Lock()
	defer m.filesLock.Unlock()

	_, ok := m.files[name]
	if !ok {
		return os.ErrNotExist
	}
	return nil
}

func (m *Memory) ReadDir(name string, opts ListOption) ([]fs.FileInfo, error) {
	m.filesLock.Lock()
	defer m.filesLock.Unlock()

	if name != "" && !strings.HasSuffix(name, "/") {
		name = name + "/"
	}

	var sfo []fs.FileInfo
	for n, f := range m.files {
		if !strings.HasPrefix(n, name) || n == name {
			continue
		}

		n = n[len(name):]
		if opts&IncludeHiddenFiles == 0 && n[0] == '.' {
			continue
		}
		if strings.ContainsRune(n, '/') {
			continue
		}

		sfo = append(sfo, SimpleFileInfo{
			name:    n,
			size:    int64(len(f.data)),
			isDir:   f.data == nil,
			modTime: f.modTime,
		})
	}
	return sfo, nil
}

func (m *Memory) Watch(name string) chan string {
	return nil
}

func (m *Memory) Stat(name string) (fs.FileInfo, error) {
	m.filesLock.Lock()
	defer m.filesLock.Unlock()

	f, ok := m.files[name]
	if !ok {
		return nil, os.ErrNotExist
	}

	return SimpleFileInfo{
		name:    path.Base(name),
		size:    int64(len(f.data)),
		isDir:   f.data == nil,
		modTime: f.modTime,
	}, nil
}

func (m *Memory) Touch(name string) error {
	m.filesLock.Lock()
	defer m.filesLock.Unlock()

	f, ok := m.files[name]
	if !ok {
		return os.ErrNotExist
	}

	f.modTime = time.Now()
	return nil
}

func (m *Memory) Rename(old, new string) error {
	m.filesLock.Lock()
	defer m.filesLock.Unlock()

	f, ok := m.files[old]
	if !ok {
		return os.ErrNotExist
	}

	m.files[new] = f
	delete(m.files, old)
	return nil
}

func (m *Memory) Close() error {
	m.filesLock.Lock()
	defer m.filesLock.Unlock()

	m.files = make(map[string]fileInMemory)
	m.expiration = 0
	return nil
}
