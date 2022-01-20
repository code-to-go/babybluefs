package notfs

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"io/fs"
	"path"
	"strings"
)

type S3Config struct {
	Endpoint  string `json:"endpoint" yaml:"endpoint"`
	Bucket    string `json:"bucket" yaml:"bucket"`
	Location  string `json:"location" yaml:"location"`
	AccessKey string `json:"accessKey" yaml:"access_key"`
	Secret    string `json:"signKey" yaml:"signKey"`
	UseSSL bool  `json:"useSsl" yaml:"useSsl"`
	Group  Group `json:"group" yaml:"group"`
}

type S3FS struct {
	c      *minio.Client
	bucket string
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

	return &S3FS{c, config.Bucket}, nil
}

func (s3 *S3FS) Props() Props {
	return Props{
		Quota: -1,
		Free: -1,
		MaxFileSize: -1,
	}
}


func (s3 *S3FS) MkdirAll(name string) error {
	panic("implement me")
}

func (s3 *S3FS) CopyTo(name string, w io.Writer) error {
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

func (s3 *S3FS) CopyFrom(name string, r io.Reader) error {
	ctx := context.Background()
	defer ctx.Done()

	_, err := s3.c.PutObject(ctx, s3.bucket, name, r, -1, minio.PutObjectOptions{})
	return err
}

func (s3 *S3FS) ReadDir(name string, opts ListOption) ([]fs.FileInfo, error) {
	ctx := context.Background()
	defer ctx.Done()

	//prefix := path.Join(s3.bucket, name)
	if name != "" {
		name += "/"
	}

	ls := s3.c.ListObjects(ctx, s3.bucket, minio.ListObjectsOptions{
		Prefix:    name,
		Recursive: false,
	})

	var sfo []fs.FileInfo
	for e := range ls {
		if strings.HasSuffix(e.Key, "/") {
			continue
		}

		n := e.Key[len(name):]
		if opts&IncludeHiddenFiles == 0 && !strings.HasPrefix(n, ".") {
			sfo = append(sfo, SimpleFileInfo{
				name:    n,
				size:    e.Size,
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
		return nil, err
	}

	return SimpleFileInfo{
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
