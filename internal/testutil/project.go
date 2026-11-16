// Package testutil provides reproducible source fixtures for performance tests.
package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Project generates independent packages with structs, fields, interfaces,
// signatures, references, and method bodies that still require syntax parsing.
func Project(tb testing.TB, packages, types int) []string {
	tb.Helper()
	root := tb.TempDir()

	files := make([]string, 0, packages)
	for p := range packages {
		source := []byte("package sample\nimport \"context\"\n")

		for i := range types {
			source = fmt.Appendf(source, "type Type%d struct { Name string; Count int; Ctx context.Context; Children []*Type%d; Lookup map[string]*Type%d }\n", i, i, i)
			source = fmt.Appendf(source, "type Interface%d interface { Run(context.Context, ...string) (int,error) }\n", i)

			for m := range 4 {
				source = fmt.Appendf(source, "func (t *Type%d) Method%d(ctx context.Context, name string) (int,error) {\n", i, m)
				source = append(source, "total:=0\n"...)

				for range 20 {
					source = append(source, "if ctx.Err()!=nil { return 0,ctx.Err() }; total+=len(name)\n"...)
				}

				source = append(source, "return total,nil\n}\n"...)
			}
		}

		dir := filepath.Join(root, fmt.Sprintf("pkg%03d", p))

		err := os.MkdirAll(dir, 0o700)
		require.NoError(tb, err)

		file := filepath.Join(dir, "types.go")

		err = os.WriteFile(file, source, 0o600)
		require.NoError(tb, err)

		files = append(files, file)
	}

	return files
}
