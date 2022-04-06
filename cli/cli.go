package cli

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"path/filepath"
	"stratofs/store"
	"strings"
)

func GetHome() string {
	home := os.Getenv("SF_HOME")
	if home == "" {
		configDir, _ := os.UserConfigDir()
		home = filepath.Join(configDir, "stratofs")
		_ = os.MkdirAll(home, 0755)
	}
	return home
}

func getFS(ph string) (f store.FS, ph2 string) {
	name := strings.Split(ph, "/")[0]
	home := GetHome()
	logrus.Infof("home is '%s'", home)

	var c store.Config
	l := store.NewLocalMount(home)
	err := store.ReadYaml(l, fmt.Sprintf("%s.yaml", name), &c)
	if err != nil {
		color.Red("remote '%s' not found", name)
		logrus.Infof("cannot load '%s' from '%s': %v", name, home, err)
		os.Exit(1)
	}

	f, err = store.NewFS(c)
	if err != nil {
		color.Red("connection fail on '%s': %v", name, err)
		logrus.Infof("cannot connect to load '%s': %v", name, err)
		os.Exit(1)
	}

	return f, ph[len(name):]
}

func List(remote string, hidden bool) {
	var opts store.ListOption
	if hidden {
		opts = store.IncludeHiddenFiles
	}

	f, ph := getFS(remote)
	ls, err := f.ReadDir(ph, opts)
	if err != nil {
		l, err := f.Stat(ph)
		if err == nil {
			ls = append(ls, l)
		} else {
			color.Red("listCmd '%s': %v", ph, err)
			return
		}
	}

	color.Green("Type\tMod\tSize\tName\n")
	for _, l := range ls {
		var ty string
		if l.IsDir() {
			ty = "dir"
		} else {
			ty = "file"
		}
		color.Green("%s\t%s\t%d\t%s\n", ty, l.ModTime(), l.Size(), l.Name())
	}
}

func pullFolder(f store.FS, ph, local string) {
	p := make(chan store.Progress)
	f = store.NewMon(f, p)
	stat, err := f.Stat(ph)
	if err != nil {
		color.Red("cannot access '%s': %v", ph, err)
		os.Exit(1)
	}

	if stat.IsDir() {
		stat2, err := os.Stat(local)
		if err != nil || !stat2.IsDir() {
			color.Red("cannot create folder in file '%s'", local)
			os.Exit(1)
		}

		local = filepath.Join(local, path.Base(ph))
		os.MkdirAll(local, 0755)

		ls, _ := f.ReadDir(ph, store.IncludeHiddenFiles)
		for _, l := range ls {
			pullFolder(f, path.Join(ph, l.Name()), local)
		}
		return
	}

	stat2, err := os.Stat(local)
	if os.IsNotExist(err) {
		n := filepath.Dir(local)
		err = os.MkdirAll(n, 0755)
		color.Green("! %s", n)
	}
	if err != nil {
		color.Red("cannot create '%s': %v", local, err)
	}

	if stat2.IsDir() {
		local = filepath.Join(local, path.Base(ph))
	}

	w, err := os.Create(local)
	if err != nil {
		color.Red("cannot write '%s': %v", local, err)
		os.Exit(1)
	}
	defer w.Close()
	err = f.Pull(ph, w)
	if err != nil {
		color.Red("cannot pull '%s': %v", ph, err)
		os.Exit(1)
	}

	color.Green("<- '%s' in '%s'", ph, local)
}

func Pull(remote, local string) {
	f, ph := getFS(remote)
	pullFolder(f, ph, local)
}

func Push(local, remote string) {
	f, ph := getFS(remote)
	r, err := os.Open(local)
	if err != nil {
		color.Red("cannot read '%s': %v", local, err)
		os.Exit(1)
	}
	defer r.Close()
	err = f.Push(ph, r)
	if err != nil {
		color.Red("cannot push '%s': %v", remote, err)
		os.Exit(1)
	}

	color.Green("push '%s' in '%s'", remote, local)
}
