package notfs

import (
	"context"
	"fmt"
	"github.com/beevik/ntp"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
	"io/fs"
	"path"
	"sort"
	"time"
)


func Sync(mesh *Mesh, name string, ignoreOlderThan time.Time, mon chan string) error {
	var err *multierror.Error

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	now := getTime()
	mesh.sync.Lock()
	remotes := copyRemotes(mesh)
	keys := copyKeys(mesh)
	local := mesh.Local
	lastSync := mesh.LastSync
	mesh.sync.Unlock()

	ec := make(chan error)
	for n, remote := range remotes {
		go func() {
			tm := ignoreOlderThan
			ec <- syncDir(name, local, remote, keys, now, tm, mon)
		}()
		select {
		case e := <-ec:
			if e == nil {
				mesh.sync.Lock()
				lastSync[n] = now
				mesh.sync.Unlock()
			}
			err = multierror.Append(e)
		case <-ctx.Done():
			return context.Canceled
		}
	}
	return err.ErrorOrNil()
}

const keyHashFile = ".keyHash"


func copyRemotes(mesh *Mesh) map[string]remote {
	var remotes = map[string]remote{}
	for n, r := range mesh.Remotes {
		remotes[n] = r
	}
	return remotes
}

func copyKeys(mesh *Mesh) Keys {
	var keys = Keys{}
	for n, b := range mesh.Keys {
		keys[n] = b
	}
	return keys
}

