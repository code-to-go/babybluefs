package main

import (
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
)

func usage() {
	fmt.Printf("usage: stratofs <command> [<args>]\n\n" +
		"These are the common commands.\n" +
		"\tlistCmd remote                             listCmd the remote path\n" +
		"\tpush local remote                       copy file to remote\n" +
		"\tpull remote local                       copy file from remote\n" +
		"\tcopy remote1 remote2                    copy files from remote1 to remote2\n" +
		"\tcreate [s3|azure|sftp|ftp|sharepoint]   create a new connection\n" +
		"\t-v                                      shows verbose log\n" +
		"\t-vv                                     shows a very verbose log\n")
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
	"listCmd":   2,
	"pull":   3,
	"push":   3,
	"create": 2,
}

func checkArgs(args []string) {
	if len(args) < 2 {
		flag.Usage()
		os.Exit(0)
	}
	if len(args) < argsMinLen[args[0]] {
		flag.Usage()
		os.Exit(0)
	}
}

func main() {
	var verbose bool
	var verbose2 bool
	var hidden bool

	flag.Usage = usage
	flag.BoolVar(&verbose, "v", false,
		"shows verbose log")
	flag.BoolVar(&verbose2, "vv", false,
		"shows very verbose log")
	flag.BoolVar(&hidden, "h", false,
		"shows hidden files")

	flag.Parse()
	nArg := flag.NArg()
	commands := os.Args[len(os.Args)-nArg:]

	cl, completion := os.LookupEnv("COMP_LINE")
	if completion {
		complete(cl)
		return
	}

	checkArgs(commands)
	setLogLevel(verbose, verbose2)

	switch commands[0] {
	case "listCmd":
		listCmd(commands[1], hidden)
	case "pull":
		pull(commands[1], commands[2])
	case "push":
		push(commands[1], commands[2])
	case "create":
		create(commands[1])
	}
}
