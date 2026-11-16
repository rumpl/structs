package analyzer

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectorReturnsImportError(t *testing.T) {
	t.Parallel()

	collector := newPackageCollector(token.NewFileSet(), packageKey{})
	file := &ast.File{Imports: []*ast.ImportSpec{{Path: &ast.BasicLit{Value: `"bad\q"`}}}}
	assert.Nil(t, collector.Visit(file))
	assert.ErrorContains(t, collector.err, "decode import")
}

func TestCollectorRetainsFieldTagError(t *testing.T) {
	t.Parallel()

	collector := newPackageCollector(token.NewFileSet(), packageKey{})
	visitor := &fileCollector{packageCollector: collector}
	fields := &ast.StructType{Fields: &ast.FieldList{List: []*ast.Field{{
		Names: []*ast.Ident{ast.NewIdent("Field")}, Type: ast.NewIdent("string"), Tag: &ast.BasicLit{Value: `"bad\q"`},
	}}}}
	assert.Nil(t, visitor.analyzeFields(fields))
	assert.ErrorContains(t, collector.err, "decode field tag")
}

func TestFormatterPreservesFirstError(t *testing.T) {
	t.Parallel()

	formatter := &syntaxFormatter{fset: token.NewFileSet()}
	assert.Empty(t, formatter.format(&ast.Comment{}))
	require.ErrorContains(t, formatter.err, "format Go declaration")
	first := formatter.err
	assert.Empty(t, formatter.format(ast.NewIdent("Later")))
	assert.Same(t, first, formatter.err)
}
