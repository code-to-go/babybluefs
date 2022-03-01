package sfs

import (
	"github.com/gabriel-vasile/mimetype"
)

func Mime(f FS, name string) *mimetype.MIME {
	d, _ := Peek(f, name, 512)
	return mimetype.Detect(d)
}
