package sfs

import (
	"bytes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/gob"
	"math/rand"
	"os"
	"time"
)

type MeshConfig3 struct {
	FTP    map[string]FTPConfig
	SFTP   map[string]SFTPConfig
	S3     map[string]S3Config
	Azure  map[string]AzureConfig
	HTTP   map[string]HTTPConfig
	Groups map[Group]string
}

type FSConfig struct {
	Name  string       `json:"name" yaml:"name"`
	Group Group        `json:"group" yaml:"group"`
	FTP   *FTPConfig   `json:"ftp,omitempty" yaml:"ftp,omitempty"`
	SFTP  *SFTPConfig  `json:"sftp,omitempty" yaml:"sftp,omitempty"`
	S3    *S3Config    `json:"s3,omitempty" yaml:"s3,omitempty"`
	Azure *AzureConfig `json:"azure,omitempty" yaml:"azure,omitempty"`
	HTTP  *HTTPConfig  `json:"http,omitempty" yaml:"http,omitempty"`
}

type MeshConfig struct {
	Remotes []FSConfig       `json:"remotes" yaml:"remotes"`
	Groups  map[Group]string `json:"groups" yaml:"groups"`
}

func UpdateFromFile(m *Mesh, f FS, configPath string, reconnect bool) error {
	c, err := ReadConfig(f, configPath)
	if err != nil {
		return err
	}
	return UpdateFromConfig(m, c, reconnect)
}

func shouldConnect(m *Mesh, n string, remotes map[string]remote, reconnect bool) bool {
	if remotes == nil {
		return true
	}
	if r, ok := remotes[n]; ok {
		if reconnect {
			delete(remotes, n)
			_ = r.F.Close()
		} else {
			m.Remotes[n] = remotes[n]
			delete(remotes, n)
			return false
		}
	}
	return true
}

func isValidKeyHash(f FS, g Group, groups map[Group]string) bool {
	k, ok := groups[g]
	if !ok {
		return false
	}

	h := sha256.Sum256([]byte(k))
	buf := bytes.NewBuffer(nil)

	_, err := f.Stat(keyHashFile)
	if os.IsNotExist(err) {
		return f.Pull(keyHashFile, bytes.NewBuffer(h[:])) == nil
	}
	if err != nil {
		return false
	}

	err = f.Pull(keyHashFile, buf)
	return err == nil && bytes.Equal(buf.Bytes(), h[:])
}

//func connect(name string, m *Mesh, remotes map[string]remote, g Group, groups map[Group]string,
//	get func() (FS, error), reconnect bool) error {
//	if !shouldConnect(m, name, remotes, reconnect) {
//		return nil
//	}
//
//	e := &multierror.Error{}
//	f, err := get()
//
//	if err == nil {
//		if !isValidKeyHash(f, g, groups) {
//			e = multierror.Append(e, os.ErrPermission)
//			m.RemotesState[name] = "Invalid Encryption Key"
//			_ = f.Close()
//		} else {
//			m.Remotes[name] = remote{
//				F:     f,
//				Group: g,
//			}
//			m.RemotesState[name] = ""
//		}
//	} else {
//		e = multierror.Append(e, err)
//		m.RemotesState[name] = err.Error()
//	}
//
//	return e
//}

func NewFS(c FSConfig) (FS, error) {
	switch {
	case c.FTP != nil:
		return NewFTP(*c.FTP)
	case c.SFTP != nil:
		return NewSFTP(*c.SFTP)
	case c.S3 != nil:
		return NewS3(*c.S3)
	case c.Azure != nil:
		return NewAzure(*c.Azure)
	case c.HTTP != nil:
		return NewHTTP(*c.HTTP)
	}

	return nil, os.ErrInvalid
}

