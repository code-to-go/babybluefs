package store

func Exists(f FS, name string) bool {
	_, err := f.Stat(name)
	return err == nil
}
