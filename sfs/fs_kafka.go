package sfs

import (
	"bytes"
	"context"
	"github.com/patrickmn/go-cache"
	"github.com/segmentio/kafka-go"
	"io"
	"io/fs"
	"math"
	"os"
	"time"
)

type KafkaConfig struct {
	Addr      string `json:"addr" yaml:"addr"`
	Topic     string `json:"topic" yaml:"topic"`
	Partition int    `json:"partition" yaml:"partition"`
	Staging string `json:"staging"`
	Group   Group  `json:"group" yaml:"group"`
}

type kafkaItem struct {
	l    SimpleFileInfo
	data []byte
}

type KafkaFS struct {
	w  *kafka.Writer
	r  *kafka.Reader
	ch *cache.Cache
}

func NewKafka(config KafkaConfig) (FS, error) {
	w := &kafka.Writer{
		Addr:     kafka.TCP(config.Addr),
		Topic:    config.Topic,
		Balancer: &kafka.LeastBytes{},
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{config.Addr},
		Topic:     config.Topic,
		Partition: 0,
		MinBytes:  10e3, // 10KB
		MaxBytes:  10e6, // 10MB
	})

	ch := cache.New(10*time.Minute, time.Hour)
	return &KafkaFS{w, r, ch}, nil
}

func (ka *KafkaFS) Props() Props {
	return Props{
		Quota:       math.MaxInt64,
		Free:        math.MaxInt64,
		MinFileSize: 0,
		MaxFileSize: 16 * 1000 * 1000,
	}
}

func (ka *KafkaFS) MkdirAll(name string) error {
	return nil
}

func (ka *KafkaFS) Pull(name string, w io.Writer) error {
	ctx := context.Background()
	defer ctx.Done()

	v, ok := ka.ch.Get(name)
	if !ok {
		return os.ErrNotExist
	}

	_, err := io.Copy(w, bytes.NewBuffer(v.(kafkaItem).data))
	return err
}

func (ka *KafkaFS) Push(name string, r io.Reader) error {
	ctx := context.Background()
	defer ctx.Done()

	buf := bytes.NewBuffer(nil)
	io.Copy(buf, r)

	return ka.w.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(name),
			Value: buf.Bytes(),
		})
}

func (ka *KafkaFS) ReadDir(name string, opts ListOption) ([]fs.FileInfo, error) {
	ctx := context.Background()
	defer ctx.Done()

	var ls []fs.FileInfo
	ka.r.SetOffset(0)
	for {
		m, err := ka.r.ReadMessage(ctx)
		if err != nil {
			break
		}

		name := string(m.Key)
		l := SimpleFileInfo{
			name:    name,
			size:    int64(len(m.Value)),
			isDir:   false,
			modTime: m.Time,
		}

		ls = append(ls, l)
		ka.ch.Set(string(m.Key), kafkaItem{
			l:    l,
			data: m.Value,
		}, time.Hour)
	}

	return ls, nil
}

func (ka *KafkaFS) Watch(name string) chan string {
	return nil
}

func (ka *KafkaFS) Stat(name string) (fs.FileInfo, error) {
	v, ok := ka.ch.Get(name)
	if !ok {
		return nil, os.ErrNotExist
	}

	return v.(kafkaItem).l, nil
}

func (ka *KafkaFS) Remove(name string) error {
	return nil
}

func (ka *KafkaFS) Touch(name string) error {
	return ErrNotSupported
}

func (ka *KafkaFS) Rename(old, new string) error {
	return nil
}

func (ka *KafkaFS) Close() error {
	return nil
}
