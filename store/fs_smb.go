package store

import (
	"encoding/base64"
	"fmt"
	"github.com/hirochachacha/go-smb2"
	"io"
	"io/fs"
	"math"
	"net"
	"path"
	"strings"
	"time"
)

type SMBConfig struct {
	Addr     string `json:"addr" yaml:"addr"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	Hash     string `json:"hash" yaml:"hash"`
	Share    string `json:"share" yaml:"share"`
}

type SMB struct {
	s   *smb2.Session
	sh  *smb2.Share
	url string
}

func NewSMB(config SMBConfig) (FS, error) {
	s, err := getSession(config)
	if err != nil {
		return nil, err
	}

	share, err := s.Mount(config.Share)
	if err != nil {
		s.Logoff()
		return nil, err
	}

	url := fmt.Sprintf("smb://%s@%s/%s", config.Username, config.Addr, config.Share)
	return &SMB{s, share, url}, nil
}

func ListSMBShares(config SMBConfig) ([]string, error) {
	s, err := getSession(config)
	if err != nil {
		return nil, err
	}
	defer s.Logoff()

	return s.ListSharenames()
}

func getSession(config SMBConfig) (*smb2.Session, error) {
	addr := config.Addr
	if !strings.ContainsRune(addr, ':') {
		addr = fmt.Sprintf("%s:445", addr)
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var hash []byte
	if config.Hash != "" {
		hash, err = base64.StdEncoding.DecodeString(config.Hash)
		if err != nil {
			return nil, err
		}
	}

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     config.Username,
			Password: config.Password,
			Hash:     hash,
		},
	}

	return d.Dial(conn)
}

func (s *SMB) Props() Props {
	return Props{
		Quota:       math.MaxInt64,
		Free:        math.MaxInt64,
		MinFileSize: 0,
		MaxFileSize: math.MaxInt64,
	}
}

func (s *SMB) MkdirAll(name string) error {
	return s.sh.MkdirAll(name, 0755)
}

func (s *SMB) mkParent(name string) error {
	return s.sh.MkdirAll(path.Dir(name), 0755)
}

func (s SMB) Pull(name string, w io.Writer) error {
	r, err := s.sh.Open(name)
	if err != nil {
		return err
	}
	defer r.Close()

	_, err = io.Copy(w, r)
	return err
}

func (s SMB) Push(name string, r io.Reader) error {
	err := s.mkParent(name)

	w, err := s.sh.Create(name)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = io.Copy(w, r)

	return err
}

func (s *SMB) ReadDir(name string, opts ListOption) ([]fs.FileInfo, error) {
	entries, err := s.sh.ReadDir(name)
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

func (s *SMB) Watch(_ string) chan string {
	return nil
}

func (s *SMB) Stat(name string) (fs.FileInfo, error) {
	return s.sh.Stat(name)
}

func (s *SMB) Remove(name string) error {
	return s.sh.Remove(name)
}

func (s *SMB) Touch(name string) error {
	return s.sh.Chtimes(name, time.Now(), time.Now())
}

func (s *SMB) Rename(old, new string) error {
	_ = s.mkParent(new)
	return s.sh.Rename(old, new)
}

func (s *SMB) Close() error {
	err := s.sh.Umount()
	if err != nil {
		return err
	}
	return s.s.Logoff()
}

func (s *SMB) String() string {
	return s.url
}
