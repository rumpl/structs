package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/cli"
)

const (
	fieldCountQuery = "SELECT name, field_count FROM structs"
	querySubcommand = "query"
	formatFlag      = "--format"
)

func commandSource(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "types.go"), []byte("package sample\ntype Box struct { A, B int }\nfunc (Box) Get() {}\ntype Reader interface { Get() }"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "types_test.go"), []byte("package sample\ntype TestOnly struct{}"), 0o600))

	return dir
}

func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()

	command, err := cli.NewCommand()
	require.NoError(t, err)

	var out bytes.Buffer
	command.SetOut(&out)
	command.SetArgs(args)
	err = command.Execute()

	return out.String(), err
}
