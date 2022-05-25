package cli

import (
	"babybluefs/store"
	"github.com/fatih/color"
	"os"
	"path/filepath"
)

func Pull(args []string) {
	var url, local string
	switch len(args) {
	case 0:
		color.Green("nothing to pull")
		return
	case 1:
		url = args[0]
		local, _ = os.Getwd()
	case 2:
		url = args[0]
		local = args[1]
	}

	from, _, fromPh, err := GetFS(url)
	if err != nil {
		return
	}

	to := store.NewLocalMount(filepath.Dir(local))
	toPh := filepath.Base(local)
	to = store.NewConsoleMon(to)
	_ = copyAll(from, to, fromPh, toPh)
}

func completePull(args []string) {
	switch len(args) {
	case 0:
		filterEscapeAndPrint("", true, completePath("")...)
	case 1:
		filterEscapeAndPrint("", true, completePath(args[0])...)
	case 2:
		filterEscapeAndPrint("", true, getLocals(args[1])...)
	}
}
