package analyzer

import (
	"go/ast"
	"go/token"

	"github.com/rumpl/structs/internal/report"
)

// referenceCollector traverses only type-expression edges. A generic AST walk
// would also visit field names, selector qualifiers, and array-length values,
// incorrectly treating those identifiers as type references.
type referenceCollector struct {
	fset       *token.FileSet
	bindings   map[string]string
	references []report.TypeReference
}

func typeReferences(fset *token.FileSet, expr ast.Expr, bindings map[string]string) []report.TypeReference {
	collector := referenceCollector{
		fset:       fset,
		bindings:   bindings,
		references: []report.TypeReference{},
	}
	collector.visit(expr, "direct")

	return collector.references
}

func referenceRelation(parent, role string) string {
	if parent == "direct" {
		return role
	}

	return parent + "/" + role
}

func (c *referenceCollector) add(name, pkg, relation string, position token.Pos) {
	pos := c.fset.PositionFor(position, false)
	c.references = append(c.references, report.TypeReference{
		Name:     name,
		Package:  pkg,
		Relation: relation,
		File:     pos.Filename,
		Line:     pos.Line,
	})
}

func (c *referenceCollector) visitFields(fields *ast.FieldList, relation, role string) {
	if fields == nil {
		return
	}

	for _, field := range fields.List {
		c.visit(field.Type, referenceRelation(relation, role))
	}
}

func (c *referenceCollector) visit(expr ast.Expr, relation string) {
	switch node := expr.(type) {
	case *ast.Ident:
		c.add(node.Name, "", relation, node.Pos())
	case *ast.SelectorExpr:
		if qualifier, ok := node.X.(*ast.Ident); ok {
			c.add(node.Sel.Name, c.bindings[qualifier.Name], relation, node.Pos())
		}
	case *ast.StarExpr:
		c.visit(node.X, referenceRelation(relation, "pointer"))
	case *ast.ParenExpr:
		c.visit(node.X, relation)
	case *ast.ArrayType:
		role := "array_element"
		if node.Len == nil {
			role = "slice_element"
		}

		c.visit(node.Elt, referenceRelation(relation, role))
	case *ast.MapType:
		c.visit(node.Key, referenceRelation(relation, "map_key"))
		c.visit(node.Value, referenceRelation(relation, "map_value"))
	case *ast.ChanType:
		c.visit(node.Value, referenceRelation(relation, "channel_element"))
	case *ast.Ellipsis:
		c.visit(node.Elt, referenceRelation(relation, "variadic_element"))
	case *ast.IndexExpr:
		c.visit(node.X, relation)
		c.visit(node.Index, referenceRelation(relation, "type_argument"))
	case *ast.IndexListExpr:
		c.visit(node.X, relation)

		for _, argument := range node.Indices {
			c.visit(argument, referenceRelation(relation, "type_argument"))
		}
	case *ast.FuncType:
		c.visitFields(node.TypeParams, relation, "constraint")
		c.visitFields(node.Params, relation, "parameter")
		c.visitFields(node.Results, relation, "result")
	case *ast.StructType:
		c.visitFields(node.Fields, relation, "struct_field")
	case *ast.InterfaceType:
		c.visitFields(node.Methods, relation, "interface_member")
	case *ast.UnaryExpr:
		c.visit(node.X, referenceRelation(relation, "approximation"))
	case *ast.BinaryExpr:
		c.visit(node.X, referenceRelation(relation, "union_term"))
		c.visit(node.Y, referenceRelation(relation, "union_term"))
	}
}
