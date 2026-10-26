package blueprint

import (
	"io/fs"
	"maps"
	"slices"
)

type File struct {
	Content []byte
	Mode    fs.FileMode
}

type Command struct {
	Name string
	Args []string
}

type Blueprint struct {
	Files    map[string]File
	Commands []Command
}

func (b Blueprint) Paths() []string {
	return slices.Sorted(maps.Keys(b.Files))
}
