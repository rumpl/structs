package analyzer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/analyzer"
	"github.com/rumpl/structs/internal/report"
)

const contextImportPath = "context"

func writeSource(t *testing.T, path, content string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func analyzeSource(t *testing.T, source string) report.Package {
	t.Helper()
	file := writeSource(t, filepath.Join(t.TempDir(), "types.go"), "package sample\n"+source)
	data, err := analyzer.Analyze([]string{file})
	require.NoError(t, err)
	require.Len(t, data.Packages, 1)

	return data.Packages[0]
}
