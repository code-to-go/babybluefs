package main

import (
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"io/fs"
	"math"
	"path"
	"strings"
	"time"
)

type SFTPConfig struct {
	Addr     string `json:"addr" yaml:"addr"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	Base  string `json:"base" yaml:"base"`
	Group Group  `json:"group" yaml:"group"`
}

type SFTP struct {
	c    *sftp.Client
	base string
}

func NewSFTP(config SFTPConfig) (FS, error) {
	addr := config.Addr
	if !strings.ContainsRune(addr, ':') {
		addr = fmt.Sprintf("%s:22", addr)
	}

	cc := &ssh.ClientConfig{
		User: config.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", addr, cc)
	if err != nil {
		return nil, err
	}
	c, err := sftp.NewClient(client)
	if err != nil {
		return nil, err
	}

	return &SFTP{c, config.Base}, nil
}

func (s *SFTP) Props() Props {
	return Props{
		Quota:       math.MaxInt64,
		Free:        math.MaxInt64,
		MinFileSize: 0,
		MaxFileSize: math.MaxInt64,
	}
}

func (s *SFTP) MkdirAll(name string) error {
	return s.c.MkdirAll(path.Join(s.base, name))
}

func (s *SFTP) mkParent(name string) error {
	dir := path.Join(s.base, path.Dir(name))
	_, err := s.c.Stat(dir)
	if err == nil {
		return nil
	}

	return s.c.MkdirAll(dir)
}

func (s SFTP) Pull(name string, w io.Writer) error {
	r, err := s.c.Open(path.Join(s.base, name))
	if err != nil {
		return err
	}
	defer r.Close()

	_, err = io.Copy(w, r)
	return err
}

func (s SFTP) Push(name string, r io.Reader) error {
	err := s.mkParent(name)

	w, err := s.c.Create(path.Join(s.base, name))
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = io.Copy(w, r)

	return err
}

func (s *SFTP) ReadDir(name string, opts ListOption) ([]fs.FileInfo, error) {
	entries, err := s.c.ReadDir(path.Join(s.base, name))
	if err != nil {
		return nil, err
	}

	var fis []fs.FileInfo
	for _, e := range entries {
		n := e.Name()
		if (opts&IncludeHiddenFiles) == 0 && strings.HasPrefix(n, ".") {
			continue
		}
		fis = append(fis, e)
	}
	return fis, err
}

func (s *SFTP) Watch(name string) chan string {
	return nil
}

func (s *SFTP) Stat(name string) (fs.FileInfo, error) {
	return s.c.Stat(path.Join(s.base, name))
}

func (s *SFTP) Remove(name string) error {
	return s.c.Remove(path.Join(s.base, name))
}

func (s *SFTP) Touch(name string) error {
	return s.c.Chtimes(path.Join(s.base, name), time.Now(), time.Now())
}

func (s *SFTP) Rename(old, new string) error {
	_ = s.mkParent(new)
	return s.c.Rename(path.Join(s.base, old), path.Join(s.base, new))
}

func (s *SFTP) Close() error {
	return s.c.Close()
}
