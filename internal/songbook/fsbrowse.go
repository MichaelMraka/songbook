package songbook

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type DirEntry struct {
	Name  string
	Path  string
	IsDir bool
}

func ListDir(path string) ([]DirEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	names, err := f.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	var out []DirEntry
	for _, name := range names {
		if name == "." || name == ".." {
			continue
		}
		full := filepath.Join(path, name)
		st, err := os.Stat(full)
		if err != nil {
			continue
		}
		out = append(out, DirEntry{Name: name, Path: full, IsDir: st.IsDir()})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func EntryLabel(e DirEntry) string {
	if e.IsDir {
		return "• " + e.Name + "/"
	}
	return e.Name
}
