package analyzer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/analyzer"
)

func TestAnalyzeAssociatesMethodsAcrossFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	types := writeSource(t, filepath.Join(dir, "types.go"), "package sample\ntype Box struct{}\n")
	methods := writeSource(t, filepath.Join(dir, "methods.go"), "package sample\nfunc (*Box) Public() {}\nfunc (Box) private() {}\n")
	data, err := analyzer.Analyze([]string{methods, types})
	require.NoError(t, err)
	require.Len(t, data.Packages, 1)
	require.Len(t, data.Packages[0].Structs, 1)
	box := data.Packages[0].Structs[0]
	assert.Equal(t, 2, box.MethodCount)
	assert.Equal(t, 1, box.PublicMethodCount)
	assert.Equal(t, 1, box.PrivateMethodCount)
	assert.Equal(t, types, box.File)
	assert.Equal(t, 2, box.Line)
	require.Len(t, box.Methods, 2)
	assert.Equal(t, "func (*Box) Public()", box.Methods[0].Declaration)
	assert.Equal(t, methods, box.Methods[0].File)
	assert.Equal(t, 2, box.Methods[0].Line)
}

func TestGenericReceiverAssociation(t *testing.T) {
	t.Parallel()

	for _, receiver := range []string{"*Box[T]", "Box[T]", "*Box[T, U]"} {
		t.Run(receiver, func(t *testing.T) {
			t.Parallel()
			pkg := analyzeSource(t, "type Box[T any, U any] struct{}\nfunc (b "+receiver+") Get() {}")
			require.Len(t, pkg.Structs, 1)
			require.Len(t, pkg.Structs[0].Methods, 1)
			assert.Equal(t, "Get", pkg.Structs[0].Methods[0].Name)
		})
	}
}

func TestAnalyzeSeparatesPackages(t *testing.T) {
	t.Parallel()

	for _, test := range []struct{ name, path, packageName string }{
		{"different directory", "child/types.go", "sample"},
		{"external tests", "types_test.go", "sample_test"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			a := writeSource(t, filepath.Join(dir, "types.go"), "package sample\ntype Item struct{}")
			b := writeSource(t, filepath.Join(dir, filepath.FromSlash(test.path)), "package "+test.packageName+"\ntype Item struct{}\nfunc (Item) Run() {}")
			data, err := analyzer.Analyze([]string{a, b})
			require.NoError(t, err)
			require.Len(t, data.Packages, 2)
			assert.Zero(t, data.Packages[0].Structs[0].MethodCount)
			assert.Equal(t, 1, data.Packages[1].Structs[0].MethodCount)
		})
	}
}

func TestInterfaceDeclaredMethods(t *testing.T) {
	t.Parallel()
	pkg := analyzeSource(t, "type Reader interface {\n Read(p []byte) (n int, err error)\n close() error\n BaseReader\n}")
	require.Len(t, pkg.Interfaces, 1)
	reader := pkg.Interfaces[0]
	assert.Equal(t, 2, reader.MethodCount)
	assert.Equal(t, []string{"BaseReader"}, reader.EmbeddedTypes)
	require.Len(t, reader.Methods, 2)
	assert.Equal(t, "Read(p []byte) (n int, err error)", reader.Methods[0].Declaration)
	assert.True(t, reader.Methods[0].Exported)
	assert.False(t, reader.Methods[1].Exported)
}

func TestStructFieldCount(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name, fields string
		want         int
	}{
		{"empty", "", 0},
		{"grouped", "A, B int", 2},
		{"embedded", "Base; *other.External", 2},
		{"nested", "Nested struct { A, B int }", 1},
		{"blank", "_ [8]byte", 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			pkg := analyzeSource(t, "type S struct {"+test.fields+"}")
			require.Len(t, pkg.Structs, 1)
			assert.Equal(t, test.want, pkg.Structs[0].FieldCount)
			assert.Len(t, pkg.Structs[0].Fields, test.want)
		})
	}
}

func TestAnalyzeSyntaxErrorHasLocation(t *testing.T) {
	t.Parallel()
	file := writeSource(t, filepath.Join(t.TempDir(), "broken.go"), "package sample\ntype Broken struct {")
	data, err := analyzer.Analyze([]string{file})
	require.Error(t, err)
	require.ErrorContains(t, err, "broken.go:")
	assert.Empty(t, data.Packages)
}

func TestAnalyzeMissingFile(t *testing.T) {
	t.Parallel()
	_, err := analyzer.Analyze([]string{filepath.Join(t.TempDir(), "missing.go")})
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestAnalyzeUsesPhysicalPositions(t *testing.T) {
	t.Parallel()
	pkg := analyzeSource(t, "//line fake.go:500\ntype Empty struct{}")
	require.Len(t, pkg.Structs, 1)
	assert.Equal(t, 3, pkg.Structs[0].Line)
	assert.Equal(t, "types.go", filepath.Base(pkg.Structs[0].File))
}

func TestAnalyzeEmptyCollections(t *testing.T) {
	t.Parallel()
	pkg := analyzeSource(t, "type Empty struct{}\ntype API interface{}")
	assert.NotNil(t, pkg.Dependencies)
	assert.NotNil(t, pkg.Imports)
	assert.NotNil(t, pkg.Values)
	assert.NotNil(t, pkg.Functions)
	require.Len(t, pkg.Structs, 1)
	assert.NotNil(t, pkg.Structs[0].Methods)
	assert.NotNil(t, pkg.Structs[0].Fields)
	require.Len(t, pkg.Interfaces, 1)
	assert.NotNil(t, pkg.Interfaces[0].Methods)
}
