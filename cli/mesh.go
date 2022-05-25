package cli

import (
	"babybluefs/mesh"
	"babybluefs/store"
	"fmt"
	"github.com/fatih/color"
	"os"
	"path/filepath"
	"time"
)

func Mesh(target string, names []string) {
	var groups map[store.Group]bool
	var mc mesh.Config

	f := store.NewLocalMount(GetHome())
	mc, _ = mesh.ReadConfig(f, fmt.Sprintf("%s.yaml", target))

	for _, n := range names {
		var c store.Config
		err := store.ReadYaml(f, fmt.Sprintf("%s.yaml", n), &c)
		if err != nil {
			color.Red("cannot parse config %s: %v", n, err)
			os.Exit(1)
		}

		groups[c.Group] = true
		mc.Remotes = append(mc.Remotes, c)
	}

	for group := range groups {
		mc.Groups[group] = store.GenerateRandomString(32)
	}

	err := mesh.WriteConfig(f, target, mc)
	if err != nil {
		color.Red("cannot create mesh config in %s: %v", target, err)
		os.Exit(1)
	}

	color.Green("new mesh config %s", target)
}

func Sync(meshName, folder string) {
	f := store.NewLocalMount(GetHome())

	var mc mesh.Config
	mc, err := mesh.ReadConfig(f, fmt.Sprintf("%s.yaml", meshName))
	if err != nil {
		color.Red("cannot read mesh config %s: %v", meshName, err)
		os.Exit(1)
	}
	var m mesh.Mesh
	err = mesh.FromConfig(mc, &m, false)
	if err != nil {
		color.Red("cannot create mesh %s: %v", meshName, err)
		os.Exit(1)
	}

	folder, _ = filepath.Abs(folder)
	stat, err := os.Stat(folder)
	if !stat.IsDir() {
		color.Red("%s must be a folder", folder)
		os.Exit(1)
	}

	m.Local = store.NewLocalMount(folder)
	err = mesh.Sync(&m, "", time.Time{}, nil)

	color.Green("sync completed %s", meshName)
}
