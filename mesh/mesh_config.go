package mesh

import (
	"bytes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/gob"
	"stratofs/fs"
	"time"
)

// MeshConfig defines a mesh built of multiple file storages and groups
type MeshConfig struct {
	Remotes []fs.Config         `json:"remotes" yaml:"remotes"`
	Groups  map[fs.Group]string `json:"groups" yaml:"groups"`
}

// MeshFromFile reads a Mesh configuration from a local file and update the provided mesh m.
// It creates a new mesh when m is nil
func MeshFromFile(f fs.FS, configPath string, m *Mesh, reconnect bool) error {
	c, err := ReadMeshConfig(f, configPath)
	if err != nil {
		return err
	}
	return MeshFromConfig(c, m, reconnect)
}

// MeshFromConfig updates a mesh m with the provided configuration c
// It creates a new mesh when m is nil
func MeshFromConfig(c MeshConfig, m *Mesh, reconnect bool) error {
	m.sync.Lock()
	defer m.sync.Unlock()

	m.Keys = map[fs.Group]cipher.Block{}
	m.Remotes = map[string]remote{}
	m.RemotesState = map[string]string{}
	m.LastSync = map[string]time.Time{}
	groups := c.Groups

	for group, key := range groups {
		b, _ := fs.NewAesCipher([]byte(key))
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

		f, err := fs.NewFS(c)
		if err != nil {
			m.RemotesState[name] = err.Error()
			continue
		}

		if !fs.isValidKeyHash(f, c.Group, groups) {
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

// ReadMeshConfig reads a mesh configuration
func ReadMeshConfig(f fs.FS, configPath string) (MeshConfig, error) {
	var c MeshConfig
	err := fs.ReadYaml(f, configPath, &c)
	return c, err
}


// WriteMeshConfig writes a mesh configuration
func WriteMeshConfig(f fs.FS, configPath string, c MeshConfig) error {
	return fs.WriteYaml(f, configPath, c)
}

// NewMeshConfig creates an empty mesh configuration
func NewMeshConfig() MeshConfig {
	return MeshConfig{
		Groups: map[fs.Group]string{
			"public": fs.generateRandomString(32),
		},
	}
}

// MeshConfigToToken converts a mesh configuration to a token, which can be used for distribution.
// The token is encoded in AES with the provided key
func MeshConfigToToken(c MeshConfig, key []byte) (string, error) {
	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(c)
	if err != nil {
		return "", err
	}

	bs := buf.Bytes()
	if key != nil {
		b, err := fs.NewAesCipher(key)
		if err != nil {
			return "", err
		}
		bs, err = fs.EncryptBytes(b, bs)
		if err != nil {
			return "", err
		}

	}
	return base64.StdEncoding.EncodeToString(bs), nil
}

// TokenToMeshConfig converts a token to a mesh configuration.
// The token is decoded in AES with the provided key
func TokenToMeshConfig(token string, key []byte) (MeshConfig, error) {
	var c MeshConfig

	bs, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return c, err
	}

	if key != nil {
		b, err := fs.NewAesCipher(key)
		if err != nil {
			return MeshConfig{}, err
		}
		bs, err = fs.DecryptBytes(b, bs)
		if err != nil {
			return MeshConfig{}, err
		}
	}

	err = gob.NewDecoder(bytes.NewBuffer(bs)).Decode(&c)
	return c, err
}


