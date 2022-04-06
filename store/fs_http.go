package store

import (
	"encoding/base64"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"io"
	"io/fs"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type HTTPConfig struct {
	Endpoint  string `json:"endpoint" yaml:"endpoint"`
	AccessKey string `json:"accessKey" yaml:"accessKey"`
	Secret    string `json:"secret" yaml:"secret"`
	SignKey   string `json:"signKey" yaml:"signKey"`
}

type HTTP struct {
	endpoint  string
	accessKey string
	signKey   []byte
	secret    string
	bearer    string
	exp       time.Time
}

func NewHTTP(config HTTPConfig) (FS, error) {
	signKey, err := base64.StdEncoding.DecodeString(config.SignKey)
	if err != nil {
		return nil, err
	}

	return &HTTP{
		endpoint:  config.Endpoint,
		accessKey: config.AccessKey,
		signKey:   signKey,
		secret:    config.Secret,
	}, nil
}

type CustomClaims struct {
	Id       string `json:"id"`
	Password string `json:"p"`
	jwt.StandardClaims
}

func (h *HTTP) getBearer() (string, error) {
	if time.Now().Before(h.exp) {
		return h.bearer, nil
	}

	exp := time.Now().Add(time.Hour)
	claims := &CustomClaims{
		Id:       h.accessKey,
		Password: h.secret,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: exp.Unix(),
			Issuer:    "noftp-client",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	bearer, err := token.SignedString(h.signKey)
	if err != nil {
		return "", err
	}
	h.bearer = bearer
	return bearer, nil
}

func (h *HTTP) Props() Props {
	return Props{
		Quota:       math.MaxInt64,
		Free:        math.MaxInt64,
		MinFileSize: 0,
		MaxFileSize: math.MaxInt64,
	}
}

func (h *HTTP) callServer(method string, url string, body io.Reader) (*http.Response, error) {
	bearer, err := h.getBearer()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearer))

	client := &http.Client{}
	return client.Do(req)
}

func (h *HTTP) MkdirAll(name string) error {
	url := fmt.Sprintf("%s/%s?dir", h.endpoint, name)
	resp, err := h.callServer(http.MethodPost, url, nil)
	defer resp.Body.Close()
	return err
}

func (h *HTTP) Pull(name string, w io.Writer) error {
	url := fmt.Sprintf("%s/%s", h.endpoint, name)
	resp, err := h.callServer(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(w, resp.Body)
	return err
}

func respToErr(r *http.Response) error {
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP response: %s", r.Status)
	} else {
		return nil
	}
}

func (h *HTTP) Push(name string, r io.Reader) error {
	url := fmt.Sprintf("%s/%s", h.endpoint, name)
	resp, err := h.callServer(http.MethodPut, url, r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return respToErr(resp)
}

func (h *HTTP) ReadDir(name string, opts ListOption) ([]fs.FileInfo, error) {
	var url string
	if opts == IncludeHiddenFiles {
		url = fmt.Sprintf("%s/%s?hidden", h.endpoint, name)
	} else {
		url = fmt.Sprintf("%s/%s", h.endpoint, name)
	}
	resp, err := h.callServer(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	err = respToErr(resp)
	if err != nil {
		return nil, err
	}

	if contentType := resp.Header.Get("content-type"); contentType != "text/plain" {
		return nil, os.ErrInvalid
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var ls []fs.FileInfo
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) == 4 {
			var dir bool
			switch parts[1] {
			case "d":
				dir = true
			case "f":
				dir = false
			default:
				continue
			}

			size, err := strconv.Atoi(parts[2])
			if err != nil {
				continue
			}

			tm, err := time.Parse(parts[3], time.RFC822)
			if err != nil {
				continue
			}

			ls = append(ls, simpleFileInfo{
				name:    parts[0],
				size:    int64(size),
				isDir:   dir,
				modTime: tm,
			})
		}
	}

	return ls, nil
}

func (h *HTTP) Watch(name string) chan string {
	return nil
}

func (h *HTTP) Stat(name string) (fs.FileInfo, error) {
	url := fmt.Sprintf("%s/%s", h.endpoint, name)
	resp, err := h.callServer(http.MethodHead, url, nil)
	if err != nil {
		return nil, err
	}

	size, err := strconv.Atoi(resp.Header.Get("Content-Size"))
	if err != nil {
		return nil, err
	}
	pragma := resp.Header.Get("Pragma")

	lastModified := resp.Header.Get("Last-Modified")
	tm, err := time.Parse(time.RFC822, lastModified)
	if err != nil {
		return nil, err
	}

	return simpleFileInfo{
		name:    path.Base(name),
		size:    int64(size),
		isDir:   pragma == "directory",
		modTime: tm,
	}, nil
}

func (h *HTTP) Remove(name string) error {
	url := fmt.Sprintf("%s/%s", h.endpoint, name)
	resp, err := h.callServer(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (h *HTTP) Touch(_ string) error {
	return ErrNotSupported
}

func (h *HTTP) Rename(old, new string) error {
	return ErrNotSupported
}

func (h *HTTP) Close() error {
	return nil
}
