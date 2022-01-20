package notfs

import (
	"time"
)


type Attr struct {
	ModifiedBy string    `json:"modifiedBy"`
	Group      Group     `json:"group"`
	SyncTime   time.Time `json:"crcTime"`
	CRC64s   []uint64  `json:"crc64s"`
}

func UpdateAttr(f FS, src string, dest string, update func(attr Attr) Attr) error {
	var attr Attr
	_ = GetMeta(f, src, &attr)
	attr = update(attr)
	return SetMeta(f, dest, attr)
}

