package notfs

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)


func TestCrc(t *testing.T) {
	f := NewMemory(nil, 0)
	f.CopyFrom("test.txt", bytes.NewBufferString("This is a sample test"))
	crc := calculateCRC64(f, "test.txt")
	crc2 := calculateCRC64(f, "test.txt")

	assert.Equal(t, crc, crc2)
}