//func UpdateFromConfig3(m *Mesh, c MeshConfig3, reconnect bool) error {
//	var e error
//	remotes := m.Remotes
//
//	m.sync.Lock()
//	defer m.sync.Unlock()
//
//
//	m.Keys = map[Group]cipher.Block{}
//	m.Remotes = map[string]remote{}
//	m.RemotesState = map[string]string{}
//	m.LastSync = map[string]time.Time{}
//	groups := c.Groups
//
//	for group, key := range groups {
//		b, _ := NewAesCipher([]byte(key))
//		m.Keys[group] = b
//	}
//
//	for n, c := range c.FTP {
//		n = fmt.Sprintf("FTP.%s", n)
//		e = multierror.Append(e, connect(n, m, remotes, c.Group, groups, func() (FS, error) {
//			return NewFTP(c)
//		}, reconnect))
//	}
//
//	for n, c := range c.SFTP {
//		n = fmt.Sprintf("SFTP.%s", n)
//		e = multierror.Append(e, connect(n, m, remotes, c.Group, groups, func() (FS, error) {
//			return NewSFTP(c)
//		}, reconnect))
//	}
//
//	for n, c := range c.S3 {
//		n = fmt.Sprintf("S3.%s", n)
//		e = multierror.Append(e, connect(n, m, remotes, c.Group, groups, func() (FS, error) {
//			return NewS3(c)
//		}, reconnect))
//	}
//	for n, c := range c.Azure {
//		n = fmt.Sprintf("Azure.%s", n)
//		e = multierror.Append(e, connect(n, m, remotes, c.Group, groups, func() (FS, error) {
//			return NewAzure(c)
//		}, reconnect))
//	}
//	for n, c := range c.HTTP {
//		n = fmt.Sprintf("HTTP.%s", n)
//		e = multierror.Append(e, connect(n, m, remotes, c.Group, groups, func() (FS, error) {
//			return NewHTTP(c)
//		}, reconnect))
//	}
//
//	for _, r := range remotes {
//		_ = r.F.Close()
//	}
//
//	return nil
//}

func UpdateFromConfig(m *Mesh, c MeshConfig, reconnect bool) error {
	m.sync.Lock()
	defer m.sync.Unlock()

	m.Keys = map[Group]cipher.Block{}
	m.Remotes = map[string]remote{}
	m.RemotesState = map[string]string{}
	m.LastSync = map[string]time.Time{}
	groups := c.Groups

	for group, key := range groups {
		b, _ := NewAesCipher([]byte(key))
		m.Keys[group] = b
	}

	for _, c := range c.Remotes {
		name := c.Name

		r, ok := m.Remotes[name]
		if ok {
			if reconnect {
				_ = r.F.Close()
			} else {
				continue
			}
		}

		f, err := NewFS(c)
		if err != nil {
			m.RemotesState[name] = err.Error()
			continue
		}

		if !isValidKeyHash(f, c.Group, groups) {
			m.RemotesState[name] = "Invalid Encryption Key"
			continue
		}

		m.Remotes[name] = remote{
			F:     f,
			Group: c.Group,
		}
	}

	return nil
}

func ConfigToToken(c MeshConfig, key []byte) (string, error) {
	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(c)
	if err != nil {
		return "", err
	}

	bs := buf.Bytes()
	if key != nil {
		b, err := NewAesCipher(key)
		if err != nil {
			return "", err
		}
		bs, err = EncryptBytes(b, bs)
		if err != nil {
			return "", err
		}

	}
	return base64.StdEncoding.EncodeToString(bs), nil
}

func TokenToConfig(token string, key []byte) (MeshConfig, error) {
	var c MeshConfig

	bs, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return c, err
	}

	if key != nil {
		b, err := NewAesCipher(key)
		if err != nil {
			return MeshConfig{}, err
		}
		bs, err = DecryptBytes(b, bs)
		if err != nil {
			return MeshConfig{}, err
		}
	}

	err = gob.NewDecoder(bytes.NewBuffer(bs)).Decode(&c)
	return c, err
}

func ReadConfig(f FS, configPath string) (MeshConfig, error) {
	var c MeshConfig
	err := ReadYaml(f, configPath, &c)
	return c, err
}

func WriteConfig(f FS, configPath string, c MeshConfig) error {
	return WriteYaml(f, configPath, c)
}

func generateRandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	rand.Seed(time.Now().UnixMicro())
	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func NewConfig() MeshConfig {
	return MeshConfig{
		Groups: map[Group]string{
			"public": generateRandomString(32),
		},
	}
}
