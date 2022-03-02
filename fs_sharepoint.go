package main

import (
	"fmt"
	"github.com/koltyakov/gosip"
	"github.com/koltyakov/gosip/api"
	strategy "github.com/koltyakov/gosip/auth/addin"
	"io"
	"io/fs"
	"math"
)

type SharepointConfig struct {
	Site         string `json:"site" yaml:"site"`
	ClientId     string `json:"clientId" yaml:"clientId"`
	ClientSecret string `json:"clientSecret" yaml:"clientSecret"`
	Group        Group  `json:"group" yaml:"group"`
}

type SharepointFS struct {
	sp *api.SP
}

func NewSharepoint(config SharepointConfig) (FS, error) {
	auth := &strategy.AuthCnfg{
		SiteURL:      config.Site,
		ClientID:     config.ClientId,
		ClientSecret: config.ClientSecret,
	}
	client := &gosip.SPClient{AuthCnfg: auth}
	sp := api.NewSP(client)

	res, err := sp.Web().Select("Title").Get()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("%s\n", res.Data().Title)

	return &SharepointFS{sp}, nil
}


func (sp *SharepointFS) Props() Props {
	return Props{
		Quota:       math.MaxInt64,
		Free:        math.MaxInt64,
		MinFileSize: 0,
		MaxFileSize: math.MaxInt64,
	}
}

func (sp *SharepointFS) MkdirAll(name string) error {
	return nil
}

func (sp *SharepointFS) Pull(name string, w io.Writer) error {
	return nil
}

func (sp *SharepointFS) Push(name string, r io.Reader) error {
	return nil
}

func (sp *SharepointFS) ReadDir(name string, opts ListOption) ([]fs.FileInfo, error) {
	return nil, nil
}

func (sp *SharepointFS) Watch(name string) chan string {
	return nil
}

func (sp *SharepointFS) Stat(name string) (fs.FileInfo, error) {
	return nil, nil
}

func (sp *SharepointFS) Remove(name string) error {
	return nil
}

func (sp *SharepointFS) Touch(name string) error {
	return ErrNotSupported
}

func (sp *SharepointFS) Rename(old, new string) error {
	return nil
}

func (sp *SharepointFS) Close() error {
	return nil
}
