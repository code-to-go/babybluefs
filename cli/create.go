package cli

import (
	"babybluefs/store"
	"fmt"
	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
	"io/ioutil"
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
	c := store.Config{
		Name:  "",
		Group: "public",
	}

	switch transport {
	case "s3":
		c.S3 = &store.S3Config{}
	case "sftp":
		c.SFTP = &store.SFTPConfig{}
	case "ftp":
		c.FTP = &store.FTPConfig{}
	case "azure":
		c.Azure = &store.AzureConfig{}
	case "smb":
		c.SMB = &store.SMBConfig{}
	case "kafka":
		c.Kafka = &store.KafkaConfig{}
	case "sharepoint":
		c.Sharepoint = &store.SharepointConfig{}
	}

	home := GetHome()

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

	before, _ := os.Stat(filepath.Join(home, name))
	cmd := exec.Command(getEditor(), filepath.Join(home, name))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		color.Red("Something went wrong: %v", err)
		os.Exit(1)
	}

	after, _ := os.Stat(filepath.Join(home, name))
	if before.ModTime() == after.ModTime() {
		color.Green("No changes. Delete file")
		_ = os.Remove(filepath.Join(home, name))
		return
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

	color.Green("configuration %s created", c.Name)
}

func Edit(remote string) {
	home := GetHome()
	ph := filepath.Join(home, fmt.Sprintf("%s.yaml", remote))
	before, err := os.Stat(ph)
	if err != nil {
		color.Red("%s does not exist", remote)
		os.Exit(1)
	}

	cmd := exec.Command(getEditor(), ph)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		color.Red("Something went wrong: %v", err)
		os.Exit(1)
	}
	after, _ := os.Stat(ph)
	if before.ModTime() == after.ModTime() {
		color.Green("No changes done")
	} else {
		color.Green("Changes done")
	}
}
