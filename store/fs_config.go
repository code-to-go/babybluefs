package store

import (
	"bytes"
	"crypto/sha256"
	"math/rand"
	"os"
	"time"
)

//type MeshConfig3 struct {
//	FTP    map[string]FTPConfig
//	SMB   map[string]SMBConfig
//	S3     map[string]S3Config
//	Azure  map[string]AzureConfig
//	HTTP   map[string]HTTPConfig
//	Groups map[Group]string
//}

type Config struct {
	Name       string            `json:"name" yaml:"name"`
	Group      Group             `json:"group" yaml:"group"`
	FTP        *FTPConfig        `json:"ftp,omitempty" yaml:"ftp,omitempty"`
	SFTP       *SFTPConfig       `json:"sftp,omitempty" yaml:"sftp,omitempty"`
	S3         *S3Config         `json:"s3,omitempty" yaml:"s3,omitempty"`
	Azure      *AzureConfig      `json:"azure,omitempty" yaml:"azure,omitempty"`
	SMB        *SMBConfig        `json:"smb,omitempty" yaml:"smb,omitempty"`
	HTTP       *HTTPConfig       `json:"http,omitempty" yaml:"http,omitempty"`
	Sharepoint *SharepointConfig `json:"sharepoint,omitempty" yaml:"sharepoint,omitempty"`
	Kafka      *KafkaConfig      `json:"kafka,omitempty" yaml:"kafka,omitempty"`
}

const keyHashFile = ".keyHash"

func IsValidKeyHash(f FS, g Group, groups map[Group]string) bool {
	k, ok := groups[g]
	if !ok {
		return false
	}

	h := sha256.Sum256([]byte(k))
	buf := bytes.NewBuffer(nil)

	_, err := f.Stat(keyHashFile)
	if os.IsNotExist(err) {
		return f.Pull(keyHashFile, bytes.NewBuffer(h[:])) == nil
	}
	if err != nil {
		return false
	}

	err = f.Pull(keyHashFile, buf)
	return err == nil && bytes.Equal(buf.Bytes(), h[:])
}

// NewFS creates a new file storage broker with the given configuration c
func NewFS(c Config) (FS, error) {
	switch {
	case c.FTP != nil:
		return NewFTP(*c.FTP)
	case c.SFTP != nil:
		return NewSFTP(*c.SFTP)
	case c.S3 != nil:
		return NewS3(*c.S3)
	case c.Azure != nil:
		return NewAzure(*c.Azure)
	case c.SMB != nil:
		return NewSMB(*c.SMB)
	case c.HTTP != nil:
		return NewHTTP(*c.HTTP)
	case c.Sharepoint != nil:
		return NewSharepoint(*c.Sharepoint)
	case c.Kafka != nil:
		return NewKafka(*c.Kafka)
	}

	return nil, os.ErrInvalid
}

func GenerateRandomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	rand.Seed(time.Now().UnixMicro())
	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}
