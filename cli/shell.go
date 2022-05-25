package cli

import (
	"fmt"
	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"io"
	"path/filepath"
	"strings"
)

func completeStoreTypes(string) []string {
	return storeTypes
}
func completeStoreList(string) []string {
	return getRemotes()
}

func completePhAt(s string, loc int) []string {
	ss := strings.Split(s, " ")

	var ss2 []string
	for _, w := range ss {
		if w != "" {
			ss2 = append(ss2, w)
		}
	}
	if loc < len(ss2) {
		return completePath(ss2[loc])
	} else {
		return completePath("")
	}
}

func completePath1(s string) []string { return completePhAt(s, 1) }
func completePath2(s string) []string { return completePhAt(s, 2) }

var completer = readline.NewPrefixCompleter(
	readline.PcItem("ls", readline.PcItemDynamic(completePath1)),
	readline.PcItem("cp", readline.PcItemDynamic(completePath1, readline.PcItemDynamic(completePath2))),
	readline.PcItem("create", readline.PcItemDynamic(completeStoreTypes)),
	readline.PcItem("echo", readline.PcItemDynamic(completePath1)),
	readline.PcItem("cat", readline.PcItemDynamic(completePath1)),
	readline.PcItem("mkdir", readline.PcItemDynamic(completePath1)),
	readline.PcItem("edit", readline.PcItemDynamic(completeStoreList)),
)

func shouldExit(line string, err error) bool {
	if err == readline.ErrInterrupt {
		return len(line) == 0
	} else {
		return err == io.EOF
	}
}

func shellUsage() {
	fmt.Printf(
		"\tls store                            list the remote path\n" +
			"\tpush local store                        copy local file to a store\n" +
			"\tpull store local                        copy local file from a store\n" +
			"\tcopy store1 store2                      copy files from one store to another\n" +
			"\trm store                                delete a file\n" +
			"\tcreate [s3|azure|sftp|ftp|sharepoint]   create a new store configuration\n" +
			"\tedit store                              edit an existing store configuration\n" +
			"\tmesh name [storage...]                  create a mesh with provided storage list\n" +
			"\tsync mesh                               align all the storage points in the mesh\n")

}

func Shell() {
	home := GetHome()

	l, _ := readline.NewEx(&readline.Config{
		Prompt:          "\033[31m»\033[0m ",
		HistoryFile:     filepath.Join(home, "history"),
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",

		HistorySearchFold: true,
	})

	color.Green("Welcome to StratoFS")
	exit := false
	for !exit {
		line, err := l.Readline()
		if shouldExit(line, err) {
			break
		}

		line = strings.TrimSpace(line)
		args := strings.Split(line, " ")
		switch args[0] {
		case "ls":
			List(args[1:])

		case "cp":
			cp(args[1:])
		case "echo":
			echo(args[1:])
		case "cat":
			cat(args[1:])
		case "create":
			Create(args[1])
		case "edit":
			Edit(args[1])
		case "mkdir":
			Mkdir(args[1:])
		case "exit":
			exit = true
		default:
			shellUsage()
		}
	}

	CloseAllFss()
}
