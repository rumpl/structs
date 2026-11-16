package analyzer

import (
	"go/scanner"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseErrorPreservesDiagnosticsAndCause(t *testing.T) {
	t.Parallel()

	diagnostics := scanner.ErrorList{
		&scanner.Error{Pos: token.Position{Filename: "broken.go", Line: 2}, Msg: "first error"},
		&scanner.Error{Pos: token.Position{Filename: "broken.go", Line: 3}, Msg: "second error"},
	}
	err := parseError("broken.go", diagnostics)

	var recovered scanner.ErrorList
	require.ErrorAs(t, err, &recovered)
	assert.Equal(t, diagnostics, recovered)

	for _, diagnostic := range diagnostics {
		assert.ErrorContains(t, err, diagnostic.Error())
	}
}
