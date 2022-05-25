package cli

import (
	"babybluefs/store"
	"github.com/fatih/color"
	"io/fs"
	"io/ioutil"
	"os"
	"strings"
)

func List(args []string) {
	if len(args) == 0 {
		color.Green(strings.Join(completePath(""), " "))
		return
	}

	var ls []fs.FileInfo
	var err error
	ph := args[0]
	if strings.HasPrefix(ph, ".") || strings.HasPrefix(ph, string(os.PathSeparator)) {
		ls, err = ioutil.ReadDir(ph)
	} else {
		var f store.FS
		f, _, ph, err = GetFS(ph)
		if err != nil {
			return
		}
		ls, err = f.ReadDir(ph, store.IncludeHiddenFiles)
		if err != nil {
			l, err := f.Stat(ph)
			if err == nil {
				ls = append(ls, l)
			} else {
				color.Red("listCmd '%s': %v", ph, err)
				return
			}
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
