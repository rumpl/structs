package render_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/query"
	"github.com/rumpl/structs/internal/render"
)

func renderQuery(t *testing.T, format string, data query.Result) string {
	t.Helper()

	var out bytes.Buffer
	require.NoError(t, render.WriteQuery(&out, format, data))

	return out.String()
}

func TestQueryJSONPreservesDuplicateColumnsAndNulls(t *testing.T) {
	t.Parallel()

	data := query.Result{Columns: []string{"name", "name", "count"}, Rows: [][]any{{fixtureStruct, nil, int64(3)}}}
	assert.JSONEq(t, `{"columns":["name","name","count"],"rows":[["Box",null,3]]}`, renderQuery(t, render.FormatJSON, data))
}

func TestQueryMarkdownEscapesCells(t *testing.T) {
	t.Parallel()

	data := query.Result{Columns: []string{"value"}, Rows: [][]any{{"<script>|a\nb"}, {nil}}}
	assert.Equal(t, "| value |\n| --- |\n| &lt;script&gt;&#124;a<br>b |\n| NULL |\n", renderQuery(t, render.FormatMarkdown, data))
}

func TestQueryHTMLEscapesCellsAndHeaders(t *testing.T) {
	t.Parallel()

	data := query.Result{Columns: []string{"<header>"}, Rows: [][]any{{"<script>alert(1)</script>"}}}
	text := renderQuery(t, render.FormatHTML, data)
	assert.Contains(t, text, "&lt;header&gt;")
	assert.Contains(t, text, "&lt;script&gt;")
	assert.NotContains(t, text, "<script>")
}

func TestQueryPreservesWriterError(t *testing.T) {
	t.Parallel()

	for _, format := range []string{render.FormatJSON, render.FormatMarkdown, render.FormatHTML} {
		t.Run(format, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, render.WriteQuery(failingWriter{}, format, query.Result{}), errWrite)
		})
	}
}

func TestQueryRejectsUnknownFormatWithoutOutput(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	require.ErrorContains(t, render.WriteQuery(&out, "csv", query.Result{}), "unsupported format")
	assert.Empty(t, out.String())
}
