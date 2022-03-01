package sfs

import (
	"archive/zip"
	"bytes"
	"path/filepath"
)

func UnzipFile(f FS, dest string, data []byte) ([]string, error) {
	var names []string

	reader := bytes.NewReader(data)
	r, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		return nil, err
	}

	for _, file := range r.File {
		name := filepath.Join(dest, file.Name)
		names = append(names, name)
		if file.FileInfo().IsDir() {
			_ = f.MkdirAll(name)
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}

		err = f.Push(name, rc)
	}
	return names, err
}

