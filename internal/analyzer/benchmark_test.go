package analyzer_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/analyzer"
	"github.com/rumpl/structs/internal/testutil"
)

func BenchmarkAnalyzeLarge(b *testing.B) {
	files := testutil.Project(b, 40, 20)
	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_, err := analyzer.Analyze(files)
		require.NoError(b, err)
	}
}
