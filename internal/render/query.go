package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/rumpl/structs/internal/query"
)

// WriteQuery renders arbitrary SQL columns independently of the type report UI.
// JSON uses ordered columns and row arrays so duplicate names are not lost.
func WriteQuery(w io.Writer, format string, result query.Result) error {
	err := ValidateFormat(format)
	if err != nil {
		return err
	}

	var out bytes.Buffer

	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(&out)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(result)
		if err != nil {
			return fmt.Errorf("encode query result: %w", err)
		}
	case FormatMarkdown:
		var table []byte

		table = appendMarkdownRow(table, result.Columns)

		separators := make([]string, len(result.Columns))
		for i := range separators {
			separators[i] = "---"
		}

		table = append(table, "| "+strings.Join(separators, " | ")+" |\n"...)

		for _, row := range result.Rows {
			cells := make([]string, len(row))
			for i, value := range row {
				cells[i] = queryCell(value)
			}

			table = appendMarkdownRow(table, cells)
		}

		out = *bytes.NewBuffer(table)
	case FormatHTML:
		queryHTML, err := template.New("query").Funcs(template.FuncMap{
			"cell": queryCell,
		}).Parse(strings.Replace(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Go query results</title>
<style>/* REPORT_STYLES */
.table-scroll { overflow-x: auto; background: white; border: 1px solid #dce2ec; border-radius: 12px; }
table { border-collapse: collapse; width: 100%; text-align: left; }
th, td { padding: 10px 16px; border-bottom: 1px solid #e2e7ef; vertical-align: top; }
th { background: #e9edf4; font-size: 12px; }
td { font-family: ui-monospace, monospace; white-space: pre-wrap; }
</style></head><body><header class="report-heading"><h1>Go query results</h1></header>
<p>{{len .Rows}} rows</p><div class="table-scroll"><table><thead><tr>{{range .Columns}}<th scope="col">{{.}}</th>{{end}}</tr></thead><tbody>
{{range .Rows}}<tr>{{range .}}<td>{{cell .}}</td>{{end}}</tr>{{end}}
</tbody></table></div></body></html>
`, "/* REPORT_STYLES */", reportCSS, 1))
		if err != nil {
			return fmt.Errorf("parse query HTML template: %w", err)
		}

		err = queryHTML.Execute(&out, result)
		if err != nil {
			return fmt.Errorf("render query HTML: %w", err)
		}
	}

	if _, err := out.WriteTo(w); err != nil {
		return fmt.Errorf("write query result: %w", err)
	}

	return nil
}

func queryCell(value any) string {
	if value == nil {
		return "NULL"
	}

	if blob, ok := value.([]byte); ok {
		return fmt.Sprintf("x'%x'", blob)
	}

	return fmt.Sprint(value)
}

func appendMarkdownRow(out []byte, cells []string) []byte {
	escape := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "|", "&#124;", "`", "&#96;", "*", "&#42;", "_", "&#95;", "\\", "&#92;", "[", "&#91;", "]", "&#93;", "\r\n", "<br>", "\n", "<br>", "\r", "<br>")

	out = append(out, '|', ' ')

	for i, cell := range cells {
		if i > 0 {
			out = append(out, ' ', '|', ' ')
		}

		out = append(out, escape.Replace(cell)...)
	}

	return append(out, ' ', '|', '\n')
}
