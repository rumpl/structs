// Package render writes reports without performing source analysis or file I/O.
package render

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	texttemplate "text/template"

	"github.com/rumpl/structs/internal/report"
)

//go:embed report.md.tmpl
var markdownTemplate string

//go:embed report.html.tmpl
var htmlTemplate string

var errUnsupportedFormat = errors.New("unsupported format")

// ValidateFormat checks whether a format is supported.
func ValidateFormat(format string) error {
	switch format {
	case FormatJSON, FormatMarkdown, FormatHTML:
		return nil
	default:
		return fmt.Errorf("%w %q (choose json, markdown, or html)", errUnsupportedFormat, format)
	}
}

// Write renders a report. Output is buffered so template failures cannot emit
// partial reports; errors from the destination writer are returned to the caller.
func Write(w io.Writer, format string, data report.Report) error {
	if err := ValidateFormat(format); err != nil {
		return err
	}

	var (
		buffer bytes.Buffer
		err    error
	)

	markdown := texttemplate.Must(texttemplate.New(FormatMarkdown).Funcs(texttemplate.FuncMap{
		"code":    markdownCode,
		"heading": markdownHeading,
	}).Parse(markdownTemplate))

	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(&buffer)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(data)
	case FormatMarkdown:
		err = markdown.Execute(&buffer, data)
	case FormatHTML:
		err = writeHTML(&buffer, data)
	}

	if err != nil {
		return fmt.Errorf("render %s: %w", format, err)
	}

	if _, err := buffer.WriteTo(w); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	return nil
}

// Code spans use a delimiter longer than any run of backticks in the source.
func markdownCode(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	longest, run := 0, 0

	for _, char := range value {
		if char == '`' {
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
	}

	delimiter := strings.Repeat("`", longest+1)

	return delimiter + " " + value + " " + delimiter
}

func markdownHeading(value string) string {
	return strings.NewReplacer("\\", "\\\\", "`", "\\`", "*", "\\*", "_", "\\_", "[", "\\[", "]", "\\]", "<", "&lt;", ">", "&gt;", "&", "&amp;", "#", "\\#", "\n", " ", "\r", " ").Replace(value)
}
