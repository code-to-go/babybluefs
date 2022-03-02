package main

import (
	"gopkg.in/yaml.v2"
)


func ReadYaml(f FS, name string, out interface{}) error {
	var s ByteStream
	err := f.Pull(name, &s)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(s.Data, out)
}

func WriteYaml(f FS, name string, in interface{}) error {
	d, err := yaml.Marshal(in)
	if err != nil {
		return err
	}
	return f.Pull(name, &ByteStream{d,0})
}
