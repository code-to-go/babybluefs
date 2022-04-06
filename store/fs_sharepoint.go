package store

import (
	"fmt"
	"github.com/koltyakov/gosip"
	"github.com/koltyakov/gosip/api"
	"github.com/koltyakov/gosip/auth/addin"
	"github.com/koltyakov/gosip/auth/saml"
	"io"
	"io/fs"
	"math"
)

type SharepointConfig struct {
	Site      string `json:"site" yaml:"site"`
	AuthAsApp *struct {
		ClientId     string `json:"clientId" yaml:"clientId"`
		ClientSecret string `json:"clientSecret" yaml:"clientSecret"`
	} `json:"authAsApp" yaml:"authAsApp"`
	AuthAsSAML *struct {
		Username string `json:"username" yaml:"username"`
		Password string `json:"password" yaml:"password"`
	} `json:"authAsSAML" yaml:"authAsSAML"`
}

type SharepointFS struct {
	sp *api.SP
}

func NewSharepoint(config SharepointConfig) (FS, error) {
	authCnfg := getSpAuth(config)
	if authCnfg == nil {
		return nil, ErrNotSupported
	}

	client := &gosip.SPClient{AuthCnfg: authCnfg}
	sp := api.NewSP(client)

	res, err := sp.Web().Select("Title").Get()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	fmt.Printf("%s\n", res.Data().Title)

	return &SharepointFS{sp}, nil
}

func getSpAuth(c SharepointConfig) gosip.AuthCnfg {
	switch {
	case c.AuthAsApp != nil:
		return &addin.AuthCnfg{
			SiteURL:      c.Site,
			ClientID:     c.AuthAsApp.ClientId,
			ClientSecret: c.AuthAsApp.ClientSecret,
		}
	case c.AuthAsSAML != nil:
		return &saml.AuthCnfg{
			SiteURL:  c.Site,
			Username: c.AuthAsSAML.Username,
			Password: c.AuthAsSAML.Password,
		}
	}
	return nil
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
