package query_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/query"
	"github.com/rumpl/structs/internal/report"
)

const (
	contextPath       = "context"
	integerConstraint = "~int"
	integerType       = "int"
	readMethod        = "Read"
	errorType         = "error"
	directRelation    = "direct"
	contextExpression = "context.Context"
	contextTypeName   = "Context"
)

func TestLoadImportsAndDependencies(t *testing.T) {
	t.Parallel()

	data := report.Report{Packages: []report.Package{{
		Name:         samplePackage,
		Dependencies: []string{contextPath}, Imports: []report.Import{{Path: contextPath, Alias: "ctx", File: "types.go", Line: 2}},
	}}}
	result, err := query.Execute(context.Background(), data, "SELECT p.name,d.import_path,i.alias,i.file,i.line FROM packages p JOIN package_dependencies d ON d.package_id=p.id JOIN imports i ON i.package_id=p.id")
	require.NoError(t, err)
	assert.Equal(t, [][]any{{samplePackage, contextPath, "ctx", "types.go", int64(2)}}, result.Rows)
}

func TestLoadGenericConstraints(t *testing.T) {
	t.Parallel()

	constraint := report.Parameter{Name: "T", Type: integerConstraint, References: []report.TypeReference{{Name: integerType, Relation: "approximation"}}}
	data := report.Report{Packages: []report.Package{{
		Name:      samplePackage,
		Types:     []report.NamedType{{Definition: report.Definition{Name: "Box"}, Kind: report.KindStruct, TypeParameters: []report.Parameter{constraint}}},
		Functions: []report.Function{{Definition: report.Definition{Name: "Make"}, FunctionDeclaration: report.FunctionDeclaration{Kind: "function"}, SignatureParameters: report.SignatureParameters{TypeParameters: []report.Parameter{constraint}}}},
	}}}
	result, err := query.Execute(context.Background(), data, "SELECT tp.name,tp.constraint_expression,r.name,(tp.type_id IS NOT NULL),(tp.function_id IS NOT NULL) FROM type_parameters tp JOIN type_references r ON r.type_parameter_id=tp.id ORDER BY tp.id")
	require.NoError(t, err)
	assert.Equal(t, [][]any{{"T", integerConstraint, integerType, int64(1), int64(0)}, {"T", integerConstraint, integerType, int64(0), int64(1)}}, result.Rows)
}

func TestLoadParametersAndResults(t *testing.T) {
	t.Parallel()

	data := report.Report{Packages: []report.Package{{Name: samplePackage, Functions: []report.Function{{
		Definition: report.Definition{Name: readMethod}, FunctionDeclaration: report.FunctionDeclaration{Kind: "function"},
		SignatureParameters: report.SignatureParameters{Parameters: []report.Parameter{{Name: "input", Type: "...byte", Variadic: true}}, Results: []report.Parameter{{Type: errorType, References: []report.TypeReference{{Name: errorType, Relation: directRelation}}}}},
	}}}}}
	result, err := query.Execute(context.Background(), data, "SELECT f.name,p.role,p.name,p.type,p.variadic FROM functions f JOIN parameters p ON p.function_id=f.id ORDER BY p.id")
	require.NoError(t, err)
	assert.Equal(t, [][]any{{readMethod, "parameter", "input", "...byte", int64(1)}, {readMethod, "result", "", errorType, int64(0)}}, result.Rows)
}

func TestLoadPackageValueReferences(t *testing.T) {
	t.Parallel()

	data := report.Report{Packages: []report.Package{{Name: samplePackage, Values: []report.Value{{Definition: report.Definition{Name: "Global"}, Kind: "var", Type: contextExpression, References: []report.TypeReference{{Name: contextTypeName, Package: contextPath, Relation: directRelation}}}}}}}
	result, err := query.Execute(context.Background(), data, "SELECT v.name,v.type,r.name,r.package FROM package_values v JOIN type_references r ON r.value_id=v.id")
	require.NoError(t, err)
	assert.Equal(t, [][]any{{"Global", contextExpression, contextTypeName, contextPath}}, result.Rows)
}

func TestLoadFieldReferencesAndTags(t *testing.T) {
	t.Parallel()

	data := report.Report{Packages: []report.Package{{Name: samplePackage, Structs: []report.Struct{{Definition: report.Definition{Name: sampleStruct}, Fields: []report.Field{{Name: "Ctx", Type: contextExpression, Tag: `json:"ctx"`, Exported: true, References: []report.TypeReference{{Name: contextTypeName, Package: contextPath, Relation: directRelation}}}}}}}}}
	result, err := query.Execute(context.Background(), data, "SELECT s.name,f.name,f.tag,f.exported,r.name FROM structs s JOIN fields f ON f.type_id=s.id JOIN type_references r ON r.field_id=f.id")
	require.NoError(t, err)
	assert.Equal(t, [][]any{{sampleStruct, "Ctx", `json:"ctx"`, int64(1), contextTypeName}}, result.Rows)
}
