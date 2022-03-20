package mesh

import (
	"crypto/cipher"
	"stratofs/fs"
	"sync"
	"time"
)

type remote struct {
	F     fs.FS
	Group fs.Group
}

type Keys map[fs.Group]cipher.Block

type Mesh struct {
	Keys         Keys
	Local        fs.FS
	Remotes      map[string]remote
	RemotesState map[string]string
	LastSync     map[string]time.Time
	Zombie       bool
	sync         sync.Mutex
}
