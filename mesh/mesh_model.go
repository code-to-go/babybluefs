package mesh

import (
	"babybluefs/store"
	"crypto/cipher"
	"sync"
	"time"
)

type remote struct {
	F     store.FS
	Group store.Group
}

type Keys map[store.Group]cipher.Block

type Mesh struct {
	Keys         Keys
	Local        store.FS
	Remotes      map[string]remote
	RemotesState map[string]string
	LastSync     map[string]time.Time
	Zombie       bool
	sync         sync.Mutex
}
