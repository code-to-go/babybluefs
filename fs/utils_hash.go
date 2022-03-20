package fs

import (
	"crypto/sha256"
	"io"
	"os"
)

func GetHash(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	h256 := sha256.New()
	if _, err := io.Copy(h256, file); err != nil {
		return nil, err
	}
	return h256.Sum(nil), nil
}


