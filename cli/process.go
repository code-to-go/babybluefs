package cli

import (
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
)

func usage() {
	home := GetHome()

	fmt.Printf("usage: bbfs <command> [<args>]\n\n"+
		"These are the common commands.\n"+
		"\tls remote                               list the remote path\n"+
		"\tpush local store                        copy local file to a store\n"+
		"\tpull store local                        copy local file from a store\n"+
		"\tcopy store1 store2                      copy files from one store to another\n"+
		"\tcreate [s3|azure|sftp|ftp|sharepoint]   create a new store configuration\n"+
		"\tedit store                              edit an existing store configuration\n"+
		"\tmesh name [storage...]                  create a mesh with provided storage list\n"+
		"\tsync mesh                               align all the storage points in the mesh\n"+
		"\t-v                                      shows verbose log\n"+
		"\t-vv                                     shows a very verbose log\n\n"+
		"Configuration will be stored in %s. Define SF_HOME variable for a different location\n\n", home)
}

func setLogLevel(verbose, verbose2 bool) {
	switch {
	case verbose:
		logrus.SetLevel(logrus.InfoLevel)
	case verbose2:
		logrus.SetLevel(logrus.DebugLevel)
	default:
		logrus.SetLevel(logrus.WarnLevel)
	}
}

var argsMinLen = map[string]int{
	"ls":     1,
	"pull":   3,
	"push":   3,
	"create": 2,
	"edit":   2,
	"mesh":   2,
	"shell":  1,
	"mkdir":  2,
	"rm":     2,
}

func checkArgs(args []string) {
	if len(args) < 1 {
		flag.Usage()
		os.Exit(0)
	}
	if len(args) < argsMinLen[args[0]] {
		flag.Usage()
		os.Exit(0)
	}
}

func Process() {
	var verbose bool
	var verbose2 bool

	flag.Usage = usage
	flag.BoolVar(&verbose, "v", false,
		"shows verbose log")
	flag.BoolVar(&verbose2, "vv", false,
		"shows very verbose log")

	flag.Parse()
	nArg := flag.NArg()
	commands := os.Args[len(os.Args)-nArg:]

	cl, completion := os.LookupEnv("COMP_LINE")
	if completion {
		Complete(cl)
		return
	}

	checkArgs(commands)
	setLogLevel(verbose, verbose2)

	switch commands[0] {
	case "ls":
		List(commands[1:])
	case "pull":
		Pull(commands[1:])
	case "push":
		Push(commands[1:])
	case "create":
		Create(commands[1])
	case "edit":
		Edit(commands[1])
	case "mesh":
		Mesh(commands[1], commands[2:])
	case "shell":
		Shell()
	case "mkdir":
		Mkdir(commands[1:])
	}
}
