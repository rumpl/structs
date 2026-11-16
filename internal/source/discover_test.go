package source_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/source"
)

func sourceFiles(t *testing.T, names ...string) string {
	t.Helper()

	dir := t.TempDir()
	for _, name := range names {
		path := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte("package sample\n"), 0o600))
	}

	return dir
}

func TestDiscoverRecursesSortsAndDeduplicates(t *testing.T) {
	t.Parallel()
	dir := sourceFiles(t, "z.go", "child/a.go", "notes.txt")
	files, err := source.Discover([]string{dir, filepath.Join(dir, "z.go")}, source.Options{})
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(dir, "child", "a.go"), filepath.Join(dir, "z.go")}, files)
}

func TestDiscoverTestFileOption(t *testing.T) {
	t.Parallel()

	for _, include := range []bool{false, true} {
		t.Run(map[bool]string{false: "excluded", true: "included"}[include], func(t *testing.T) {
			t.Parallel()
			dir := sourceFiles(t, "a.go", "a_test.go")
			files, err := source.Discover([]string{dir}, source.Options{IncludeTests: include})
			require.NoError(t, err)

			want := []string{filepath.Join(dir, "a.go")}
			if include {
				want = append(want, filepath.Join(dir, "a_test.go"))
			}

			assert.Equal(t, want, files)
		})
	}
}

func TestDiscoverSkipsExcludedDirectories(t *testing.T) {
	t.Parallel()
	dir := sourceFiles(t, "a.go", "vendor/v.go", "child/vendor/v.go", ".git/x.go", "testdata/t.go", "node_modules/n.go", "vendorish/ok.go")
	files, err := source.Discover([]string{dir}, source.Options{})
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(dir, "a.go"), filepath.Join(dir, "vendorish", "ok.go")}, files)
}

func TestDiscoverAlwaysExcludesExplicitVendorPaths(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"vendor", "vendor/v.go", "child/vendor", "child/vendor/v.go"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := sourceFiles(t, "vendor/v.go", "child/vendor/v.go")
			files, err := source.Discover([]string{filepath.Join(dir, filepath.FromSlash(name))}, source.Options{IncludeTests: true})
			require.NoError(t, err)
			assert.Empty(t, files)
		})
	}
}

func TestDiscoverAllowsExplicitTestdata(t *testing.T) {
	t.Parallel()
	dir := sourceFiles(t, "testdata/t.go")
	files, err := source.Discover([]string{filepath.Join(dir, "testdata")}, source.Options{})
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(dir, "testdata", "t.go")}, files)
}

func TestDiscoverDoesNotFollowSymlinks(t *testing.T) {
	t.Parallel()
	dir := sourceFiles(t, "a.go")
	require.NoError(t, os.Symlink(filepath.Join(dir, "a.go"), filepath.Join(dir, "link.go")))
	files, err := source.Discover([]string{dir}, source.Options{})
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(dir, "a.go")}, files)
}

func TestDiscoverMissingPath(t *testing.T) {
	t.Parallel()
	_, err := source.Discover([]string{filepath.Join(t.TempDir(), "missing")}, source.Options{})
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestDiscoverRejectsExplicitNonGoFile(t *testing.T) {
	t.Parallel()
	dir := sourceFiles(t, "notes.txt")
	_, err := source.Discover([]string{filepath.Join(dir, "notes.txt")}, source.Options{})
	assert.ErrorContains(t, err, "not a Go source file")
}