func listAndSortFiles(dir string, f FS, ignoreOlderThan time.Time, dirs map[string]bool) []fs.FileInfo {
	conflicts := make(map[string]bool)

	ls, _ := f.ReadDir(dir, 0)
	for _, l := range ls {
		ok, prefix, _, ext := parseConflict(l.Name())
		if ok {
			conflicts[fmt.Sprintf("%s%s", prefix, ext)] = true
			conflicts[l.Name()] = true
		}
	}

	var files []fs.FileInfo
	for _, l := range ls {
		if l.IsDir() {
			dirs[l.Name()] = true
		} else if l.ModTime().After(ignoreOlderThan) && !conflicts[l.Name()] {
			files = append(files, l)
		}
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	return files
}

type item struct {
	name string
	l    fs.FileInfo
	r    fs.FileInfo
	la   Attr
	ra   Attr
}

func hasAccess(remote remote, keys Keys) bool {
	if keys == nil {
		return true
	}
	_, ok := keys[remote.Group]
	return ok
}
func getTime() time.Time {
	for i := 0; i < 10; i++ {
		tm, err := ntp.Time("0.beevik-ntp.pool.ntp.org")
		if err == nil {
			return tm
		}
		time.Sleep(time.Duration(i)*time.Second)
	}
	return time.Now()
}

func collect(dir string, localFiles, remoteFiles []fs.FileInfo, local FS, remote remote,
	keys Keys) []item {
	i := 0
	j := 0
	var items []item
	for i < len(localFiles) || j < len(remoteFiles) {
		var l, r fs.FileInfo
		var la, ra Attr

		if i < len(localFiles) {
			l = localFiles[i]
		}
		if j < len(remoteFiles) {
			r = remoteFiles[j]
		}

		switch {
		case l != nil && r != nil && l.Name() == r.Name():
			n := path.Join(dir, l.Name())
			_ = GetMeta(local, n, &la)
			_ = GetMeta(remote.F, n, &ra)

			if hasAccess(remote, keys) {
				items = append(items, item{n, l, r, la, ra})
			}
			i++
			j++
		case r == nil || l != nil && l.Name() < r.Name():
			n := path.Join(dir, l.Name())
			_ = GetMeta(local, n, &la)
			_ = GetMeta(remote.F, n, &ra)
			if la.Group == remote.Group {
				items = append(items, item{n, l, nil, la, ra})
			}
			i++
		case l == nil || r != nil && l.Name() > r.Name():
			n := path.Join(dir, r.Name())
			_ = GetMeta(local, n, &la)
			_ = GetMeta(remote.F, n, &ra)
			if hasAccess(remote, keys) {
				items = append(items, item{n, nil, r, la, ra})
			}
			j++
		}
	}
	return items
}


func getEncryptedAccessToFile(r remote, keys Keys) FS {
	if keys == nil {
		return r.F
	}
	b := keys[r.Group]
	return NewEncrypted(r.F, b)
}

func sameContent(a, b Attr) bool {
	return a.CRC64s != nil && b.CRC64s != nil &&
		a.CRC64s[0] == b.CRC64s[0]
}

func deriveFrom(a,b Attr) bool {
	if b.CRC64s == nil {
		logrus.Debugf("Derived match because of empty target")
		return true
	}
	if a.CRC64s == nil {
		logrus.Debugf("No derived match because of empty source")
		return false
	}
	for idx, c := range a.CRC64s[1:] {
		if c == b.CRC64s[0] {
			logrus.Debugf("Derived match on item %d", idx+1)
			return true
		}
	}
	logrus.Debugf("No derived match because of mismatch")
	return false
}

func getAction(i item) string {
	if sameContent(i.la, i.ra) {
		switch  {
		case i.l == nil: {
			logrus.Debugf("Remote content for %s matches a deletion. Push a deletion", i.name)
			return "push"
		}
		case i.r == nil: {
			logrus.Debugf("Local content for %s matches a deletion. Pull a deletion", i.name)
			return "pull"
		}
		default: {
			logrus.Debugf("Identical content for %s. No action", i.name)
			return ""
		}
		}
	}

	if i.la.SyncTime.After(i.ra.SyncTime) {
		if i.r == nil || deriveFrom(i.la, i.ra) {
			logrus.Debugf("Local for %s is newer than remote. Push", i.name)
			return "push"
		} else {
			logrus.Debugf("Local for %s is newer than remote but content in in conflict. Conflict pull", i.name)
			return "conflict"
		}
	} else {
		if i.l == nil || deriveFrom(i.ra, i.la) {
			logrus.Debugf("Local for %s is older than remote. Pull", i.name)
			return "pull"
		} else {
			logrus.Debugf("Local for %s is older than remote but content in in conflict. Conflict pull", i.name)
			return "conflict"
		}
	}
}

func apply(i item, local FS, remote remote, keys Keys, mon chan string) error {
	logrus.Debugf("evaluate action for %s", i.name)
	switch getAction(i) {
	case "push":
		return pushFile(i, local, remote, keys, mon)
	case "pull":
		return pullFile(i, local, remote, false, keys, mon)
	case "conflict":
		return pullFile(i, local, remote, true, keys, mon)
	}
	return nil
}

func addZombies(dir string, local FS, remote remote, remoteFiles []fs.FileInfo) {
	zombies, _ := GetZombies(local, dir)
	for _, z := range zombies {
		stat, err := remote.F.Stat(path.Join(dir, z))
		if err == nil {
			remoteFiles = append(remoteFiles, stat)
		}
	}
	sort.Slice(remoteFiles, func(i, j int) bool {
		return remoteFiles[i].Name() < remoteFiles[j].Name()
	})
}

func syncDir(dir string, local FS, remote remote, keys Keys, now time.Time,
	ignoreOlderThan time.Time, mon chan string) error {
	var me *multierror.Error

	_ = ClearConflicts(local, dir, mon)

	dirs := make(map[string]bool)
	localFiles := listAndSortFiles(dir, local, ignoreOlderThan, dirs)
	remoteFiles := listAndSortFiles(dir, remote.F, ignoreOlderThan, dirs)

	syncLocalFiles(dir, local, now, localFiles)
//	addZombies(dir, local, remote, remoteFiles)
	items := collect(dir, localFiles, remoteFiles, local, remote, keys)
	for _, i := range items {
		logrus.Infof("process '%s', local: %v, remote: %v, la: %v, ra: %v ", i.name, i.l, i.r, i.la, i.ra)
		me = multierror.Append(me, apply(i, local, remote, keys, mon))
	}

	for d := range dirs {
		d = path.Join(dir, d)
		me = multierror.Append(me, syncDir(d, local, remote, keys, now, ignoreOlderThan, mon))
	}

	return me.ErrorOrNil()
}

func pushFile(i item, local FS, remote remote, keys Keys, mon chan string) error {
	var me *multierror.Error
	if i.l == nil {
		logrus.Infof("file %s removed from remote", i.name)
		return deleteFile(remote.F, i.name, i.la.ModifiedBy, mon)
	}
	r := getEncryptedAccessToFile(remote, keys)
	me = multierror.Append(Copy(local, r, i.name, i.name, false, time.Minute))
	me = multierror.Append(SetMeta(remote.F, i.name, i.la))

	if me.Len() == 0 {
		if mon != nil {
			logrus.Debugf("file %s pushed to remote", i.name)
			mon <- fmt.Sprintf("push,%s,%s,%x", i.name, i.la.ModifiedBy, i.la.CRC64s[0])
		}
	}
	return me
}

func pullFile(i item, local FS, remote remote, conflict bool, keys Keys, mon chan string) error {
	var me *multierror.Error
	if i.r == nil {
		logrus.Infof("file %s removed from local", i.name)
		return deleteFile(local, i.name, i.ra.ModifiedBy, mon)
	}

	var dest string
	if conflict {
		dir, base := path.Split(i.name)
		ext := path.Ext(base)
		dest = path.Join(dir, fmt.Sprintf("%s!!%s%x%s", base[0:len(base)-len(ext)],
			i.ra.ModifiedBy, i.ra.CRC64s[0] % 256, ext))
	} else {
		dest = i.name
	}

	r := getEncryptedAccessToFile(remote, keys)
	me = multierror.Append(me, Copy(r, local, i.name, dest, false, 0))
	me = multierror.Append(me, SetMeta(local, dest, i.ra))

	if me.Len() == 0 {
		logrus.Infof("file %s pulled from remote into %s", i.name, dest)
		if mon != nil {
			if conflict {
				mon <- fmt.Sprintf("conflict,%s,%s,%x", i.name, i.ra.ModifiedBy, i.ra.CRC64s[0])
			} else {
				mon <- fmt.Sprintf("pull,%s,%s,%x", i.name, i.ra.ModifiedBy, i.ra.CRC64s[0])
			}
		}
	}
	return me
}

func deleteFile(f FS, name, modifiedBy string, mon chan string) error {
	if mon != nil {
		mon <- fmt.Sprintf("delete,%s,%s,%x", name, modifiedBy, 0)
	}
	return f.Remove(name)
}

func syncLocalFiles(dir string, local FS, now time.Time, localFiles []fs.FileInfo) {

	for _, l := range localFiles {
		n := path.Join(dir, l.Name())
		_ = UpdateAttr(local, n, n, func(attr Attr) Attr {
			if l.ModTime().After(attr.SyncTime) {
				crc := calculateCRC64(local, n)
				if len(attr.CRC64s) == 0 || crc != attr.CRC64s[0] {
					if len(attr.CRC64s) > 16 {
						attr.CRC64s = append([]uint64{crc}, attr.CRC64s[0:15]...)
					} else {
						attr.CRC64s = append([]uint64{crc}, attr.CRC64s...)
					}
				}
				attr.SyncTime = now
			}
			return attr
		})
	}
}
