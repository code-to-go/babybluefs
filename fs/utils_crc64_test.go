package fs

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)


func TestCrc(t *testing.T) {
	f := NewMemory(nil, 0)
	f.Pull("test.txt", bytes.NewBufferString("This is a sample test"))
	crc := CalculateCRC64(f, "test.txt")
	crc2 := CalculateCRC64(f, "test.txt")

	assert.Equal(t, crc, crc2)
}
