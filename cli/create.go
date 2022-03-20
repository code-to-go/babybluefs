package cli

import (
	"fmt"
	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"stratofs/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)


func getUnixEditor(editor ...string) string {
	for _, e := range editor {
		for _, f := range []string{"/usr/bin", "/bin"} {
			n := filepath.Join(f, e)
			if _, err := os.Stat(n); err == nil {
				return n
			}
		}
	}
	return "vi"
}

func getEditor() string {
	switch runtime.GOOS {
	case "windows":
		return "notepad"
	case "linux", "mac":
		return getUnixEditor("micro", "nano", "code", "vim", "vi")
	}
	color.Red("cannot find editor")
	os.Exit(1)
	return ""
}

func Create(transport string) {
	c := fs.Config{
		Name:  "",
		Group: "public",
	}

	switch transport {
	case "s3":
		c.S3 = &fs.S3Config{}
	case "sftp":
		c.SFTP = &fs.SFTPConfig{}
	case "azure":
		c.Azure = &fs.AzureConfig{}
	}

	home := os.Getenv("SF_HOME")
	if home == "" {
		home, _ = os.Getwd()
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		color.Red("cannot generate %s file: %v", transport, err)
		os.Exit(1)
	}

	name := fmt.Sprintf("%s-%d.yaml", transport, time.Now().Unix())
	err = ioutil.WriteFile(filepath.Join(home, name), data, 0644)
	if err != nil {
		color.Red("cannot write %s: %v", name, err)
		os.Exit(1)
	}

	cmd := exec.Command(getEditor(), filepath.Join(home, name))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		color.Red("Something went wrong: %v", err)
		os.Exit(1)
	}

	data, err = os.ReadFile(filepath.Join(home, name))
	if err != nil {
		color.Red("Something went wrong: %v", err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(data, &c)
	if err != nil {
		color.Red("Something went wrong: %v", err)
		os.Exit(1)
	}

	c.Name = strings.TrimSpace(c.Name)
	if c.Name != "" {
		err = os.Rename(filepath.Join(home, name), filepath.Join(home, fmt.Sprintf("%s.yaml", c.Name)))
		if err != nil {
			color.Red("Cannot rename %s to %s: %v", name, c.Name, err)
			os.Exit(1)
		}
	}
}
