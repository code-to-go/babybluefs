package main

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"scrum-to-go/core"
	"testing"
	"time"
)


var numberFile = 100
var maxSize = 1024 * 1024

func TestSyncMatrix(t *testing.T) {
	f := NewMemory(nil, 0)
	m := &Mesh{Local: f}
	err := UpdateFromFile(m, NewLocal("../../..", 0644),"credentials/mesh.yaml", true)
	assert.NoError(t, err)

	for i := 0; i < numberFile; i++ {
		name := fmt.Sprintf("file%d.txt", i)
		s := core.GenerateRandomString(rand.Intn(maxSize))
		err = CopyFrom(f, name, bytes.NewBufferString(s), &Attr{
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
