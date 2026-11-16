// Package source discovers Go source files without loading dependencies.
package source

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

var errNotGoSource = errors.New("not a Go source file")

// Options controls source discovery.
type Options struct{ IncludeTests bool }

// Discover returns sorted unique absolute paths, excluding vendor even when
// explicitly selected. Directory symlinks and file symlinks are not followed.
func Discover(paths []string, opts Options) ([]string, error) {
	if len(paths) == 0 {
		paths = []string{"."}
	}

	files := make(map[string]struct{})

	for _, path := range paths {
		root, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve %q: %w", path, err)
		}

		if withinVendor(root) {
			continue
		}

		err = filepath.WalkDir(root, sourceVisitor(root, opts, files))
		if err != nil {
			return nil, fmt.Errorf("scan %q: %w", path, err)
		}
	}

	result := make([]string, 0, len(files))
	for file := range files {
		result = append(result, file)
	}

	sort.Strings(result)

	return result, nil
}

func sourceVisitor(root string, opts Options, files map[string]struct{}) fs.WalkDirFunc {
	return func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			if path != root && skipDirectory(entry.Name()) {
				return filepath.SkipDir
			}

			return nil
		}

		if !entry.Type().IsRegular() {
			return nil
		}

		if filepath.Ext(path) != ".go" {
			if path == root {
				return fmt.Errorf("%w: %s", errNotGoSource, path)
			}

			return nil
		}

		if opts.IncludeTests || !strings.HasSuffix(path, "_test.go") {
			files[path] = struct{}{}
		}

		return nil
	}
}

func withinVendor(path string) bool {
	return slices.Contains(strings.Split(filepath.Clean(path), string(filepath.Separator)), "vendor")
}

func skipDirectory(name string) bool {
	return strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" || name == "node_modules"
}
