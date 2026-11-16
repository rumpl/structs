package analyzer_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/query"
	"github.com/rumpl/structs/internal/report"
)

func TestNamedTypeKinds(t *testing.T) {
	t.Parallel()
	pkg := analyzeSource(t, "type Alias = string\ntype Counter int\ntype Record struct{}\ntype Reader interface{}")

	kinds := make(map[string]string)
	for i := range pkg.Types {
		kinds[pkg.Types[i].Name] = pkg.Types[i].Kind
	}

	assert.Equal(t, map[string]string{"Alias": report.KindAlias, "Counter": report.KindNamed, "Record": report.KindStruct, "Reader": report.KindInterface}, kinds)
}

func TestNestedFieldReferences(t *testing.T) {
	t.Parallel()

	for _, test := range []struct{ name, expression, relation string }{
		{"slice", "[]ctx.Context", "slice_element"},
		{"map pointer", "map[string]*ctx.Context", "map_value/pointer"},
		{"callback", "func(ctx.Context) error", "parameter"},
		{"nested channel", "struct { Input chan ctx.Context }", "struct_field/channel_element"},
		{"generic argument", "Box[ctx.Context]", "type_argument"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			pkg := analyzeSource(t, "import ctx \"context\"\ntype S struct { Value "+test.expression+" }")
			refs := pkg.Structs[0].Fields[0].References

			var contexts []report.TypeReference

			for _, ref := range refs {
				if ref.Package == contextImportPath {
					contexts = append(contexts, ref)
				}
			}

			require.Len(t, contexts, 1)
			assert.Equal(t, "Context", contexts[0].Name)
			assert.Equal(t, test.relation, contexts[0].Relation)
		})
	}
}

func TestFunctionSignature(t *testing.T) {
	t.Parallel()
	pkg := analyzeSource(t, "import ctx \"context\"\nfunc Fetch[T any](name string, c ctx.Context, rest ...T) (result T, err error) { panic(\"unused\") }")
	require.Len(t, pkg.Functions, 1)
	function := pkg.Functions[0]
	assert.Equal(t, "function", function.Kind)
	assert.True(t, function.Exported)
	require.Len(t, function.Parameters, 3)
	assert.Equal(t, 1, function.Parameters[1].Position)
	assert.Equal(t, contextImportPath, function.Parameters[1].TypePackage)
	assert.True(t, function.Parameters[2].Variadic)
	assert.Equal(t, "...T", function.Parameters[2].Type)
	assert.Equal(t, []string{"result", "err"}, []string{function.Results[0].Name, function.Results[1].Name})
	require.Len(t, function.TypeParameters, 1)
	assert.Equal(t, "any", function.TypeParameters[0].Type)
}

func TestGroupedParametersHaveSeparatePositions(t *testing.T) {
	t.Parallel()
	pkg := analyzeSource(t, "func Add(a, b int) (int,error) {return 0,nil}")
	parameters := pkg.Functions[0].Parameters
	require.Len(t, parameters, 2)
	assert.Equal(t, "a", parameters[0].Name)
	assert.Equal(t, "b", parameters[1].Name)
	assert.Equal(t, 0, parameters[0].Position)
	assert.Equal(t, 1, parameters[1].Position)
	assert.Equal(t, parameters[0].Type, parameters[1].Type)
}

func TestInterfaceMethodSignature(t *testing.T) {
	t.Parallel()
	pkg := analyzeSource(t, "type Reader interface { Read([]byte) (int,error) }")
	require.Len(t, pkg.Functions, 1)
	method := pkg.Functions[0]
	assert.Equal(t, "interface_method", method.Kind)
	assert.Equal(t, "Reader", method.ReceiverName)
	require.Len(t, method.Parameters, 1)
	assert.Empty(t, method.Parameters[0].Name)
	assert.Equal(t, "[]byte", method.Parameters[0].Type)
	require.Len(t, method.Results, 2)
	assert.Equal(t, "error", method.Results[1].Type)
}

func TestNamedReceiverMethodSQLLink(t *testing.T) {
	t.Parallel()
	pkg := analyzeSource(t, "type Counter int\nfunc (*Counter) Add() {}")
	rows, err := query.Execute(context.Background(), report.Report{Packages: []report.Package{pkg}},
		"SELECT t.name,m.name,f.pointer_receiver FROM types t JOIN methods m ON m.type_id=t.id JOIN functions f ON f.id=m.function_id")
	require.NoError(t, err)
	assert.Equal(t, [][]any{{"Counter", "Add", int64(1)}}, rows.Rows)
}

func TestPackageValuesDoNotInferInitializerTypes(t *testing.T) {
	t.Parallel()
	pkg := analyzeSource(t, "import \"context\"\nvar Global context.Context\nvar Inferred = make(chan int)")
	require.Len(t, pkg.Values, 2)
	assert.Equal(t, "context.Context", pkg.Values[0].Type)
	require.Len(t, pkg.Values[0].References, 1)
	assert.Equal(t, contextImportPath, pkg.Values[0].References[0].Package)
	assert.Empty(t, pkg.Values[1].Type)
	assert.Empty(t, pkg.Values[1].References)
	assert.Equal(t, "make(chan int)", pkg.Values[1].Initializer)
}

func TestGroupedValuesPreserveWholeInitializer(t *testing.T) {
	t.Parallel()
	pkg := analyzeSource(t, "var X, Y = pair()")
	require.Len(t, pkg.Values, 2)
	assert.Equal(t, "X", pkg.Values[0].Name)
	assert.Equal(t, "Y", pkg.Values[1].Name)
	assert.Equal(t, "pair()", pkg.Values[0].Initializer)
	assert.Equal(t, pkg.Values[0].Initializer, pkg.Values[1].Initializer)
}
