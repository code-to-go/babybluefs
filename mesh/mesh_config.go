package mesh

import (
	"babybluefs/store"
	"bytes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/gob"
	"time"
)

// Config defines a mesh built of multiple file storages and groups
type Config struct {
	Remotes []store.Config         `json:"remotes" yaml:"remotes"`
	Groups  map[store.Group]string `json:"groups" yaml:"groups"`
}

// FromFile reads a Mesh configuration from a local file and update the provided mesh m.
// It creates a new mesh when m is nil
func FromFile(f store.FS, configPath string, m *Mesh, reconnect bool) error {
	c, err := ReadConfig(f, configPath)
	if err != nil {
		return err
	}
	return FromConfig(c, m, reconnect)
}

// FromConfig updates a mesh m with the provided configuration c
// It creates a new mesh when m is nil
func FromConfig(c Config, m *Mesh, reconnect bool) error {
	m.sync.Lock()
	defer m.sync.Unlock()

	m.Keys = map[store.Group]cipher.Block{}
	m.Remotes = map[string]remote{}
	m.RemotesState = map[string]string{}
	m.LastSync = map[string]time.Time{}
	groups := c.Groups

	for group, key := range groups {
		b, _ := store.NewAesCipher([]byte(key))
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

		f, err := store.NewFS(c)
		if err != nil {
			m.RemotesState[name] = err.Error()
			continue
		}

		if !store.IsValidKeyHash(f, c.Group, groups) {
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

// ReadConfig reads a mesh configuration
func ReadConfig(f store.FS, configPath string) (Config, error) {
	var c Config
	err := store.ReadYaml(f, configPath, &c)
	return c, err
}

// WriteConfig writes a mesh configuration
func WriteConfig(f store.FS, configPath string, c Config) error {
	return store.WriteYaml(f, configPath, c)
}

// NewConfig creates an empty mesh configuration
func NewConfig() Config {
	return Config{
		Groups: map[store.Group]string{
			"public": store.GenerateRandomString(32),
		},
	}
}

// ConfigToToken converts a mesh configuration to a token, which can be used for distribution.
// The token is encoded in AES with the provided key
func ConfigToToken(c Config, key []byte) (string, error) {
	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(c)
	if err != nil {
		return "", err
	}

	bs := buf.Bytes()
	if key != nil {
		b, err := store.NewAesCipher(key)
		if err != nil {
			return "", err
		}
		bs, err = store.EncryptBytes(b, bs)
		if err != nil {
			return "", err
		}

	}
	return base64.StdEncoding.EncodeToString(bs), nil
}

// TokenToConfig converts a token to a mesh configuration.
// The token is decoded in AES with the provided key
func TokenToConfig(token string, key []byte) (Config, error) {
	var c Config

	bs, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return c, err
	}

	if key != nil {
		b, err := store.NewAesCipher(key)
		if err != nil {
			return Config{}, err
		}
		bs, err = store.DecryptBytes(b, bs)
		if err != nil {
			return Config{}, err
		}
	}

	err = gob.NewDecoder(bytes.NewBuffer(bs)).Decode(&c)
	return c, err
}
