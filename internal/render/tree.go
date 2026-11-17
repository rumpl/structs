package render

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// treeNode is presentation-only: source directories/files contain the same type
// anchors used by the report cards, regardless of their struct/interface view.
type treeNode struct {
	Name     string
	Path     string
	Kind     string
	Children []*treeNode
	Entries  []treeEntry
	Count    int
	Open     bool
}

type treeEntry struct {
	Name    string
	ID      string
	Kind    string
	Package string
}

func sourceTree(sections []htmlSection) (*treeNode, error) {
	var directories []string

	for _, section := range sections {
		for _, pkg := range section.Packages {
			directories = append(directories, filepath.Clean(pkg.Directory))
		}
	}

	if len(directories) == 0 {
		return &treeNode{}, nil
	}

	common := directories[0]
	for _, directory := range directories[1:] {
		for !withinDirectory(common, directory) {
			parent := filepath.Dir(common)
			if parent == common {
				break
			}

			common = parent
		}
	}

	root := &treeNode{
		Name: filepath.Base(common),
		Path: common,
		Kind: "directory",
		Open: true,
	}

	for _, section := range sections {
		for _, pkg := range section.Packages {
			relative, err := filepath.Rel(common, filepath.Clean(pkg.Directory))
			if err != nil {
				return nil, fmt.Errorf("build source tree for %q: %w", pkg.Directory, err)
			}

			node := root

			if relative != "." {
				for component := range strings.SplitSeq(relative, string(filepath.Separator)) {
					node = node.child(component, "directory", filepath.Join(node.Path, component))
				}
			}

			for i := range pkg.Types {
				typ := &pkg.Types[i]
				file := node.child(filepath.Base(typ.File), "file", typ.File)
				file.Entries = append(file.Entries, treeEntry{
					Name:    typ.Name,
					ID:      typ.ID,
					Kind:    section.Kind,
					Package: pkg.Name,
				})
			}
		}
	}

	root.finish()

	return root, nil
}

func withinDirectory(parent, path string) bool {
	rel, err := filepath.Rel(parent, path)

	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (n *treeNode) child(name, kind, path string) *treeNode {
	for _, child := range n.Children {
		if child.Name == name && child.Kind == kind {
			return child
		}
	}

	child := &treeNode{
		Name: name,
		Path: path,
		Kind: kind,
	}
	n.Children = append(n.Children, child)

	return child
}

func (n *treeNode) finish() {
	sort.Slice(n.Children, func(i, j int) bool {
		a, b := n.Children[i], n.Children[j]
		if a.Kind != b.Kind {
			return a.Kind == "directory"
		}

		return a.Name < b.Name
	})
	sort.SliceStable(n.Entries, func(i, j int) bool {
		a, b := n.Entries[i], n.Entries[j]
		if a.Name != b.Name {
			return a.Name < b.Name
		}

		if a.Package != b.Package {
			return a.Package < b.Package
		}

		return a.ID < b.ID
	})

	n.Count = len(n.Entries)
	for _, child := range n.Children {
		child.finish()
		n.Count += child.Count
	}
}
