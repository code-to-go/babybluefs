package notfs

import (
	"crypto/cipher"
	"sync"
	"time"
)

type remote struct {
	F     FS
	Group Group
}

type Keys map[Group]cipher.Block

type Mesh struct {
	Keys    Keys
	Local   FS
	Remotes map[string]remote
	RemotesState map[string]string
	LastSync     map[string]time.Time
	Zombie       bool
	sync         sync.Mutex
}
