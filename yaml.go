package notfs

import (
	"gopkg.in/yaml.v2"
)


func ReadYaml(f FS, name string, out interface{}) error {
	var s ByteStream
	err := f.CopyTo(name, &s)
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
	return f.CopyFrom(name, &ByteStream{d,0})
}
