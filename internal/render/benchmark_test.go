package render_test

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/analyzer"
	"github.com/rumpl/structs/internal/render"
	"github.com/rumpl/structs/internal/testutil"
)

func BenchmarkHTMLPipeline(b *testing.B) {
	files := testutil.Project(b, 40, 20)
	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		data, err := analyzer.Analyze(files)
		require.NoError(b, err)

		require.NoError(b, render.Write(io.Discard, render.FormatHTML, data))
	}
}

func BenchmarkRenderLarge(b *testing.B) {
	data, err := analyzer.Analyze(testutil.Project(b, 40, 20))
	require.NoError(b, err)

	for _, format := range []string{render.FormatHTML, render.FormatJSON, render.FormatMarkdown} {
		b.Run(format, func(b *testing.B) {
			b.ReportAllocs()

			for range b.N {
				err := render.Write(io.Discard, format, data)
				require.NoError(b, err)
			}
		})
	}
}
