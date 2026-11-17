package query_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/query"
	"github.com/rumpl/structs/internal/report"
)

func TestReferenceBatchesPreserveOrderAndOwners(t *testing.T) {
	t.Parallel()

	for _, count := range []int{0, 127, 128, 129, 261} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			t.Parallel()

			typ := report.NamedType{Name: "T", Kind: report.KindNamed}

			want := make([][]any, 0, count)
			for i := range count {
				name := fmt.Sprintf("Ref%d", i)
				typ.References = append(typ.References, report.TypeReference{Name: name, Relation: directRelation, Line: i + 1})
				want = append(want, []any{int64(i + 1), name, int64(i + 1), int64(1)})
			}

			data := report.Report{Packages: []report.Package{{Name: samplePackage, Types: []report.NamedType{typ}}}}
			result, err := query.Execute(context.Background(), data, "SELECT id,name,line,type_id FROM type_references ORDER BY id")
			require.NoError(t, err)
			assert.Equal(t, want, result.Rows)
		})
	}
}
