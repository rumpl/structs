package cli_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/report"
)

func TestReportCommandJSON(t *testing.T) {
	t.Parallel()
	out, err := execute(t, commandSource(t))
	require.NoError(t, err)

	var data report.Report
	require.NoError(t, json.Unmarshal([]byte(out), &data))
	require.Len(t, data.Packages, 1)
	require.Len(t, data.Packages[0].Structs, 1)
	assert.Equal(t, "Box", data.Packages[0].Structs[0].Name)
	assert.Equal(t, 1, data.Packages[0].Structs[0].PublicMethodCount)
	require.Len(t, data.Packages[0].Interfaces, 1)
	assert.Equal(t, "Reader", data.Packages[0].Interfaces[0].Name)
}

func TestReportCommandIncludeTests(t *testing.T) {
	t.Parallel()
	out, err := execute(t, "--include-tests", commandSource(t))
	require.NoError(t, err)

	var data report.Report
	require.NoError(t, json.Unmarshal([]byte(out), &data))
	require.Len(t, data.Packages, 1)
	require.Len(t, data.Packages[0].Structs, 2)
	assert.Equal(t, "TestOnly", data.Packages[0].Structs[1].Name)
}

func TestReportCommandTextFormats(t *testing.T) {
	t.Parallel()

	for _, test := range []struct{ format, marker string }{{"markdown", "# Go type report"}, {"html", "<!doctype html>"}} {
		t.Run(test.format, func(t *testing.T) {
			t.Parallel()
			out, err := execute(t, formatFlag, test.format, commandSource(t))
			require.NoError(t, err)
			assert.Contains(t, out, test.marker)
			assert.Contains(t, out, "Reader")
		})
	}
}

func TestReportCommandRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name    string
		message string
		args    []string
	}{
		{name: "format", args: []string{formatFlag, "csv"}, message: "unsupported format"},
		{name: "path", args: []string{filepath.Join(t.TempDir(), "missing")}, message: "missing"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			out, err := execute(t, test.args...)
			require.ErrorContains(t, err, test.message)
			assert.Empty(t, out)
		})
	}
}
