package render

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/report"
)

func TestParallelHTMLIsDeterministic(t *testing.T) {
	t.Parallel()

	data := report.Report{}
	for _, name := range []string{"alpha", "beta", "gamma"} {
		data.Packages = append(data.Packages, report.Package{
			Name: name, Directory: "/sample/" + name,
			Structs: []report.Struct{{Definition: report.Definition{Name: "Box", File: "/sample/" + name + "/types.go"}}},
		})
	}

	var sequential bytes.Buffer
	require.NoError(t, writeHTMLWithWorkers(&sequential, data, 1))

	for range 3 {
		var parallel bytes.Buffer
		require.NoError(t, writeHTMLWithWorkers(&parallel, data, 8))
		assert.Equal(t, sequential.String(), parallel.String())
	}
}
