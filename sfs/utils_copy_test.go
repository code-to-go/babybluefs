package sfs

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.DebugLevel)
}

func TestFSClone(t *testing.T) {
	var c FTPConfig
	err := readTestConfig("ftp.yaml", &c)

	l1 := NewLocal(os.TempDir(), 0644)
	l2, err := NewFTP(c)
	assert.NoError(t, err)

	err = l1.Push("stg/test/cloneTest.txt", bytes.NewReader([]byte("Hello")))
	assert.NoError(t, err)

	err = Copy(l1, l2, "stg/test/cloneTest.txt", "stg/test/clone2.txt", false, 0)
	assert.NoError(t, err)

}
