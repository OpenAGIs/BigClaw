package regression

import (
	"io/fs"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestRepoPythonInventoryStaysOnPackageOnlySurface(t *testing.T) {
	root := filepath.Clean(filepath.Join(repoRoot(t), ".."))
	var got []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".py" {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			got = append(got, filepath.ToSlash(rel))
		}
		if d.Name() == "pyproject.toml" || d.Name() == "setup.py" {
			t.Fatalf("unexpected Python packaging file present: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo python inventory: %v", err)
	}

	sort.Strings(got)
	var want []string
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected Python inventory:\n got=%v\nwant=%v", got, want)
	}
}
