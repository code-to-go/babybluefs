package fs

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"path"
	"reflect"
	"strings"
)

type MetaBlob map[string][]byte

func metaName(name string) string {
	dir, name := path.Split(name)
	return path.Join(dir, fmt.Sprintf(".%s!.meta", name))
}

func IsMeta(name string) bool {
	return strings.HasSuffix(name, "!.meta")
}

func SetMeta(f FS, name string, metas ...interface{}) error {
	m := make(MetaBlob)
	name = metaName(name)
	bs := new(bytes.Buffer)
	err := f.Pull(name, bs)
	if err == nil {
		_ = gob.NewDecoder(bs).Decode(&m)
	}

	for _, meta := range metas {
		t := strings.Trim(reflect.TypeOf(meta).String(), "*")
		buf := new(bytes.Buffer)
		err = gob.NewEncoder(buf).Encode(meta)
		if err != nil {
			return err
		}
		m[t] = buf.Bytes()
	}
	buf := new(bytes.Buffer)
	err = gob.NewEncoder(buf).Encode(m)
	if err != nil {
		return err
	}

	return f.Pull(name, buf)
}


func GetMeta(f FS, name string, metas ...interface{}) error {
	m := make(MetaBlob)
	name = metaName(name)

	bs := new(bytes.Buffer)
	err := f.Pull(name, bs)
	if err != nil {
		return err
	}

	err = gob.NewDecoder(bs).Decode(&m)
	if err != nil {
		return err
	}
	err = &multierror.Error{}
	for _, meta := range metas {
		t := strings.Trim(reflect.TypeOf(meta).String(), "*")
		if v, ok := m[t]; ok {
			d := gob.NewDecoder(bytes.NewBuffer(v))
			err = multierror.Append(err, d.Decode(meta))
		}
	}

	return err.(*multierror.Error).ErrorOrNil()
}


func RemoveMeta(f FS, name string) error {
	return f.Remove(metaName(name))
}

func GetZombies(f FS, name string) ([]string, error) {
	live := make(map[string]bool)

	ls, err := f.ReadDir(name, IncludeHiddenFiles)
	if err != nil {
		return nil, err
	}

	for _, l := range ls {
		name = l.Name()
		if IsMeta(name) {
			orig := name[1:len(name)-len("!.meta")]
			live[orig] = live[orig] || false
		} else {
			live[name] = live[name] || true
		}
	}

	var zombies []string
	for n, l := range live {
		if !l {
			zombies = append(zombies, n)
		}
	}
	return zombies, nil
}

func PurgeZombies(f FS, name string) error {
	zombies, err := GetZombies(f, name)
	if err != nil {
		return err
	}

	err = multierror.Append(err)
	for _, zombie := range zombies {
		err = multierror.Append(err, f.Remove(metaName(zombie)))
	}
	return err.(*multierror.Error).ErrorOrNil()
}