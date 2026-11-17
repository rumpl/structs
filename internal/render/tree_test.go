package render

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/report"
)

const (
	treeClientPackage = "client"
	treeClientType    = "Client"
	treeRoot          = "/repo"
)

func treeFixture(t *testing.T) *treeNode {
	t.Helper()

	data := report.Report{Packages: []report.Package{
		{Name: "main", Directory: "/repo/cmd/tool", Structs: []report.Struct{{Name: "Options", File: "/repo/cmd/tool/main.go"}}},
		{Name: treeClientPackage, Directory: "/repo/internal/client", Structs: []report.Struct{{Name: treeClientType, File: "/repo/internal/client/client.go"}}, Interfaces: []report.Interface{{Name: "Transport", File: "/repo/internal/client/client.go"}}},
		{Name: "client_test", Directory: "/repo/internal/client", Structs: []report.Struct{{Name: treeClientType, File: "/repo/internal/client/client_test.go"}}},
	}}
	view, err := htmlData(data)
	require.NoError(t, err)
	require.NotNil(t, view.Tree)

	return view.Tree
}

func TestSourceTreeRootAndFolderOrder(t *testing.T) {
	t.Parallel()
	tree := treeFixture(t)
	assert.Equal(t, "repo", tree.Name)
	assert.Equal(t, 4, tree.Count)
	assert.True(t, tree.Open)
	require.Len(t, tree.Children, 2)
	assert.Equal(t, "cmd", tree.Children[0].Name)
	assert.Equal(t, "internal", tree.Children[1].Name)
}

func TestSourceTreeGroupsFileTypes(t *testing.T) {
	t.Parallel()
	tree := treeFixture(t)
	client := tree.Children[1].Children[0]
	assert.Equal(t, treeClientPackage, client.Name)
	assert.Equal(t, 3, client.Count)
	require.Len(t, client.Children, 2)
	file := client.Children[0]
	assert.Equal(t, "client.go", file.Name)
	assert.Equal(t, "file", file.Kind)
	assert.Equal(t, 2, file.Count)
	assert.Equal(t, []treeEntry{
		{Name: treeClientType, ID: "struct-1-0", Kind: report.KindStruct, Package: treeClientPackage},
		{Name: "Transport", ID: "interface-1-0", Kind: report.KindInterface, Package: treeClientPackage},
	}, file.Entries)
}

func TestSourceTreeKeepsExternalTestPackage(t *testing.T) {
	t.Parallel()
	tree := treeFixture(t)
	file := tree.Children[1].Children[0].Children[1]
	assert.Equal(t, "client_test.go", file.Name)
	require.Len(t, file.Entries, 1)
	assert.Equal(t, "client_test", file.Entries[0].Package)
}

func TestSourceTreeCommonDirectory(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		root string
		dirs []string
	}{
		{name: "prefix is not ancestor", dirs: []string{"/repo/pkg", "/repo/pkg2"}, root: treeRoot},
		{name: "parent", dirs: []string{treeRoot, "/repo/sub"}, root: treeRoot},
		{name: "unrelated", dirs: []string{"/a", "/b"}, root: "/"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			section := htmlSection{Kind: report.KindStruct}
			for _, dir := range test.dirs {
				section.Packages = append(section.Packages, htmlPackage{Directory: dir, Types: []htmlType{{Name: "T", File: dir + "/types.go"}}})
			}

			tree, err := sourceTree([]htmlSection{section})
			require.NoError(t, err)
			require.NotNil(t, tree)
			assert.Equal(t, test.root, tree.Path)
			assert.Equal(t, 2, tree.Count)
		})
	}
}

func TestSourceTreeEmpty(t *testing.T) {
	t.Parallel()

	tree, err := sourceTree(nil)
	require.NoError(t, err)
	require.NotNil(t, tree)
	assert.Zero(t, tree.Count)
	assert.Empty(t, tree.Children)
}
