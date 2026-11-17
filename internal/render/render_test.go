package render_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/render"
	"github.com/rumpl/structs/internal/report"
)

const (
	fixtureContext = "context"
	fixtureFile    = "types.go"
	fixtureStruct  = "Box"
)

func fixture() report.Report {
	return report.Report{Packages: []report.Package{{
		Name: "sample", Directory: "/tmp/<script>alert(1)</script>",
		Dependencies: []string{fixtureContext}, Imports: []report.Import{{Path: fixtureContext, Alias: "ctx", File: fixtureFile, Line: 2}},
		Structs: []report.Struct{{
			Name: fixtureStruct, File: fixtureFile, Line: 3, FieldCount: 1, MethodCount: 1, PublicMethodCount: 1,
			Fields:  []report.Field{{Name: "Ctx", Type: "ctx.Context", TypeName: "Context", TypePackage: fixtureContext, Tag: `json:"ctx"`, File: fixtureFile, Line: 4, Exported: true}},
			Methods: []report.Method{{Name: "Get", Declaration: "func (b Box[T]) Get() T", Exported: true}},
		}},
		Interfaces: []report.Interface{{Name: "Reader", File: fixtureFile, Line: 7, MethodCount: 1, Methods: []report.Method{{Name: "Read", Declaration: "Read([]byte) error"}}}},
	}}}
}

func renderReport(t *testing.T, format string, data report.Report) string {
	t.Helper()

	var out bytes.Buffer
	require.NoError(t, render.Write(&out, format, data))

	return out.String()
}

func TestJSONReportRoundTrip(t *testing.T) {
	t.Parallel()

	data := fixture()

	var decoded report.Report
	require.NoError(t, json.Unmarshal([]byte(renderReport(t, render.FormatJSON, data)), &decoded))
	assert.Equal(t, data, decoded)
}

func TestMarkdownReportDeclarations(t *testing.T) {
	t.Parallel()
	text := renderReport(t, render.FormatMarkdown, fixture())
	assert.Contains(t, text, "# Go type report")
	assert.Contains(t, text, "Fields: **1**")
	assert.Contains(t, text, "func (b Box[T]) Get() T")
	assert.Contains(t, text, "Read([]byte) error")
	assert.Contains(t, text, `json:"ctx"`)
}

var errWrite = errors.New("write failed")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errWrite }

func TestReportRejectsUnknownFormatWithoutOutput(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	require.ErrorContains(t, render.Write(&out, "csv", fixture()), "unsupported format")
	assert.Empty(t, out.String())
}

func TestReportPreservesWriterError(t *testing.T) {
	t.Parallel()

	for _, format := range []string{render.FormatJSON, render.FormatMarkdown, render.FormatHTML} {
		t.Run(format, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, render.Write(failingWriter{}, format, fixture()), errWrite)
		})
	}
}

func TestEmptyReport(t *testing.T) {
	t.Parallel()

	for _, format := range []string{render.FormatJSON, render.FormatMarkdown, render.FormatHTML} {
		t.Run(format, func(t *testing.T) {
			t.Parallel()

			text := renderReport(t, format, report.Report{Packages: []report.Package{}})
			if format == render.FormatJSON {
				assert.JSONEq(t, `{"packages":[]}`, text)

				return
			}

			assert.Contains(t, text, "No Go packages found.")
		})
	}
}
