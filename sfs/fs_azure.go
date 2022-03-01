package sfs

import (
	"bytes"
	"context"
	"fmt"
	"github.com/Azure/azure-pipeline-go/pipeline"
	"github.com/Azure/azure-storage-file-go/azfile"
	"io"
	"io/fs"
	"math"
	"net/url"
	"path"
	"strings"
)

type AzureConfig struct {
	Addr        string `json:"addr" yaml:"addr"`
	AccountName string `json:"accountName" yaml:"accountName"`
	AccountKey  string `json:"accountKey" yaml:"accountKey"`
	Share string `json:"share" yaml:"share"`
	Group Group  `json:"group" yaml:"group"`
}

type AzureFS struct {
	p    pipeline.Pipeline
	root string
}

func NewAzure(config AzureConfig) (FS, error) {
	credential, err := azfile.NewSharedKeyCredential(config.AccountName, config.AccountKey)
	if err != nil {
		return nil, err
	}
	root := fmt.Sprintf("https://%s/%s", config.Addr, config.Share)
	if _, err = url.Parse(root); err != nil {
		return nil, err
	}
	p := azfile.NewPipeline(credential, azfile.PipelineOptions{})

	return &AzureFS{p, root}, nil
}

func (az *AzureFS) Props() Props {
	return Props{
		Quota:       math.MaxInt64,
		Free:        math.MaxInt64,
		MinFileSize: 0,
		MaxFileSize: math.MaxInt64,
	}
}

func (az *AzureFS) MkdirAll(name string) error {
	if name == "" {
		return nil
	}
	ctx := context.Background()
	directoryUrl, err := az.getDirectoryUrl(name)
	if err != nil {
		return err
	}
	_, err = directoryUrl.GetProperties(ctx)
	if err == nil {
		return nil
	}

	d := ""
	for _, p := range strings.Split(name, "/") {
		directoryUrl, _ = az.getDirectoryUrl(d)
		directoryUrl = directoryUrl.NewDirectoryURL(p)
		_, err = directoryUrl.Create(ctx, azfile.Metadata{}, azfile.SMBProperties{})
		if err != nil {
			return err
		}
		d = path.Join(d, p)
	}
	return nil
}

func (az *AzureFS) getFileUrl(name string) (azfile.FileURL, error) {
	u, err := url.Parse(fmt.Sprintf("%s/%s", az.root, name))
	if err != nil {
		return azfile.FileURL{}, err
	}
	return azfile.NewFileURL(*u, az.p), nil
}

func (az *AzureFS) getDirectoryUrl(name string) (azfile.DirectoryURL, error) {
	u, err := url.Parse(fmt.Sprintf("%s/%s", az.root, name))
	if err != nil {
		return azfile.DirectoryURL{}, err
	}
	return azfile.NewDirectoryURL(*u, az.p), nil
}

func (az *AzureFS) Pull(name string, w io.Writer) error {
	ctx := context.Background()
	defer ctx.Done()

	fileURL, err := az.getFileUrl(name)
	if err != nil {
		return err
	}

	resp, err := fileURL.Download(ctx, 0, azfile.CountToEnd, false)
	if err != nil {
		return err
	}
	r := resp.Body(azfile.RetryReaderOptions{MaxRetryRequests: 3})
	defer r.Close()

	_, err = io.Copy(w, r)
	return err
}

func (az *AzureFS) Push(name string, r io.Reader) error {
	ctx := context.Background()
	defer ctx.Done()

	_ = az.MkdirAll(path.Dir(name))

	fileURL, err := az.getFileUrl(name)
	if err != nil {
		return err
	}
	_, err = fileURL.Create(ctx, azfile.FileMaxSizeInBytes, azfile.FileHTTPHeaders{}, azfile.Metadata{})
	if err != nil {
		return err
	}

	var offset int64
	var n int
	buf := make([]byte, 16000)
	for err != io.EOF {
		n, err = r.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n > 0 {
			body := bytes.NewReader(buf[0:n])
			_, err = fileURL.UploadRange(ctx, offset, body, nil)
			if err != nil {
				_, _ = fileURL.Resize(ctx, 0)
				return err
			}
			offset += int64(n)
		}
	}

	_, err = fileURL.Resize(ctx, offset)
	return err
}

func (az *AzureFS) ReadDir(name string, opts ListOption) ([]fs.FileInfo, error) {
	ctx := context.Background()
	defer ctx.Done()

	directoryURL, err := az.getDirectoryUrl(name)
	if err != nil {
		return nil, err
	}

	ls, err := directoryURL.ListFilesAndDirectoriesSegment(ctx, azfile.Marker{},
		azfile.ListFilesAndDirectoriesOptions{})
	if err != nil {
		return nil, err
	}

	var sfo []fs.FileInfo
	for _, e := range ls.FileItems {
		n := e.Name
		if opts&IncludeHiddenFiles == 0 && !strings.HasPrefix(n, ".") {
			fileUrl, err := az.getFileUrl(path.Join(name, n))
			if err == nil {
				properties, err := fileUrl.GetProperties(ctx)
				if err == nil {
					sfo = append(sfo, SimpleFileInfo{
						name:    n,
						size:    properties.ContentLength(),
						modTime: properties.LastModified(),
					})
				}
			}
		}

	}
	return sfo, nil
}

func (az *AzureFS) Watch(name string) chan string {
	return nil
}

func (az *AzureFS) Stat(name string) (fs.FileInfo, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fileUrl, err := az.getFileUrl(name)
	if err != nil {
		return nil, err
	}

	properties, err := fileUrl.GetProperties(ctx)
	if err != nil {
		return nil, err
	}

	return SimpleFileInfo{
		name:    path.Base(name),
		size:    properties.ContentLength(),
		isDir:   false,
		modTime: properties.LastModified(),
	}, nil
}

func (az *AzureFS) Remove(name string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fileUrl, err := az.getFileUrl(name)
	if err != nil {
		return err
	}
	_, err = fileUrl.Delete(ctx)
	return err
}

func (az *AzureFS) Touch(name string) error {
	return ErrNotSupported
}

func (az *AzureFS) Rename(old, new string) error {
	oldUrl, err := az.getFileUrl(old)
	if err != nil {
		return err
	}
	newUrl, err := az.getFileUrl(new)
	if err != nil {
		return err
	}

	ctx := context.Background()

	_, err = newUrl.Create(ctx, azfile.FileMaxSizeInBytes, azfile.FileHTTPHeaders{}, azfile.Metadata{})
	if err != nil {
		return err
	}

	_, err = newUrl.StartCopy(ctx, oldUrl.URL(), azfile.Metadata{})
	if err != nil {
		return err
	}
	_, _ = oldUrl.Delete(ctx)
	return err
}

func (az *AzureFS) Close() error {
	return nil
}
