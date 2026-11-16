package render_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/render"
	"github.com/rumpl/structs/internal/report"
)

func TestHTMLAnchorsAreUniqueAndResolvable(t *testing.T) {
	t.Parallel()

	data := fixture()
	data.Packages = append(data.Packages, data.Packages[0])
	data.Packages[0].Structs = append(data.Packages[0].Structs, data.Packages[0].Structs[0])
	text := renderReport(t, render.FormatHTML, data)

	ids := map[string]bool{}
	for _, match := range regexp.MustCompile(`\bid="([^"]+)"`).FindAllStringSubmatch(text, -1) {
		assert.NotContains(t, ids, match[1], "duplicate anchor")
		ids[match[1]] = true
	}

	links := regexp.MustCompile(`href="#([^"]+)"`).FindAllStringSubmatch(text, -1)
	require.NotEmpty(t, links)

	for _, match := range links {
		assert.Contains(t, ids, match[1], "broken link")
	}

	for _, id := range []string{"struct-0-0", "struct-0-1", "struct-1-0", "interface-0-0", "interface-1-0"} {
		assert.Contains(t, text, fmt.Sprintf(`data-toc-entry=%q`, id))
	}
}

func TestHTMLInteractiveControls(t *testing.T) {
	t.Parallel()

	text := renderReport(t, render.FormatHTML, fixture())
	for _, markup := range []string{`type="search"`, `data-filter="struct"`, `data-filter="interface"`, `aria-live="polite"`, `aria-label="Table of contents"`, `data-type-section="struct"`, `data-type-section="interface"`} {
		assert.Contains(t, text, markup)
	}
}

func TestHTMLIsStandalone(t *testing.T) {
	t.Parallel()
	text := renderReport(t, render.FormatHTML, fixture())
	assert.NotContains(t, text, `<script src=`)
	assert.NotContains(t, text, `<link `)
	assert.Contains(t, text, "&lt;script&gt;")
	assert.NotContains(t, text, "ZgotmplZ")
}

func TestHTMLEscapesDeclarations(t *testing.T) {
	t.Parallel()

	data := fixture()
	data.Packages[0].Structs[0].Methods[0].Declaration = `func Evil() string // </script><script>alert("x")</script>`
	data.Packages[0].Structs[0].Name = `" onmouseover="alert(1)`
	text := renderReport(t, render.FormatHTML, data)
	assert.NotContains(t, text, `</script><script>alert`)
	assert.NotContains(t, text, `" onmouseover="alert(1)`)
	assert.Contains(t, text, `&#34; onmouseover=&#34;alert(1)`)
}

func TestHTMLPackageWithoutTypes(t *testing.T) {
	t.Parallel()
	text := renderReport(t, render.FormatHTML, report.Report{Packages: []report.Package{{Name: "main"}}})
	assert.NotContains(t, text, "No Go packages found.")
	assert.Contains(t, text, `id="structs"`)
	assert.Contains(t, text, `id="interfaces"`)
	assert.Contains(t, text, `<p class="empty-section">None.</p>`)
}
