package cli

import (
	"bytes"
	"github.com/fatih/color"
)

func cat(args []string) {
	if len(args) < 1 {
		color.Green("need target")
		return
	}

	source := args[0]

	from, _, fromPh, err := GetFS(source)
	if err != nil {
		color.Red("invalid source %s: %v", source, err)
		return
	}

	buf := bytes.Buffer{}
	err = from.Pull(fromPh, &buf)
	if err != nil {
		color.Red("cannot cat %s: %v", source, err)
	} else {
		color.Green(buf.String())
	}
}

func completeCat(args []string) {
	switch len(args) {
	case 0:
		filterEscapeAndPrint("", true, completePath("")...)
	case 1:
		filterEscapeAndPrint("", true, completePath(args[1])...)
	}
}
