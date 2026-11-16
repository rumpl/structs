package cli_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/query"
)

func TestQueryCommandForms(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		prefix []string
	}{
		{"shorthand", nil}, {"explicit", []string{querySubcommand}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			args := slices.Clone(test.prefix)
			args = append(args, fieldCountQuery, commandSource(t))
			out, err := execute(t, args...)
			require.NoError(t, err)

			var result query.Result
			require.NoError(t, json.Unmarshal([]byte(out), &result))
			assert.Equal(t, []string{"name", "field_count"}, result.Columns)
			assert.Equal(t, [][]any{{"Box", float64(2)}}, result.Rows)
		})
	}
}

func TestQueryCommandFormatFlagPlacement(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		marker string
		args   []string
	}{
		{name: "after subcommand", args: []string{querySubcommand, formatFlag, "markdown", fieldCountQuery}, marker: "| name | field&#95;count |"},
		{name: "before subcommand", args: []string{formatFlag, "html", querySubcommand, fieldCountQuery}, marker: "<th scope=\"col\">field_count</th>"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			out, err := execute(t, append(test.args, commandSource(t))...)
			require.NoError(t, err)
			assert.Contains(t, out, test.marker)
			assert.Contains(t, out, "Box")
		})
	}
}

func TestQuerySchemaNeedsNoSource(t *testing.T) {
	t.Parallel()
	out, err := execute(t, querySubcommand, "--schema")
	require.NoError(t, err)
	assert.Equal(t, query.Schema, out)
}

func TestQueryCommandErrorsEmitNoReport(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name    string
		message string
		args    []string
	}{
		{name: "missing SQL", args: []string{querySubcommand}, message: "requires at least"},
		{name: "schema arguments", args: []string{querySubcommand, "--schema", "SELECT 1"}, message: "does not accept"},
		{name: "unknown table", args: []string{querySubcommand, "SELECT * FROM missing", t.TempDir()}, message: "no such table"},
		{name: "write", args: []string{querySubcommand, "DELETE FROM types", t.TempDir()}, message: "read-only"},
		{name: "format", args: []string{querySubcommand, formatFlag, "csv", "SELECT 1", t.TempDir()}, message: "unsupported format"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			out, err := execute(t, test.args...)
			require.ErrorContains(t, err, test.message)
			assert.Empty(t, out)
		})
	}
}
