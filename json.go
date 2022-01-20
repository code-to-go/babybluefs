package notfs

import (
	"encoding/json"
)

//ReadJSON reads a JSON file
func ReadJSON(f FS, name string, out interface{}) error {
	var s ByteStream
	err := f.CopyTo(name, &s)
	if err != nil {
		return err
	}

	return json.Unmarshal(s.Data, out)
}

func WriteJSON(f FS, name string, in interface{}) error {
	d, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return f.CopyFrom(name, &ByteStream{d,0})
}

func ConvertWithJSON(in interface{}, out interface{}) error {
	if j, err := json.Marshal(in); err != nil {
		return err
	} else {
		return json.Unmarshal(j, out)
	}
}