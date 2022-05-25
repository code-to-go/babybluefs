package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/patrickmn/go-cache"
	"github.com/segmentio/kafka-go"
	"io"
	"io/fs"
	"math"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const kafkaCacheExpiration = time.Minute * 10

type KafkaConfig struct {
	Brokers           []string      `json:"brokers" yaml:"brokers"`
	NumPartitions     int           `json:"numPartitions" yaml:"numPartitions"`
	ReplicationFactor int           `yaml:"replicationFactor" yaml:"replicationFactor"`
	Staging           string        `json:"staging"`
	GroupId           string        `json:"groupId" yaml:"groupId"`
	MaxLs             int           `json:"maxLs" yaml:"maxLs"`
	ReadTimeout       time.Duration `json:"readTimeout" yaml:"readTimeout"`
}

type kafkaTopic struct {
	w        *kafka.Writer
	r        *kafka.Reader
	messages map[string]kafka.Message
}

type KafkaFS struct {
	config KafkaConfig
	c      *kafka.Conn
	ch     *cache.Cache
}

func NewKafka(config KafkaConfig) (FS, error) {
	c, err := kafka.Dial("tcp", config.Brokers[0])
	if err != nil {
		return nil, err
	}

	f := &KafkaFS{config, c, cache.New(kafkaCacheExpiration, kafkaCacheExpiration)}
	if f.config.MaxLs == 0 {
		f.config.MaxLs = 256
	}
	if f.config.ReadTimeout == 0 {
		f.config.ReadTimeout = time.Millisecond * 200
	}
	return f, nil
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
	if strings.Contains(name, "/") {
		return ErrNotSupported
	}

	topicConfig := kafka.TopicConfig{
		Topic:             name,
		NumPartitions:     ka.config.NumPartitions,
		ReplicationFactor: ka.config.ReplicationFactor,
	}

	return ka.c.CreateTopics(topicConfig)
}

func (ka *KafkaFS) Pull(name string, w io.Writer) error {
	topicName, key := path.Split(name)
	v, err := ka.getKafkaTopic(topicName)
	if err != nil {
		return err
	}

	switch key {
	case "@offset":
		_, err = w.Write([]byte(strconv.FormatInt(v.r.Offset(), 10)))
		return err
	case "@lag":
		_, err = w.Write([]byte(strconv.FormatInt(v.r.Lag(), 10)))
		return err
	case "@stat":
		data, err := json.Marshal(v.r.Stats())
		if err == nil {
			_, err = w.Write(data)
		}
		return err
	}

	_ = ka.fetchKafkaMessages(v)
	m, ok := v.messages[key]
	if ok {
		var err error
		if w != nil {
			_, err = io.Copy(w, bytes.NewBuffer(m.Value))
		}
		if err == nil {
			ctx := context.Background()
			defer ctx.Done()
			_ = v.r.CommitMessages(ctx, m)
			delete(v.messages, key)
		}
		return err
	} else {
		return fs.ErrNotExist
	}
}

func (ka *KafkaFS) Push(name string, r io.Reader) error {
	topicName, key := path.Split(name)
	if topicName == "" {
		topicName = key
		key = ""
	}
	v, err := ka.getKafkaTopic(topicName)
	if err != nil {
		return err
	}

	ctx := context.Background()
	defer ctx.Done()

	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, r)
	if err != nil {
		return err
	}

	if key == "@offset" {
		o, err := strconv.ParseInt(buf.String(), 10, 64)
		if err != nil {
			return err
		}
		return v.r.SetOffset(o)
	}
	if key == "" {
		return v.w.WriteMessages(ctx,
			kafka.Message{
				Value: buf.Bytes(),
			})
	} else {
		return v.w.WriteMessages(ctx,
			kafka.Message{
				Key:   []byte(key),
				Value: buf.Bytes(),
			})
	}
}

func (ka *KafkaFS) getKafkaTopics() ([]string, error) {
	partitions, err := ka.c.ReadPartitions()
	if _, ok := err.(*net.OpError); ok {
		ka.c, err = kafka.Dial("tcp", ka.config.Brokers[0])
		if err != nil {
			return nil, err
		}
	}

	if err != nil {
		return nil, err
	}

	m := map[string]struct{}{}
	for _, p := range partitions {
		m[p.Topic] = struct{}{}
	}

	var topics []string
	for n := range m {
		topics = append(topics, n)
	}
	return topics, nil
}

