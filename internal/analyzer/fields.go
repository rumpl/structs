package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"path"
	"strconv"

	"github.com/rumpl/structs/internal/report"
)

// fileImports keeps file-local aliases separate: two files in a package may
// bind the same alias to different import paths.
type importScope struct {
	bindings map[string]string
	imports  []report.Import
}

func fileImports(fset *token.FileSet, file *ast.File) (importScope, error) {
	imports := make([]report.Import, 0, len(file.Imports))
	bindings := make(map[string]string)

	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return importScope{}, fmt.Errorf("decode import at %s: %w", fset.PositionFor(spec.Pos(), false), err)
		}

		alias := ""
		binding := path.Base(importPath)

		if spec.Name != nil {
			alias = spec.Name.Name
			binding = alias
		}

		pos := fset.PositionFor(spec.Pos(), false)

		imports = append(imports, report.Import{
			Path:  importPath,
			Alias: alias,
			File:  pos.Filename,
			Line:  pos.Line,
		})
		if binding != "_" && binding != "." {
			bindings[binding] = importPath
		}
	}

	return importScope{imports: imports, bindings: bindings}, nil
}

func (c *fileCollector) analyzeFields(node *ast.StructType) []report.Field {
	result := []report.Field{}

	for _, field := range node.Fields.List {
		reference := fieldType(field.Type, c.bindings)

		tag := ""

		if field.Tag != nil {
			var err error

			tag, err = strconv.Unquote(field.Tag.Value)
			if err != nil {
				c.err = fmt.Errorf("decode field tag at %s: %w", c.fset.PositionFor(field.Tag.Pos(), false), err)

				return nil
			}
		}

		base := report.Field{
			Type:        c.format(field.Type),
			TypeName:    reference.name,
			TypePackage: reference.packagePath,
			Tag:         tag,
			References:  typeReferences(c.fset, field.Type, c.bindings),
		}
		if len(field.Names) == 0 {
			base.Name = embeddedName(field.Type)
			base.Embedded = true
			base.Exported = ast.IsExported(base.Name)
			pos := c.fset.PositionFor(field.Pos(), false)
			base.File, base.Line = pos.Filename, pos.Line
			result = append(result, base)

			continue
		}

		for _, name := range field.Names {
			entry := base
			entry.Name, entry.Exported = name.Name, ast.IsExported(name.Name)
			pos := c.fset.PositionFor(name.Pos(), false)
			entry.File, entry.Line = pos.Filename, pos.Line
			result = append(result, entry)
		}
	}

	return result
}

// Only the outer named type (through pointers/parentheses) is associated with
// an import. Slices, maps, functions, and anonymous types are not direct holds
// of their element/parameter types. Generic arguments retain their source text.
type namedTypeReference struct {
	name        string
	packagePath string
}

func fieldType(expr ast.Expr, bindings map[string]string) namedTypeReference {
	switch expr := expr.(type) {
	case *ast.StarExpr:
		return fieldType(expr.X, bindings)
	case *ast.ParenExpr:
		return fieldType(expr.X, bindings)
	case *ast.IndexExpr:
		return fieldType(expr.X, bindings)
	case *ast.IndexListExpr:
		return fieldType(expr.X, bindings)
	case *ast.Ident:
		return namedTypeReference{name: expr.Name}
	case *ast.SelectorExpr:
		if qualifier, ok := expr.X.(*ast.Ident); ok {
			return namedTypeReference{name: expr.Sel.Name, packagePath: bindings[qualifier.Name]}
		}
	}

	return namedTypeReference{}
}

func embeddedName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.SelectorExpr:
		return expr.Sel.Name
	case *ast.StarExpr:
		return embeddedName(expr.X)
	case *ast.IndexExpr:
		return embeddedName(expr.X)
	case *ast.IndexListExpr:
		return embeddedName(expr.X)
	case *ast.ParenExpr:
		return embeddedName(expr.X)
	case *ast.Ident:
		return expr.Name
	}

	return ""
}
