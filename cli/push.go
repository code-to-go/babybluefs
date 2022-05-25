package cli

import (
	"babybluefs/store"
	"github.com/fatih/color"
	"path/filepath"
)

func Push(args []string) {
	var remote, local string
	switch len(args) {
	case 0, 1:
		color.Green("nothing to push")
		return
	case 2:
		local = args[0]
		remote = args[1]
	}

	from := store.NewLocalMount(filepath.Dir(local))
	fromPh := filepath.Base(local)

	to, _, toPh, err := GetFS(remote)
	if err != nil {
		return
	}
	to = store.NewConsoleMon(to)
	_ = copyAll(from, to, fromPh, toPh)
}

func completePush(args []string) {
	switch len(args) {
	case 0:
		filterEscapeAndPrint("", true, getLocals("")...)
	case 1:
		filterEscapeAndPrint("", true, getLocals(args[0])...)
	case 2:
		filterEscapeAndPrint("", true, completePath(args[1])...)
	}
}
