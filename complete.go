package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"stratofs/sfs"
	"strconv"
	"strings"
)

func filterEscapeAndPrint(filter string, escape bool, l ...string) {
	for _, t := range l {
		if strings.HasPrefix(t, filter) && t != filter {
			if escape {
				fmt.Println(strings.ReplaceAll(t, " ", "\\ "))
			} else {
				fmt.Println(t)
			}
		}
	}
}

func getRemoteList(home, remote, ph string) []string {
	data, err := ioutil.ReadFile(filepath.Join(home, fmt.Sprintf("%s.yaml", remote)))
	if err != nil {
		return nil
	}

	var c sfs.FSConfig
	err = yaml.Unmarshal(data, &c)
	if err != nil {
		return nil
	}

	f, err := sfs.NewFS(c)
	if err != nil {
		return nil
	}

	dir := path.Dir(ph)
	filter := ph[len(dir):]
	filter = strings.Trim(filter, "/")
	//fmt.Println(dir+"::"+filter)
	//fmt.Println("...")

	var remotes []string
	ls, _ := f.ReadDir(dir, sfs.IncludeHiddenFiles)
	for _, l := range ls {
		if strings.HasPrefix(l.Name(), filter) {
			if l.IsDir() {
				remotes = append(remotes,
					fmt.Sprintf("%s/", path.Join(remote, dir, l.Name())))
			} else {
				remotes = append(remotes, path.Join(remote, dir, l.Name()))
			}
		}
	}
	return remotes
}

func getRemotes(startWith string) []string {
	home := os.Getenv("SF_HOME")
	if home == "" {
		home, _ = os.Getwd()
	}

	var remotes []string
	ls, _ := ioutil.ReadDir(home)
	for _, l := range ls {
		ext := filepath.Ext(l.Name())
		if ext == ".yaml" {
			name := l.Name()[0 : len(l.Name())-len(ext)]
			if strings.HasPrefix(name, startWith) {
				remotes = append(remotes, fmt.Sprintf("%s/",name))
			}
			if strings.HasPrefix(startWith, name) {
				remotes = append(remotes, getRemoteList(home, name, startWith[len(name):])...)
			}
		}
	}
	return remotes
}

func getLocals(startWith string) []string{
	dir := filepath.Dir(startWith)
	var filter string

	if dir == "." {
		filter = startWith
	} else {
		filter = startWith[len(dir):]
	}
	filter = strings.Trim(filter, "." )
	filter = strings.Trim(filter, strconv.QuoteRune(filepath.Separator))

	ls, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil
	}

	var locals []string
	for _, l := range ls {
		if strings.HasPrefix(l.Name(), filter) {
			if l.IsDir() {
				locals = append(locals,
					fmt.Sprintf("%s%c",
						filepath.Join(dir, l.Name()), filepath.Separator))
			} else {
				locals = append(locals,
						filepath.Join(dir, l.Name()))
			}
		}
	}
	return locals
}

func completeList(args []string) {
	switch len(args) {
	case 0:
		filterEscapeAndPrint("", true, getRemotes("")...)
	case 1:
		filterEscapeAndPrint("", true, getRemotes(args[0])...)
	}
}

func completePull(args []string) {
	switch len(args) {
	case 0:
		filterEscapeAndPrint("", true, getRemotes("")...)
	case 1:
		filterEscapeAndPrint("", true, getRemotes(args[0])...)
	case 2:
		filterEscapeAndPrint("", true, getLocals(args[1])...)
	}
}

func completePush(args []string) {
	switch len(args) {
	case 0:
		filterEscapeAndPrint("", true, getLocals("")...)
	case 1:
		filterEscapeAndPrint("", true, getLocals(args[0])...)
	case 2:
		filterEscapeAndPrint("", true, getRemotes(args[1])...)
	}
}

func completeCreate(args []string) {
	filter := ""
	if len(args) > 0 {
		filter = args[0]
	}

	filterEscapeAndPrint(filter, false, "s3 ", "azure ", "sftp ", "ftp ", "sharepoint ")
}

func complete(cl string) {
	args := strings.Split(cl, " ")

	if len(args) < 2 {
		filterEscapeAndPrint("", false, "pull ", "push ", "list ", "create ")
		return
	}

	switch args[1] {
	case "list":
		completeList(args[2:])
	case "pull":
		completePull(args[2:])
	case "push":
		completePush(args[2:])
	case "create":
		completeCreate(args[2:])
	default:
		filterEscapeAndPrint(args[1], false, "pull ", "push ", "list ", "create ")
	}
}
