package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatementErrorIdentity(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		want      error
		statement string
	}{
		{statement: "/*", want: errUnterminatedComment},
		{statement: "SELECT 1; SELECT 2", want: errMultipleStatements},
		{statement: "SELECT 1;;", want: errMultipleStatements},
		{statement: "DELETE FROM types", want: errNotReadOnly},
		{statement: "SELECT '", want: errUnterminatedQuote},
		{statement: "", want: errEmptyQuery},
	} {
		t.Run(test.statement, func(t *testing.T) {
			t.Parallel()

			_, err := singleStatement(test.statement)
			assert.ErrorIs(t, err, test.want)
		})
	}
}
