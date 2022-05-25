package store

import (
	"bytes"
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"io/fs"
	"math"
	"path"
	"strings"
	"time"
)

type S3Config struct {
	Endpoint  string `json:"endpoint" yaml:"endpoint"`
	Bucket    string `json:"bucket" yaml:"bucket"`
	Location  string `json:"location" yaml:"location"`
	AccessKey string `json:"accessKey" yaml:"accessKey"`
	Secret    string `json:"secret" yaml:"secret"`
	UseSSL    bool   `json:"useSsl" yaml:"useSsl"`
}

type S3FS struct {
	c      *minio.Client
	bucket string
	url    string
}

func checkBucket(c *minio.Client, ctx context.Context, bucket string, location string) error {
	exists, err := c.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	} else {
		return c.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: location})
	}
}

func NewS3(config S3Config) (FS, error) {
	ctx := context.Background()
	defer ctx.Done()

	c, err := minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.Secret, ""),
		Secure: config.UseSSL,
	})

	err = checkBucket(c, ctx, config.Bucket, config.Location)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("s3://%s@%s/%s#loc-%s", config.AccessKey, config.Endpoint, config.Bucket, config.Location)
	return &S3FS{c, config.Bucket, url}, nil
}

func (s3 *S3FS) Props() Props {
	return Props{
		Quota:       math.MaxInt64,
		Free:        math.MaxInt64,
		MinFileSize: 0,
		MaxFileSize: math.MaxInt64,
	}
}

func (s3 *S3FS) MkdirAll(name string) error {
	if !strings.HasSuffix(name, "/") {
		name = name + "/"
	}
	return s3.Push(name, bytes.NewReader(nil))
}

func (s3 *S3FS) Pull(name string, w io.Writer) error {
	ctx := context.Background()
	defer ctx.Done()

	r, err := s3.c.GetObject(ctx, s3.bucket, name, minio.GetObjectOptions{})
	if err != nil {
		return err
	}
	defer r.Close()

	_, err = io.Copy(w, r)
	return err
}

func (s3 *S3FS) Push(name string, r io.Reader) error {
	ctx := context.Background()
	defer ctx.Done()

	if strings.HasSuffix(name, "/") {
		return fmt.Errorf("file can not have / suffix")
	}

	_, err := s3.c.PutObject(ctx, s3.bucket, name, r, -1, minio.PutObjectOptions{})
	return err
}

func (s3 *S3FS) ReadDir(name string, opts ListOption) ([]fs.FileInfo, error) {
	ctx := context.Background()
	defer ctx.Done()

	if name != "" && !strings.HasSuffix(name, "/") {
		name += "/"
	}

	ls := s3.c.ListObjects(ctx, s3.bucket, minio.ListObjectsOptions{
		Prefix:    name,
		Recursive: false,
	})

	var sfo []fs.FileInfo
	for e := range ls {
		n := e.Key[len(name):]
		if n != "" && (opts&IncludeHiddenFiles == 1 || !strings.HasPrefix(n, ".")) {
			isDir := strings.HasSuffix(e.Key, "/")
			if isDir {
				n = n[0 : len(n)-1]
			}
			sfo = append(sfo, simpleFileInfo{
				name:    n,
				size:    e.Size,
				isDir:   isDir,
				modTime: e.LastModified,
			})
		}

	}
	return sfo, nil
}

func (s3 *S3FS) Watch(name string) chan string {
	return nil
}

func (s3 *S3FS) Stat(name string) (fs.FileInfo, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, err := s3.c.StatObject(ctx, s3.bucket, name, minio.StatObjectOptions{})
	if err != nil {
		if ls, err := s3.ReadDir(name, IncludeHiddenFiles); err == nil && len(ls) > 0 {
			var tm time.Time
			for _, l := range ls {
				if tm.Before(l.ModTime()) {
					tm = l.ModTime()
				}
			}
			return simpleFileInfo{
				name:    name,
				size:    0,
				isDir:   true,
				modTime: tm,
			}, nil
		}

		return nil, err
	}

	return simpleFileInfo{
		name:    path.Base(r.Key),
		size:    r.Size,
		isDir:   strings.HasSuffix(r.Key, "/"),
		modTime: r.LastModified,
	}, nil
}

func (s3 *S3FS) Remove(name string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	return s3.c.RemoveObject(ctx, s3.bucket, name, minio.RemoveObjectOptions{})
}

func (s3 *S3FS) Touch(_ string) error {
	return ErrNotSupported
}

func (s3 *S3FS) Rename(old, new string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := s3.c.CopyObject(ctx, minio.CopyDestOptions{Object: new},
		minio.CopySrcOptions{Object: old})
	if err != nil {
		return err
	}
	return s3.Remove(old)
}

func (s3 *S3FS) Close() error {
	return nil
}

func (s3 *S3FS) String() string {
	return s3.url
}
