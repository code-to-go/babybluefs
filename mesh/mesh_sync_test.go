package mesh

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"stratofs/fs"
	"testing"
	"time"
)


var numberFile = 100
var maxSize = 1024 * 1024

func GenerateRandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	rand.Seed(time.Now().UnixMicro())
	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func TestSyncMatrix(t *testing.T) {
	f := fs.NewMemory(nil, 0)
	m := &Mesh{Local: f}
	err := MeshFromFile(fs.NewLocal("../../..", 0644),"credentials/mesh.yaml", m, true)
	assert.NoError(t, err)

	for i := 0; i < numberFile; i++ {
		name := fmt.Sprintf("file%d.txt", i)
		s := GenerateRandomString(rand.Intn(maxSize))
		err = fs.CopyFrom(f, name, bytes.NewBufferString(s), &fs.Attr{
			ModifiedBy: "mp@gmail.com",
			Group:      "public",
		})
		assert.NoError(t, err)
	}

	mon := make(chan string)
	go func() {for {
		print(<- mon)
	}}()
	err = Sync(m, "", time.Time{}, mon)

	assert.NoError(t, err)

	_ = m.Local.Remove("file0.txt")
	err = Sync(m, "", time.Time{}, mon)

}
