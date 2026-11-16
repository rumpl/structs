package query_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/analyzer"
	"github.com/rumpl/structs/internal/query"
	"github.com/rumpl/structs/internal/testutil"
)

func BenchmarkQueryLarge(b *testing.B) {
	data, err := analyzer.Analyze(testutil.Project(b, 20, 20))
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		_, err := query.Execute(context.Background(), data, "SELECT count(*) FROM structs")
		require.NoError(b, err)
	}
}
