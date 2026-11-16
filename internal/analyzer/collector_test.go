package analyzer_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/analyzer"
	"github.com/rumpl/structs/internal/report"
)

func collectorSources(t *testing.T) []string {
	t.Helper()

	dir := t.TempDir()
	methods := writeSource(t, filepath.Join(dir, "a_methods.go"), `package sample
import ctx "context"
func (s *Server[T]) Zebra(ctx ctx.Context) error { return nil }
func (s Server[T]) Alpha() {}
func Local() { type Hidden struct{}; const hidden = 1; var ignored int; _ = ignored }
var Factory = func() { type NotPackageLevel int }
const ( A int = iota; B )
const ( C string = "value"; D )
`)
	types := writeSource(t, filepath.Join(dir, "z_types.go"), `package sample
import "context"
type Server[T any] struct { Z int; A, B context.Context }
type Reader interface { Zebra(context.Context) error; Alpha() }
type Alias = struct { Omitted int }
`)

	return []string{types, methods}
}

func collectorPackage(t *testing.T) report.Package {
	t.Helper()

	data, err := analyzer.Analyze(collectorSources(t))
	require.NoError(t, err)
	require.Len(t, data.Packages, 1)

	return data.Packages[0]
}

func TestCollectorPackageScope(t *testing.T) {
	t.Parallel()

	pkg := collectorPackage(t)
	counts := map[string]int{
		"types":      len(pkg.Types),
		"structs":    len(pkg.Structs),
		"interfaces": len(pkg.Interfaces),
		"functions":  len(pkg.Functions),
		"values":     len(pkg.Values),
	}
	want := map[string]int{
		"types":      3,
		"structs":    1,
		"interfaces": 1,
		"functions":  5,
		"values":     5,
	}

	assert.Equal(t, want, counts)
}

func TestCollectorMethodOrdering(t *testing.T) {
	t.Parallel()

	pkg := collectorPackage(t)
	methods := pkg.Structs[0].Methods

	require.Len(t, methods, 2)
	assert.Equal(t, "Alpha", methods[0].Name)
	assert.Equal(t, "Zebra", methods[1].Name)
}

func TestCollectorFieldSourceOrder(t *testing.T) {
	t.Parallel()

	pkg := collectorPackage(t)
	names := make([]string, 0, len(pkg.Structs[0].Fields))

	for i := range pkg.Structs[0].Fields {
		names = append(names, pkg.Structs[0].Fields[i].Name)
	}

	assert.Equal(t, []string{"Z", "A", "B"}, names)
}

func TestCollectorNamedTypeMethodProjection(t *testing.T) {
	t.Parallel()

	pkg := collectorPackage(t)
	for i := range pkg.Types {
		typ := &pkg.Types[i]
		if typ.Name != "Server" {
			continue
		}

		assert.Equal(t, pkg.Structs[0].Methods, typ.ReceiverMethods)

		return
	}

	assert.Fail(t, "missing Server type")
}

func TestCollectorFunctionSignatureProjection(t *testing.T) {
	t.Parallel()

	pkg := collectorPackage(t)
	functions := make(map[string]*report.Function)

	for i := range pkg.Functions {
		function := &pkg.Functions[i]
		if function.Kind == "method" {
			functions[function.Name] = function
		}
	}

	for i := range pkg.Structs[0].Methods {
		method := &pkg.Structs[0].Methods[i]
		function, found := functions[method.Name]

		require.True(t, found, "missing canonical function: %s", method.Name)
		assert.Equal(t, method.Declaration, function.Declaration)
		assert.Equal(t, method.Signature, function.Signature)
		assert.Equal(t, method.SignatureParameters, function.SignatureParameters)
	}
}

func TestCollectorConstGroupInheritance(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name        string
		typeName    string
		initializer string
	}{
		{name: "B", typeName: "int", initializer: "iota"},
		{name: "D", typeName: "string", initializer: `"value"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			pkg := collectorPackage(t)
			for i := range pkg.Values {
				value := &pkg.Values[i]
				if value.Name != test.name {
					continue
				}

				assert.Equal(t, test.typeName, value.Type)
				assert.Equal(t, test.initializer, value.Initializer)
				assert.True(t, value.Inherited)

				return
			}

			assert.Fail(t, "missing constant", "name: %s", test.name)
		})
	}
}

func TestCollectorInputOrderIndependence(t *testing.T) {
	t.Parallel()

	paths := collectorSources(t)

	data, err := analyzer.Analyze(paths)
	require.NoError(t, err)

	reordered, err := analyzer.Analyze([]string{paths[1], paths[0], paths[1]})
	require.NoError(t, err)
	assert.Equal(t, data, reordered, "collector output depends on input order")
}
