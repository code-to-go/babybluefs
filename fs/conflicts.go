package fs

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"io/fs"
	"path"
	"strings"
	"time"
)

type Item struct {
	Name    string    `json:"name"`
	Size    int64    `json:"size"`
	ModTime time.Time `json:"modTime"`
	Attr    Attr      `json:"attr"`
}

type Conflict struct {
	Dir   string `json:"dir"`
	Name  string `json:"name"`
	Items []Item `json:"items"`
}

func parseConflict(name string) (isIt bool, prefix, tag string, ext string) {
	ext = path.Ext(name)
	name = name[0 : len(name)-len(ext)]
	idx := strings.LastIndex(name, "!!")
	if idx != -1 {
		prefix = name[0:idx]
		tag = name[idx+2:]
		return true, prefix, tag, ext
	}
	return false, name, "", ext
}

func ClearConflicts(f FS, dir string, mon chan string) error {

	zombies, err := GetZombies(f, dir)
	if err != nil {
		return err
	}
	cs, err := getCollisions(f, dir)
	if err != nil {
		return err
	}

	var dead = map[string][]string{}
	var zCrc64s = map[string][]uint64{}
	for _, z := range zombies {
		_, zp, _, ext := parseConflict(z)
		c := cs[zp+ext]
		if len(c) == 1 {
			var attr Attr
			_ = GetMeta(f, path.Join(dir,z), &attr)
			if len(attr.CRC64s) > 0 {
				zCrc64s[zp+ext] = append(zCrc64s[zp+ext], attr.CRC64s[0])
			}
			dead[zp+ext] = append(dead[zp+ext], z)
		}
	}

	err = &multierror.Error{}
	for n, c := range cs {
		if len(c) == 1 && len(zCrc64s[n]) > 0{
			rn := c[0].Name()
			err := UpdateAttr(f, path.Join(dir, rn), path.Join(dir, n), func(attr Attr) Attr {
				crc64s := attr.CRC64s
				switch len(crc64s) {
				case 0: attr.CRC64s = zCrc64s[n]
				case 1: attr.CRC64s = append(crc64s[0:1], zCrc64s[n]...)
				default:
					attr.CRC64s = append(crc64s[0:1], zCrc64s[n]...)
					attr.CRC64s = append(attr.CRC64s, crc64s[1:]...)
				}
				return attr
			})
			if rn != n {
				e := f.Rename(path.Join(dir, rn), path.Join(dir, n))
				if e == nil && mon != nil {
					mon <- fmt.Sprintf("solve,%s,,0", path.Join(dir, rn))
				}
				err = multierror.Append(err, e)

			}
			if err == nil {
				for _, z := range dead[n] {
					_ = RemoveMeta(f, path.Join(dir, z))
				}
			}
		}
	}
	return err.(*multierror.Error).ErrorOrNil()
}

func SolveConflicts(f FS, dir string, names ...string) error {
	cs, err := getCollisions(f, dir)
	if err != nil {
		return err
	}
	for _, n := range names {
		_, prefix, _, ext := parseConflict(n)
		for _, file := range cs[prefix+ext] {
			if file.Name() != n {
				_ = f.Remove(path.Join(dir, file.Name()))
			}
		}
	}
	return ClearConflicts(f, dir, nil)
}

func getCollisions(f FS, dir string) (map[string][]fs.FileInfo, error) {
	ls, err := f.ReadDir(dir, 0)
	if err != nil {
		return nil, err
	}

	var dirs []string
	m := map[string][]fs.FileInfo{}
	for _, l := range ls {
		if l.IsDir() {
			dirs = append(dirs, l.Name())
			continue
		}
		name := l.Name()
		_, prefix, _, ext := parseConflict(name)
		m[prefix+ext] = append(m[prefix+ext], l)
	}

	return m, nil
}

func GetConflicts(f FS, dir string, recursive bool) ([]Conflict, error) {
	m, err := getCollisions(f, dir)
	if err != nil {
		return nil, err
	}

	var conflicts []Conflict
	for n, files := range m {
		if len(files) == 1 {
			continue
		}

		conflict := Conflict{
			Dir:  dir,
			Name: n,
		}

		for _, file := range files {
			var attr Attr
			_ = GetMeta(f, path.Join(dir, file.Name()), &attr)

			conflict.Items = append(conflict.Items, Item{
				Name:    file.Name(),
				Size:    file.Size(),
				ModTime: file.ModTime(),
				Attr:    attr,
			})
		}
		conflicts = append(conflicts, conflict)
	}

	me := &multierror.Error{}
	if recursive {
		ls, err := f.ReadDir(dir, 0)
		if err != nil {
			me = multierror.Append(me, err)
		} else {
			for _, l := range ls {
				if l.IsDir() {
					cs, err := GetConflicts(f, path.Join(dir, l.Name()), recursive)
					if err == nil {
						conflicts = append(conflicts, cs...)
					} else {
						me = multierror.Append(me, err)
					}
				}
			}
		}
	}

	return conflicts, me.ErrorOrNil()
}
