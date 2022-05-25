package cli

import (
	"babybluefs/store"
	"github.com/fatih/color"
)

func cp(args []string) {
	var source, dest string
	switch len(args) {
	case 0, 1:
		color.Green("source and destination are required")
		return
	case 2:
		source = args[0]
		dest = args[1]
	}

	from, _, fromPh, err := GetFS(source)
	if err != nil {
		color.Red("invalid source %s: %v", source, err)
		return
	}

	to, _, toPh, err := GetFS(dest)
	if err != nil {
		color.Red("invalid destination %s: %v", dest, err)
		return
	}
	to = store.NewConsoleMon(to)
	_ = copyAll(from, to, fromPh, toPh)
}

func completeCp(args []string) {
	switch len(args) {
	case 0:
		filterEscapeAndPrint("", true, completePath("")...)
	case 1:
		filterEscapeAndPrint("", true, completePath(args[1])...)
	case 2:
		filterEscapeAndPrint("", true, completePath(args[1])...)
	}
}
