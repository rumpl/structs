package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/testutil"
)

func TestParallelMatchesSequential(t *testing.T) {
	t.Parallel()
	files := testutil.Project(t, 4, 1)
	sequential, err := analyzeWithWorkers(files, 1)
	require.NoError(t, err)
	parallel, err := analyzeWithWorkers(files, 4)
	require.NoError(t, err)
	assert.Equal(t, sequential, parallel)
}

func TestParallelReturnsFirstInputError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	paths := []string{filepath.Join(dir, "z.go"), filepath.Join(dir, "a.go")}
	for _, path := range paths {
		require.NoError(t, os.WriteFile(path, []byte("package broken\ntype T struct {"), 0o600))
	}

	for range 5 {
		data, err := analyzeWithWorkers(paths, 2)
		require.Error(t, err)
		require.ErrorContains(t, err, paths[0])
		assert.Empty(t, data.Packages)
	}
}
