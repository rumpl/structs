package analyzer_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/analyzer"
)

func TestFieldTypeMetadata(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name, declaration, typeName, packagePath string
		embedded                                 bool
	}{
		{"aliased", "Ctx ctx.Context", "Context", contextImportPath, false},
		{"pointer", "Client *http.Client", "Client", "net/http", false},
		{"embedded", "ctx.Context", "Context", contextImportPath, true},
		{"slice", "Requests []ctx.Context", "", "", false},
		{"nested", "Nested struct { Ctx ctx.Context }", "", "", false},
		{"callback", "Callback func(ctx.Context)", "", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			pkg := analyzeSource(t, "import ctx \"context\"\nimport \"net/http\"\ntype Holder struct {"+test.declaration+"}")
			require.Len(t, pkg.Structs, 1)
			require.Len(t, pkg.Structs[0].Fields, 1)
			field := pkg.Structs[0].Fields[0]
			assert.Equal(t, test.typeName, field.TypeName)
			assert.Equal(t, test.packagePath, field.TypePackage)
			assert.Equal(t, test.embedded, field.Embedded)
		})
	}
}

func TestFieldTagsVisibilityAndLocation(t *testing.T) {
	t.Parallel()
	pkg := analyzeSource(t, "type Holder struct {\n Ctx string `json:\"ctx\"`\n private int\n _ bool\n}")
	fields := pkg.Structs[0].Fields
	require.Len(t, fields, 3)
	assert.Equal(t, `json:"ctx"`, fields[0].Tag)
	assert.Equal(t, 3, fields[0].Line)
	assert.True(t, fields[0].Exported)
	assert.False(t, fields[1].Exported)
	assert.False(t, fields[2].Exported)
}

func TestImportAliasesAreFileLocal(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	a := writeSource(t, filepath.Join(dir, "a.go"), "package sample\nimport ctx \"context\"\ntype A struct { Ctx ctx.Context }")
	b := writeSource(t, filepath.Join(dir, "b.go"), "package sample\nimport ctx \"example.org/other\"\ntype B struct { Ctx ctx.Context }")
	data, err := analyzer.Analyze([]string{b, a})
	require.NoError(t, err)
	require.Len(t, data.Packages, 1)
	structs := data.Packages[0].Structs
	require.Len(t, structs, 2)
	assert.Equal(t, contextImportPath, structs[0].Fields[0].TypePackage)
	assert.Equal(t, "example.org/other", structs[1].Fields[0].TypePackage)
}

func TestDependenciesDeduplicateImportPaths(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	a := writeSource(t, filepath.Join(dir, "a.go"), "package sample\nimport ctx \"context\"\nimport _ \"embed\"")
	b := writeSource(t, filepath.Join(dir, "b.go"), "package sample\nimport \"context\"\nimport . \"fmt\"")
	data, err := analyzer.Analyze([]string{b, a})
	require.NoError(t, err)

	pkg := data.Packages[0]
	assert.Equal(t, []string{contextImportPath, "embed", "fmt"}, pkg.Dependencies)
	require.Len(t, pkg.Imports, 4)
	assert.Equal(t, "ctx", pkg.Imports[0].Alias)
	assert.Equal(t, a, pkg.Imports[0].File)
	assert.Equal(t, 2, pkg.Imports[0].Line)
	assert.Equal(t, "_", pkg.Imports[1].Alias)
	assert.Empty(t, pkg.Imports[2].Alias)
	assert.Equal(t, ".", pkg.Imports[3].Alias)
}
