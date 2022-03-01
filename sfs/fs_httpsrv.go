package sfs

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"time"
)

type HTTPServer struct {
	Get func(w http.ResponseWriter, r *http.Request)
	Put func(w http.ResponseWriter, r *http.Request)
	Head func(w http.ResponseWriter, r *http.Request)
	Delete func(w http.ResponseWriter, r *http.Request)
}

func getFileInfo(f FS, ph string, w http.ResponseWriter) (fs.FileInfo, error) {
	l, err := f.Stat(ph)
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	return l, err
}

func NewHttpServer(f FS, prefix string) (HTTPServer, error) {
	return HTTPServer{
		Get: func(w http.ResponseWriter, r *http.Request) {
			ph := r.URL.Path[len(prefix):]
			l, err := getFileInfo(f, ph, w)
			if err != nil {
				return
			}

			w.WriteHeader(http.StatusOK)
			if l.IsDir() {
				var ls []fs.FileInfo
				if r.URL.RawQuery == "hidden" {
					ls, _ = f.ReadDir(ph, IncludeHiddenFiles)
				} else {
					ls, _ = f.ReadDir(ph, 0)
				}
				for _, l = range ls {
					w.Header().Add("Last-Modified", l.ModTime().Format(time.RFC822))
					if l.IsDir() {
						w.Header().Add("Content-Type", "text/plain")
						w.Header().Add("Pragma", "directory")
						line := fmt.Sprintf("%s\td\t0\t%s\n", l.Name(), l.ModTime().Format(time.RFC822))
						w.Write([]byte(line))
					} else {
						w.Header().Add("Content-Type", Mime(f, ph).String())
						line := fmt.Sprintf("%s\tf\t%d\t%s\n", l.Name(), l.Size(), l.ModTime().Format(time.RFC822))
						w.Write([]byte(line))
					}
				}
			} else {
				f.Pull(ph, w)
			}
		},
		Head: func(w http.ResponseWriter, r *http.Request) {
			ph := r.URL.Path[len(prefix):]
			l, err := getFileInfo(f, ph, w)
			if err != nil {
				return
			}
			w.WriteHeader(http.StatusOK)
			if l.IsDir() {
				w.Header().Add("Content-Type", "text/plain")
				w.Header().Add("Pragma", "directory")
			} else {
				w.Header().Add("Content-Type", Mime(f, ph).String())
			}
			w.Header().Add("Last-Modified", l.ModTime().Format(time.RFC822))
		},
		Put: func(w http.ResponseWriter, r *http.Request) {
			ph := r.URL.Path[len(prefix):]
			var err error
			if r.URL.RawQuery == "dir"  {
				err = f.MkdirAll(ph)
			} else {
				err = f.Push(ph, r.Body)
			}

			if err == nil {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
			}
		},
		Delete: func(w http.ResponseWriter, r *http.Request) {
			ph := r.URL.Path[len(prefix):]
			err := f.Remove(ph)
			if err == nil {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
			}
		},
	}, nil
}
