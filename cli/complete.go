package cli

import (
	"babybluefs/store"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
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

func getStoreList(prefix string, f store.FS, ph string) []string {
	dir, filter := path.Split(ph)
	filter = strings.Trim(filter, "/")
	for strings.HasPrefix(dir, "/") {
		dir = dir[1:]
	}

	var storeList []string
	ls, _ := f.ReadDir(dir, store.IncludeHiddenFiles)
	for _, l := range ls {
		if strings.HasPrefix(l.Name(), filter) {
			if l.IsDir() {
				storeList = append(storeList,
					fmt.Sprintf("%s%s/", prefix, path.Join(dir, l.Name())))
			} else {
				storeList = append(storeList, prefix+path.Join(dir, l.Name()))
			}
		}
	}
	return storeList
}

func completePath(startWith string) []string {
	if isLocalPath(startWith) {
		dir, ph := filepath.Split(startWith)
		f := store.NewLocalMount(dir)
		return getStoreList(dir, f, ph)
	}

	home := GetHome()
	var phs []string
	ls, _ := ioutil.ReadDir(home)
	for _, l := range ls {
		ext := filepath.Ext(l.Name())
		if ext == ".yaml" {
			name := l.Name()[0 : len(l.Name())-len(ext)]
			if strings.HasPrefix(name, startWith) {
				phs = append(phs, fmt.Sprintf("%s/", name))
			}
			if strings.HasPrefix(startWith, name) {
				f, name, ph, err := GetFS(startWith)
				if err == nil {
					phs = append(phs, getStoreList(name+"/", f, ph)...)
				}
			}
		}
	}

	if startWith == "" {
		phs = append(phs, "."+string(os.PathSeparator))
	}

	return phs
}

func getLocals(startWith string) []string {
	dir := filepath.Dir(startWith)
	var filter string

	if dir == "." {
		filter = startWith
	} else {
		filter = startWith[len(dir):]
	}
	filter = strings.Trim(filter, ".")
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
		filterEscapeAndPrint("", true, completePath("")...)
	case 1:
		filterEscapeAndPrint("", true, completePath(args[0])...)
	}
}

var storeTypes = []string{"s3", "smb", "azure", "sftp", "ftp", "sharepoint", "kafka"}

func completeCreate(args []string) {
	filter := ""
	if len(args) > 0 {
		filter = args[0]
	}

	var types []string
	for _, t := range storeTypes {
		types = append(types, t+" ")
	}
	filterEscapeAndPrint(filter, false, types...)
}

func getRemotes() []string {
	home := GetHome()

	var remotes []string
	ls, _ := ioutil.ReadDir(home)
	for _, l := range ls {
		ext := filepath.Ext(l.Name())
		if ext == ".yaml" {
			name := l.Name()[0 : len(l.Name())-len(ext)]
			remotes = append(remotes, name)
		}
	}
	return remotes
}

func completeEdit(args []string) {
	filter := ""
	if len(args) > 0 {
		filter = args[0]
	}

	filterEscapeAndPrint(filter, false, getRemotes()...)
}

func Complete(cl string) {
	args := strings.Split(cl, " ")

	if len(args) < 2 {
		filterEscapeAndPrint("", false, "pull ", "push ",
			"list ", "create ", "edit ", "mesh ", "sync ")
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
	case "edit":
		completeEdit(args[2:])
	default:
		filterEscapeAndPrint(args[1], false, "pull ", "push ",
			"list ", "create ", "edit ", "mesh ", "sync ")
	}
}
