package cli

import (
	"github.com/fatih/color"
)

func Mkdir(args []string) {
	if len(args) < 1 {
		color.Green("missing target")
		return
	}

	target := args[0]
	to, _, toPh, err := GetFS(target)
	if err != nil {
		return
	}

	err = to.MkdirAll(toPh)
	if err == nil {
		color.Green("%s created", target)
	} else {
		color.Red("cannot create %s: %v", target, err)
	}
}
