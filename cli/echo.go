package cli

import (
	"bytes"
	"github.com/fatih/color"
)

func echo(args []string) {
	if len(args) < 3 {
		color.Green("need some content, | and the destinations")
		return
	}

	var content string
	var dests []string

	pf := false
	for _, arg := range args {
		if arg == "" {
			continue
		}
		if arg == ">" {
			pf = true
		} else if pf {
			dests = append(dests, arg)
		} else {
			if len(content) > 0 {
				content += " "
			}
			content += arg
		}
	}

	for _, dest := range dests {
		buf := bytes.NewBufferString(content)
		to, _, toPh, err := GetFS(dest)
		if err != nil {
			color.Red("invalid destination %s: %v", dest, err)
			continue
		}
		err = to.Push(toPh, buf)
		if err != nil {
			color.Red("cannot echo to %s: %v", dest, err)
		}
	}
}

func completeEcho(args []string) {
	switch len(args) {
	case 0:
		filterEscapeAndPrint("", true, completePath("")...)
	case 1:
		filterEscapeAndPrint("", true, completePath(args[1])...)
	case 2:
		filterEscapeAndPrint("", true, completePath(args[1])...)
	}
}
