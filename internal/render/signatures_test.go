package render

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMethodSignature(t *testing.T) {
	t.Parallel()

	for _, test := range []struct{ name, source, want string }{
		{"generic receiver", "func (b *Box[T, U]) Get(ctx context.Context) (T, error)", "func Get(ctx context.Context) (T, error)"},
		{"unnamed receiver", "func (Box) Close() error", "func Close() error"},
		{"interface", "Read([]byte) error", "Read([]byte) error"},
		{"multiline", "func (b *Box[T]) Many(\n first string,\n second int,\n) error", "func Many(first string, second int) error"},
	} {
		t.Run(test.name, func(t *testing.T) { t.Parallel(); assert.Equal(t, test.want, methodSignature(test.source)) })
	}
}

func TestSingleLineSignaturePreservesTagsAndSeparators(t *testing.T) {
	t.Parallel()

	source := "func Make(\n input struct {\n Name string `json:\"first  last\"`\n Count int\n },\n) error"
	assert.Equal(t, "func Make(input struct { Name string `json:\"first  last\"`; Count int; }) error", singleLineSignature(source))
}