func (ka *KafkaFS) getKafkaTopic(name string) (kafkaTopic, error) {
	name = path.Clean(name)
	topic, ok := ka.ch.Get(name)
	if !ok {
		topics, err := ka.getKafkaTopics()
		if err != nil {
			return kafkaTopic{}, err
		}

		for _, t := range topics {
			if t == name {
				topic = kafkaTopic{
					w: &kafka.Writer{
						Addr:     kafka.TCP(ka.config.Brokers...),
						Topic:    name,
						Balancer: &kafka.LeastBytes{},
					},
					r: kafka.NewReader(kafka.ReaderConfig{
						Brokers: ka.config.Brokers,
						GroupID: ka.config.GroupId,
						Topic:   name,
					}),
					messages: make(map[string]kafka.Message),
				}
				_ = ka.ch.Add(name, topic, kafkaCacheExpiration)
				return topic.(kafkaTopic), nil
			}
		}
		return kafkaTopic{}, fs.ErrNotExist
	}

	return topic.(kafkaTopic), nil
}

func (ka *KafkaFS) fetchKafkaMessages(topic kafkaTopic) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*80)
	defer cancel()

	for len(topic.messages) < ka.config.MaxLs {
		m, err := topic.r.FetchMessage(ctx)
		if errors.Is(err, context.DeadlineExceeded) {
			break
		}
		if err != nil {
			return err
		}
		if m.Key == nil {
			topic.messages[strconv.FormatInt(m.Offset, 10)] = m
		} else {
			topic.messages[string(m.Key)] = m
		}
		println(string(m.Key), m.Offset)
	}
	return nil
}

func (ka *KafkaFS) ReadDir(name string, opts ListOption) ([]fs.FileInfo, error) {
	ctx := context.Background()
	defer ctx.Done()

	var ls []fs.FileInfo
	if name == "" || name == "/" {
		topics, err := ka.getKafkaTopics()
		if err != nil {
			return nil, err
		}
		for _, topic := range topics {
			if opts == IncludeHiddenFiles || !strings.HasPrefix(topic, ".") {
				ls = append(ls, simpleFileInfo{
					name:    topic,
					size:    0,
					isDir:   true,
					modTime: time.Time{},
				})
			}
		}
		return ls, nil
	}

	name = path.Clean(name)
	if strings.Contains(name, "/") {
		return nil, fs.ErrNotExist
	}

	topic, err := ka.getKafkaTopic(name)
	if err != nil {
		return nil, err
	}
	err = ka.fetchKafkaMessages(topic)
	if err != nil {
		return nil, err
	}

	for n, m := range topic.messages {
		if opts == IncludeHiddenFiles || !strings.HasPrefix(n, ".") {
			ls = append(ls, simpleFileInfo{
				name:    n,
				size:    int64(len(m.Value)),
				isDir:   false,
				modTime: m.Time,
			})
		}
	}

	ls = append(ls,
		simpleFileInfo{name: "@stat"},
		simpleFileInfo{name: "@offset"},
		simpleFileInfo{name: "@lag"})

	return ls, nil
}

func (ka *KafkaFS) Watch(string) chan string {
	return nil
}

func (ka *KafkaFS) Stat(name string) (fs.FileInfo, error) {
	topicName, key := path.Split(name)
	v, ok := ka.ch.Get(topicName)
	if !ok {
		return nil, os.ErrNotExist
	}

	if key == "" {
		return simpleFileInfo{
			name:    topicName,
			size:    0,
			isDir:   true,
			modTime: time.Time{},
		}, nil
	}

	_ = ka.fetchKafkaMessages(v.(kafkaTopic))
	m, ok := v.(kafkaTopic).messages[key]
	if ok {
		return simpleFileInfo{
			name:    key,
			size:    int64(len(m.Value)),
			isDir:   false,
			modTime: m.Time,
		}, nil
	} else {
		return nil, fs.ErrNotExist
	}
}

func (ka *KafkaFS) Remove(name string) error {
	return ka.Pull(name, nil)
}

func (ka *KafkaFS) Touch(string) error {
	return ErrNotSupported
}

func (ka *KafkaFS) Rename(old, new string) error {
	_, oldKey := path.Split(old)
	_, newKey := path.Split(new)
	if oldKey == "" || newKey == "" {
		return ErrNotSupported
	}

	buf := bytes.NewBuffer(nil)
	err := ka.Pull(old, buf)
	if err != nil {
		return err
	}
	return ka.Push(new, buf)
}

func (ka *KafkaFS) Close() error {
	return nil
}

func (ka *KafkaFS) String() string {
	return "kafka://" + strings.Join(ka.config.Brokers, ",")
}
