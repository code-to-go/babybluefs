package store

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func readTestConfig(name string, out interface{}) error {
	name = filepath.Join("..", "..", "credentials", name)
	name, _ = filepath.Abs(name)
	data, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, out)
}

func TestLocal(t *testing.T) {
	l := NewLocalMount(os.TempDir())
	testLayer(t, l)
}

func TestMemory(t *testing.T) {
	l := NewMemory(nil, 0)
	testLayer(t, l)
}
func TestFTPLayer(t *testing.T) {
	var c FTPConfig
	err := readTestConfig("ftp.yaml", &c)
	l, err := NewFTP(c)
	assert.NoError(t, err)

	testLayer(t, l)

}

func TestEncrypted(t *testing.T) {
	b, _ := NewAesCipher([]byte("Hello"))
	f := NewEncrypted(NewMemory(nil, 0), b)

	testLayer(t, f)
}

func TestS3(t *testing.T) {
	var c S3Config
	err := readTestConfig("s3.yaml", &c)
	l, err := NewS3(c)
	assert.NoError(t, err)

	testLayer(t, l)

}

func TestHTTP(t *testing.T) {
	var c HTTPConfig
	err := readTestConfig("http.yaml", &c)
	l, err := NewHTTP(c)
	assert.NoError(t, err)

	testLayer(t, l)
}

func TestAzure(t *testing.T) {
	var c AzureConfig
	err := readTestConfig("azure.yaml", &c)
	l, err := NewAzure(c)
	assert.NoError(t, err)

	testLayer(t, l)
}

func TestSharepoint(t *testing.T) {
	var c Config
	err := readTestConfig("sharepoint.yaml", &c)
	l, err := NewFS(c)
	assert.NoError(t, err)

	testLayer(t, l)
}

type testMeta struct {
	Owner string
}

func testLayer(t *testing.T, l FS) {

	name := "stg/test/layerTest.enc"
	content := []byte("Some content")

	now := time.Now()
	r := bytes.NewReader(content)
	err := l.Push(name, r)
	assert.NoError(t, err)

	w := &ByteStream{}
	err = l.Pull(name, w)
	assert.NoError(t, err)
	assert.EqualValues(t, content, w.Data)

	st, err := l.Stat(name)
	assert.NoError(t, err)
	assert.Equal(t, int64(len(content)), st.Size())
	assert.Equal(t, false, st.IsDir())
	d := st.ModTime().Sub(now)
	assert.True(t, d < time.Minute)
	assert.True(t, d > -time.Minute)

	m := testMeta{"me"}
	err = SetMeta(l, name, m)
	assert.NoError(t, err)

	m.Owner = ""
	err = GetMeta(l, name, &m)
	assert.NoError(t, err)
	assert.Equal(t, "me", m.Owner)

	err = l.Remove(name)
	assert.NoError(t, err)
}
