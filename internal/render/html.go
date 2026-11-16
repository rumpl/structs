package render

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/rumpl/structs/internal/report"
)

// These assets are trusted, embedded source, never report content.
//
//go:embed report.css
var reportCSS string

//go:embed report.js
var reportJS string

type htmlReport struct {
	Tree                *treeNode
	Sections            []htmlSection
	Packages            []report.Package
	DependencyPackages  []*bytes.Buffer
	DeclarationPackages []*bytes.Buffer
	HasPackages         bool
}

type htmlSection struct {
	Kind     string
	Title    string
	Packages []htmlPackage
	Count    int
}

type htmlPackage struct {
	Kind      string
	Content   *bytes.Buffer
	Name      string
	Directory string
	Types     []htmlType
}

type htmlType struct {
	report.Definition

	ID                 string
	Fields             []report.Field
	Methods            []report.Method
	EmbeddedTypes      []string
	FieldCount         int
	MethodCount        int
	PublicMethodCount  int
	PrivateMethodCount int
}

// htmlData gives both the TOC and content the same stable, unique anchors.
// Index-based IDs remain unambiguous for repeated names and build variants.
func htmlData(data report.Report) (htmlReport, error) {
	result := htmlReport{
		HasPackages: len(data.Packages) > 0,
		Packages:    data.Packages,
	}

	for _, kind := range []string{
		report.KindStruct,
		report.KindInterface,
	} {
		section := htmlSection{
			Kind:  kind,
			Title: "Structs",
		}
		if kind == report.KindInterface {
			section.Title = "Interfaces"
		}

		for pi := range data.Packages {
			pkg := &data.Packages[pi]

			group := htmlPackage{
				Kind:      kind,
				Name:      pkg.Name,
				Directory: pkg.Directory,
			}
			if kind == report.KindStruct {
				for ti := range pkg.Structs {
					typ := &pkg.Structs[ti]
					group.Types = append(group.Types, htmlType{
						Definition:         typ.Definition,
						ID:                 fmt.Sprintf("struct-%d-%d", pi, ti),
						Fields:             typ.Fields,
						FieldCount:         typ.FieldCount,
						MethodCount:        typ.MethodCount,
						PublicMethodCount:  typ.PublicMethodCount,
						PrivateMethodCount: typ.PrivateMethodCount,
						Methods:            typ.Methods,
						EmbeddedTypes:      typ.EmbeddedTypes,
					})
				}
			} else {
				for ti, typ := range pkg.Interfaces {
					group.Types = append(group.Types, htmlType{
						Definition:    typ.Definition,
						ID:            fmt.Sprintf("interface-%d-%d", pi, ti),
						MethodCount:   typ.MethodCount,
						Methods:       typ.Methods,
						EmbeddedTypes: typ.EmbeddedTypes,
					})
				}
			}

			if len(group.Types) > 0 {
				section.Packages = append(section.Packages, group)
				section.Count += len(group.Types)
			}
		}

		result.Sections = append(result.Sections, section)
	}

	tree, err := sourceTree(result.Sections)
	if err != nil {
		return htmlReport{}, err
	}

	if tree.Count > 0 {
		result.Tree = tree
	}

	return result, nil
}
