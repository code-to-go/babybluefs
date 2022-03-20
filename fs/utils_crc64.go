package fs

import (
	"hash"
	"hash/crc64"
)

type ChecksumWriter struct {
	h hash.Hash64
}

func (w *ChecksumWriter) Write(bs []byte) (int, error) {
	return w.h.Write(bs)
}

func CalculateCRC64(fs FS, file string) uint64 {
	w := &ChecksumWriter{
		h: crc64.New(crc64.MakeTable(crc64.ECMA)),
	}
	_ = fs.Pull(file, w)

	return w.h.Sum64()
}
