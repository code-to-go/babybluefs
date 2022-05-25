package cli

import (
	"babybluefs/store"
	"fmt"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

var fsCache = map[string]store.FS{}
var home string

func GetHome() string {
	if home != "" {
		return home
	}

	home = os.Getenv("SF_HOME")
	if home == "" {
		configDir, _ := os.UserConfigDir()
		home = filepath.Join(configDir, "babybluefs")
		_ = os.MkdirAll(home, 0755)
	}
	return home
}

func isLocalPath(ph string) bool {
	if strings.HasPrefix(ph, ".") || strings.HasPrefix(ph, string(os.PathSeparator)) {
		return true
	}

	if runtime.GOOS == "windows" && len(ph) >= 2 && ph[1] == ':' {
		return true
	}

	return false
}

func GetFS(ph string) (f store.FS, name, ph2 string, err error) {
	if isLocalPath(ph) {
		dir, ph2 := filepath.Split(ph)
		return store.NewLocalMount(dir), ".", ph2, nil
	}

	name = strings.Split(ph, "/")[0]

	var found bool
	if f, found = fsCache[name]; !found {
		home := GetHome()
		logrus.Infof("home is '%s'", home)

		var c store.Config
		l := store.NewLocalMount(home)
		err := store.ReadYaml(l, fmt.Sprintf("%s.yaml", name), &c)
		if err != nil {
			color.Red("store '%s' not defined", name)
			logrus.Infof("cannot load '%s' from '%s': %v", name, home, err)
			return nil, "", "", err
		}

		f, err = store.NewFS(c)
		if err != nil {
			color.Red("connection fail on store '%s': %v", name, err)
			logrus.Infof("cannot connect to load '%s': %v", name, err)
			return nil, name, "", err
		}

		fsCache[name] = f
	}

	if len(name) == len(ph) {
		return f, name, "", nil
	} else {
		return f, name, cleanSuffixSlash(ph[len(name):]), nil
	}
}

func copyAll(from, to store.FS, fromPh, toPh string) error {
	statFrom, err := from.Stat(fromPh)
	if err != nil {
		color.Red("cannot access '%s': %v", fromPh, err)
		return err
	}

	if statFrom.IsDir() {
		statTo, err := to.Stat(toPh)
		if err != nil || !statTo.IsDir() {
			color.Red("cannot create folder in '%v/%s'", to, toPh)
			return err
		}

		toPh = path.Join(toPh, path.Base(fromPh))
		_ = to.MkdirAll(toPh)

		ls, _ := from.ReadDir(fromPh, store.IncludeHiddenFiles)
		for _, l := range ls {
			_ = copyAll(from, to, path.Join(fromPh, l.Name()), toPh)
		}
		return nil
	}

	if strings.HasSuffix(toPh, "/") {
		toPh = path.Join(toPh, path.Base(fromPh))
	} else {
		statTo, err := to.Stat(toPh)
		if err == nil && statTo.IsDir() {
			toPh = path.Join(toPh, path.Base(fromPh))
		}
	}

	err = store.Copy(from, to, fromPh, toPh, true, 0)
	if err != nil {
		color.Red("cannot copy '%s' to '%s': %v", fromPh, toPh, err)
		return err
	}

	color.Green("%v/%s -> %v/%s", from, fromPh, to, toPh)
	return nil
}

func CloseAllFss() {
	for _, f := range fsCache {
		_ = f.Close()
	}

	fsCache = map[string]store.FS{}
}

func cleanSuffixSlash(s string) string {
	for strings.HasPrefix(s, "/") {
		s = s[1:]
	}
	return s
}
